package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cnsapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/dagapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/fileapi"
)

const (
	// TokenMetadataKey is the gRPC metadata key a storage node reads the DDC
	// access token from.
	TokenMetadataKey = "token"

	// PieceChunkSize is the content message size PutPiece streams in: the
	// storage node's own stream chunk size.
	PieceChunkSize = 1 << 20

	// MaxRawPieceSize is the largest raw piece a storage node accepts. Larger
	// content needs a multipart piece.
	MaxRawPieceSize = 128 << 20

	// cnsNotFoundMessage is the storage node's message for an unknown CNS
	// name. The node answers codes.NotFound for a missing bucket too, so the
	// code alone does not say the name is absent; the message does.
	cnsNotFoundMessage = "unknown cns record"
)

var (
	// ErrNameNotFound is returned by ResolveName when the bucket has no CNS
	// record under the name. No other failure means the name is absent: an
	// unreachable node, a missing bucket or a refused token says nothing about
	// what the name points at.
	ErrNameNotFound = errors.New("cns name not found")

	// ErrNoRecordSigner is returned by PutName when the client has no signer.
	ErrNoRecordSigner = errors.New("no cns record signer configured")

	// ErrNoToken is returned by every call when the client has no token.
	ErrNoToken = errors.New("no access token configured")
)

// Client stores and reads content on one DDC storage node over its gRPC API:
// raw pieces, DAG nodes, and CNS names that point at them. CIDs cross the
// client as base32 strings, the form DDC CDN URLs and manifests use.
type Client struct {
	file   fileapi.FileApiClient
	dag    dagapi.DagApiClient
	cns    cnsapi.CnsApiClient
	token  func(context.Context) (string, error)
	signer auth.Signer
}

// Option configures a Client.
type Option func(*Client)

// WithToken authorizes every call with one encoded DDC access token (see
// auth.EncodeToken). The token must grant the operations the calls need on
// their bucket: PUT for writes, GET for reads.
func WithToken(encoded string) Option {
	return func(c *Client) {
		c.token = func(context.Context) (string, error) { return encoded, nil }
	}
}

// WithTokenSource authorizes each call with the token source returns at call
// time, for tokens that are refreshed before they expire.
func WithTokenSource(source func(context.Context) (string, error)) Option {
	return func(c *Client) { c.token = source }
}

// WithRecordSigner sets the key that signs CNS records in PutName. The storage
// node checks that a record verifies under the key it names; whether the caller
// may point the name is a question for the access token.
func WithRecordSigner(signer auth.Signer) Option {
	return func(c *Client) { c.signer = signer }
}

// NewClient returns a client for the storage node behind conn.
func NewClient(conn grpc.ClientConnInterface, opts ...Option) *Client {
	c := &Client{
		file: fileapi.NewFileApiClient(conn),
		dag:  dagapi.NewDagApiClient(conn),
		cns:  cnsapi.NewCnsApiClient(conn),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) withToken(ctx context.Context) (context.Context, error) {
	if c.token == nil {
		return nil, ErrNoToken
	}
	token, err := c.token(ctx)
	if err != nil {
		return nil, fmt.Errorf("access token: %w", err)
	}
	return metadata.AppendToOutgoingContext(ctx, TokenMetadataKey, token), nil
}

// PutPiece stores size bytes read from content as one raw piece and returns its
// CID. It streams a metadata message announcing the bucket and size, then the
// content in PieceChunkSize messages. The node rejects a body that disagrees
// with the announced size, so size must be exact; PutPiece checks it before the
// node does.
//
// When PutPiece gives up part-way (the content is shorter or longer than size,
// or reading it fails) it cancels the stream rather than closing it, so the node
// never sees a complete upload and stores nothing.
func (c *Client) PutPiece(ctx context.Context, bucketID uint64, content io.Reader, size int64) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("put piece: size must be positive, got %d", size)
	}
	if size > MaxRawPieceSize {
		return "", fmt.Errorf("put piece: %d bytes exceeds the %d-byte raw piece limit", size, MaxRawPieceSize)
	}
	ctx, err := c.withToken(ctx)
	if err != nil {
		return "", fmt.Errorf("put piece: %w", err)
	}
	// Cancelling this context is how an abandoned upload is aborted, and it
	// releases the stream however PutPiece returns.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := c.file.PutRawPiece(ctx)
	if err != nil {
		return "", fmt.Errorf("put piece: open stream: %w", err)
	}
	if err := stream.Send(&fileapi.PutRawPieceRequest{Body: &fileapi.PutRawPieceRequest_Metadata_{
		Metadata: &fileapi.PutRawPieceRequest_Metadata{BucketId: bucketID, Size: uint64(size)},
	}}); err != nil {
		return "", fmt.Errorf("put piece: send metadata: %w", err)
	}
	var sent int64
	for {
		// A fresh buffer per message: gRPC may still read a sent message after
		// Send returns. Read one byte past the announced size to catch content
		// that is too long before its last chunk goes out.
		buf := make([]byte, min(PieceChunkSize, size-sent+1))
		n, rerr := io.ReadFull(content, buf)
		if n > 0 {
			sent += int64(n)
			if sent > size {
				return "", fmt.Errorf("put piece: content is longer than the announced %d bytes", size)
			}
			if err := stream.Send(&fileapi.PutRawPieceRequest{Body: &fileapi.PutRawPieceRequest_Content_{
				Content: &fileapi.PutRawPieceRequest_Content{Data: buf[:n]},
			}}); err != nil {
				return "", fmt.Errorf("put piece: send content: %w", err)
			}
		}
		if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
			break
		}
		if rerr != nil {
			return "", fmt.Errorf("put piece: read content: %w", rerr)
		}
	}
	if sent != size {
		return "", fmt.Errorf("put piece: content is %d bytes, announced %d", sent, size)
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", fmt.Errorf("put piece: %w", err)
	}
	return CIDString(resp.GetCid())
}

// PutNode stores a DAG node and returns the CID the node computed for it.
func (c *Client) PutNode(ctx context.Context, bucketID uint64, node *dagapi.Node) (string, error) {
	ctx, err := c.withToken(ctx)
	if err != nil {
		return "", fmt.Errorf("put node: %w", err)
	}
	resp, err := c.dag.Put(ctx, &dagapi.PutRequest{BucketId: bucketID, Node: node})
	if err != nil {
		return "", fmt.Errorf("put node: %w", err)
	}
	return CIDString(resp.GetCid())
}

// GetNode reads a DAG node by CID.
func (c *Client) GetNode(ctx context.Context, bucketID uint64, id string) (*dagapi.Node, error) {
	raw, err := CIDBytes(id)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	ctx, err = c.withToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	resp, err := c.dag.Get(ctx, &dagapi.GetRequest{BucketId: bucketID, Cid: raw})
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", id, err)
	}
	if resp.GetNode() == nil {
		return nil, fmt.Errorf("get node %s: empty response", id)
	}
	return resp.GetNode(), nil
}

// PutName points a CNS name in the bucket at a CID, with a record signed by the
// client's record signer. The signature covers the record marshaled without its
// signature, signed as-is, which is what the storage node verifies.
func (c *Client) PutName(ctx context.Context, bucketID uint64, name, id string) error {
	if c.signer == nil {
		return fmt.Errorf("put name %q: %w", name, ErrNoRecordSigner)
	}
	raw, err := CIDBytes(id)
	if err != nil {
		return fmt.Errorf("put name %q: %w", name, err)
	}
	record := &cnsapi.Record{Cid: raw, Name: name}
	if _, err := auth.SignMessage(c.signer, record); err != nil {
		return fmt.Errorf("put name %q: sign record: %w", name, err)
	}
	ctx, err = c.withToken(ctx)
	if err != nil {
		return fmt.Errorf("put name %q: %w", name, err)
	}
	if _, err := c.cns.Put(ctx, &cnsapi.PutRequest{BucketId: bucketID, Record: record}); err != nil {
		return fmt.Errorf("put name %q: %w", name, err)
	}
	return nil
}

// ResolveName returns the CID a CNS name in the bucket points at. It returns an
// error wrapping ErrNameNotFound only when the node says the bucket has no such
// name.
func (c *Client) ResolveName(ctx context.Context, bucketID uint64, name string) (string, error) {
	ctx, err := c.withToken(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve name %q: %w", name, err)
	}
	resp, err := c.cns.Get(ctx, &cnsapi.GetRequest{BucketId: bucketID, Name: name})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound && st.Message() == cnsNotFoundMessage {
			return "", fmt.Errorf("%w: %q in bucket %d", ErrNameNotFound, name, bucketID)
		}
		return "", fmt.Errorf("resolve name %q: %w", name, err)
	}
	if len(resp.GetRecord().GetCid()) == 0 {
		return "", fmt.Errorf("resolve name %q: record carries no cid", name)
	}
	return CIDString(resp.GetRecord().GetCid())
}

// CIDBytes decodes a base32 CID string into the binary form the gRPC API
// carries.
func CIDBytes(s string) ([]byte, error) {
	c, err := cid.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return c.Bytes(), nil
}

// CIDString encodes a binary CID from the gRPC API as a base32 string.
func CIDString(raw []byte) (string, error) {
	c, err := cid.Decode(raw)
	if err != nil {
		return "", fmt.Errorf("undecodable cid: %w", err)
	}
	return c.String(), nil
}
