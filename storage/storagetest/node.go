// Package storagetest runs an in-process DDC storage node for tests of code
// that stores content in DDC: the file, DAG and CNS gRPC services a real
// storage node exposes, served over an in-memory listener.
//
// It speaks the real wire contracts (fileapi, dagapi, cnsapi) so a client under
// test exercises its actual gRPC code, and it authorizes requests with the real
// token rule (auth.Authorizer) against an owner index the test controls, so a
// credential a storage node would refuse is refused here too.
//
// Two deliberate simplifications, both invisible to a writer: CIDs are computed
// as blake3 over the stored bytes for every content type (a real node uses a
// merkle root for raw pieces), and nothing replicates, because there is one
// node. A writer never computes a CID itself; it uses the ones the node returns.
package storagetest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cnsapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/dagapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/fileapi"
)

// Messages the real storage node answers with, reproduced so a client can be
// tested against the exact status it has to interpret. A missing CNS name is
// codes.NotFound with this message — and so is a missing BUCKET, with another
// message, which is why a client must not read every NotFound as "no name".
const (
	UnknownCnsRecordMessage = "unknown cns record"
	BucketNotFoundMessage   = "bucket not found"
)

// Node is one in-process storage node. The zero value is not usable; call Start.
type Node struct {
	fileapi.UnimplementedFileApiServer
	dagapi.UnimplementedDagApiServer

	authorizer auth.Authorizer

	mu      sync.Mutex
	buckets map[uint64]*bucket

	// Hooks let a test hold a request at a precise point. Each is called
	// outside the node's lock.
	//
	// AfterResolve runs once a CNS name has been READ and before the answer
	// is returned, so a test holding it knows the caller's view of the root
	// is already fixed — which is what makes two held resolves provably see
	// the same root.
	AfterResolve func(bucketID uint64, name string)
	// FailGetNode, when it returns a non-nil error for a CID, makes the DAG
	// read of that node fail with it — standing in for a node outage mid-read.
	FailGetNode func(bucketID uint64, cid string) error

	conn *grpc.ClientConn
}

type bucket struct {
	public bool
	pieces map[string][]byte // CID string -> raw bytes (a raw piece or a marshaled DAG node)
	names  map[string]*cnsapi.Record
	// puts counts every write that reached storage, so a test can assert a
	// refused request wrote nothing at all.
	puts int
}

// Start serves a node whose buckets are owned by owner, and returns it with a
// client connection already dialled. owner's key authorizes writes, exactly as
// a bucket owner's does on chain.
func Start(t testing.TB, owner ed25519.PublicKey, buckets ...uint64) *Node {
	t.Helper()
	owners := make(map[uint64]auth.LookupResult, len(buckets))
	n := &Node{buckets: make(map[uint64]*bucket, len(buckets))}
	for _, b := range buckets {
		owners[b] = auth.LookupResult{Owner: owner, HasBalance: true}
		n.buckets[b] = &bucket{public: true, pieces: map[string][]byte{}, names: map[string]*cnsapi.Record{}}
	}
	n.authorizer = auth.New(ownerIndex(owners))

	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	fileapi.RegisterFileApiServer(srv, n)
	dagapi.RegisterDagApiServer(srv, n)
	cnsapi.RegisterCnsApiServer(srv, cnsServer{n: n})
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///storagetest",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("storagetest: dial: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		srv.Stop()
	})
	n.conn = conn
	return n
}

// Conn is a client connection to this node.
func (n *Node) Conn() *grpc.ClientConn { return n.conn }

// SetPublic flips a bucket between public (anonymous GET allowed) and private.
func (n *Node) SetPublic(bucketID uint64, public bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.buckets[bucketID].public = public
}

// Writes reports how many writes (pieces, nodes, names) reached the bucket.
func (n *Node) Writes(bucketID uint64) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.buckets[bucketID].puts
}

// Name returns the CNS record stored under name, or nil.
func (n *Node) Name(bucketID uint64, name string) *cnsapi.Record {
	n.mu.Lock()
	defer n.mu.Unlock()
	r := n.buckets[bucketID].names[name]
	if r == nil {
		return nil
	}
	return proto.Clone(r).(*cnsapi.Record)
}

// Piece returns the raw bytes stored under a CID string, or nil.
func (n *Node) Piece(bucketID uint64, id string) []byte {
	n.mu.Lock()
	defer n.mu.Unlock()
	return bytes.Clone(n.buckets[bucketID].pieces[id])
}

// SeedNode stores a DAG node directly, bypassing authorization, and returns its
// CID. For arranging content that already exists before the code under test
// ever runs.
func (n *Node) SeedNode(bucketID uint64, node *dagapi.Node) string {
	raw, err := proto.Marshal(node)
	if err != nil {
		panic(err)
	}
	id := sum(cid.ContentTypeDagPb, raw)
	n.mu.Lock()
	defer n.mu.Unlock()
	n.buckets[bucketID].pieces[id.String()] = raw
	return id.String()
}

// SeedName points a CNS name at a CID directly, bypassing authorization.
func (n *Node) SeedName(bucketID uint64, name, id string) {
	c, err := cid.DecodeString(id)
	if err != nil {
		panic(err)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.buckets[bucketID].names[name] = &cnsapi.Record{Cid: c.Bytes(), Name: name}
}

// ReadNode returns a stored DAG node by CID, bypassing authorization.
func (n *Node) ReadNode(bucketID uint64, id string) *dagapi.Node {
	raw := n.Piece(bucketID, id)
	if raw == nil {
		return nil
	}
	var node dagapi.Node
	if err := proto.Unmarshal(raw, &node); err != nil {
		panic(err)
	}
	return &node
}

func sum(ct cid.ContentType, data []byte) *cid.Cid {
	id, err := cid.Sum(cid.DefaultParams(ct), cid.MhTypeBlake3, data)
	if err != nil {
		panic(err)
	}
	return id
}

func tokenFrom(ctx context.Context) string {
	if v := metadata.ValueFromIncomingContext(ctx, "token"); len(v) > 0 {
		return v[0]
	}
	return ""
}

// lookup returns the bucket or the storage node's own NotFound.
func (n *Node) lookup(bucketID uint64) (*bucket, error) {
	b, ok := n.buckets[bucketID]
	if !ok {
		return nil, status.Error(codes.NotFound, BucketNotFoundMessage)
	}
	return b, nil
}

// allow is the storage node's gate: a public bucket serves GET to anyone, and
// everything else needs a token the shared authorizer accepts for this bucket
// and operation.
func (n *Node) allow(ctx context.Context, bucketID uint64, op pb.Operation) error {
	n.mu.Lock()
	b, err := n.lookup(bucketID)
	public := b != nil && b.public
	n.mu.Unlock()
	if err != nil {
		return err
	}
	if public && op == pb.Operation_GET {
		return nil
	}
	if _, err := n.authorizer.Authorize(ctx, tokenFrom(ctx), op, bucketID); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}

// PutRawPiece stores one raw piece. The first message carries the metadata,
// the rest carry content; a body longer than the announced size is refused as
// the real node refuses it.
func (n *Node) PutRawPiece(stream fileapi.FileApi_PutRawPieceServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	meta := first.GetMetadata()
	if meta == nil {
		return status.Error(codes.InvalidArgument, "first message must carry metadata")
	}
	if err := n.allow(stream.Context(), meta.GetBucketId(), pb.Operation_PUT); err != nil {
		return err
	}
	var buf bytes.Buffer
	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		buf.Write(msg.GetContent().GetData())
		if uint64(buf.Len()) > meta.GetSize() {
			return status.Error(codes.InvalidArgument, "piece is larger than announced")
		}
	}
	if buf.Len() == 0 || uint64(buf.Len()) != meta.GetSize() {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid piece: got %d bytes, announced %d", buf.Len(), meta.GetSize()))
	}
	id := sum(cid.ContentTypeRaw, buf.Bytes())
	n.mu.Lock()
	b := n.buckets[meta.GetBucketId()]
	b.pieces[id.String()] = buf.Bytes()
	b.puts++
	n.mu.Unlock()
	return stream.SendAndClose(&fileapi.PutRawPieceResponse{Cid: id.Bytes()})
}

// Put stores a DAG node and returns its CID.
func (n *Node) Put(ctx context.Context, req *dagapi.PutRequest) (*dagapi.PutResponse, error) {
	if err := n.allow(ctx, req.GetBucketId(), pb.Operation_PUT); err != nil {
		return nil, err
	}
	raw, err := proto.Marshal(req.GetNode())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id := sum(cid.ContentTypeDagPb, raw)
	n.mu.Lock()
	b := n.buckets[req.GetBucketId()]
	b.pieces[id.String()] = raw
	b.puts++
	n.mu.Unlock()
	return &dagapi.PutResponse{Cid: id.Bytes()}, nil
}

// Get reads a DAG node by CID (path traversal is not needed by any writer).
func (n *Node) Get(ctx context.Context, req *dagapi.GetRequest) (*dagapi.GetResponse, error) {
	if err := n.allow(ctx, req.GetBucketId(), pb.Operation_GET); err != nil {
		return nil, err
	}
	id := cid.ToBase32(req.GetCid())
	if n.FailGetNode != nil {
		if err := n.FailGetNode(req.GetBucketId(), id); err != nil {
			return nil, err
		}
	}
	raw := n.Piece(req.GetBucketId(), id)
	if raw == nil {
		return nil, status.Error(codes.NotFound, "no such node")
	}
	var node dagapi.Node
	if err := proto.Unmarshal(raw, &node); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dagapi.GetResponse{Node: &node}, nil
}

// cnsServer is the CNS half. It has to be a separate type: CnsApi and DagApi
// both name their methods Put and Get.
type cnsServer struct {
	cnsapi.UnimplementedCnsApiServer
	n *Node
}

// Put stores a CNS record after checking it is signed by the key it names,
// which is the only thing the real node checks about the record itself.
func (s cnsServer) Put(ctx context.Context, req *cnsapi.PutRequest) (*cnsapi.PutResponse, error) {
	rec := req.GetRecord()
	if rec == nil || rec.GetSignature() == nil || len(rec.GetCid()) == 0 || rec.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "record, signature, cid and name are required")
	}
	// As a storage node does: the signature must verify, under the key it
	// names, over the record marshaled without its signature (ed25519 or
	// sr25519, raw or polkadot.js-wrapped).
	if ok, err := auth.VerifySignature(proto.Clone(rec).(*cnsapi.Record)); err != nil || !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid signature")
	}
	if err := s.n.allow(ctx, req.GetBucketId(), pb.Operation_PUT); err != nil {
		return nil, err
	}
	s.n.mu.Lock()
	b := s.n.buckets[req.GetBucketId()]
	b.names[rec.GetName()] = proto.Clone(rec).(*cnsapi.Record)
	b.puts++
	s.n.mu.Unlock()
	return &cnsapi.PutResponse{}, nil
}

// Get resolves a CNS name.
func (s cnsServer) Get(ctx context.Context, req *cnsapi.GetRequest) (*cnsapi.GetResponse, error) {
	s.n.mu.Lock()
	b, err := s.n.lookup(req.GetBucketId())
	var rec *cnsapi.Record
	if b != nil {
		rec = b.names[req.GetName()]
	}
	s.n.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if s.n.AfterResolve != nil {
		s.n.AfterResolve(req.GetBucketId(), req.GetName())
	}
	// The name lookup comes before authorization, as on the real node
	// (internal/storage/cns/api.go), so a caller without a valid token also
	// learns whether a name exists.
	if rec == nil {
		return nil, status.Error(codes.NotFound, UnknownCnsRecordMessage)
	}
	if err := s.n.allow(ctx, req.GetBucketId(), pb.Operation_GET); err != nil {
		return nil, err
	}
	return &cnsapi.GetResponse{Record: proto.Clone(rec).(*cnsapi.Record)}, nil
}

type ownerIndex map[uint64]auth.LookupResult

func (o ownerIndex) Lookup(_ context.Context, bucketID uint64) (auth.LookupResult, error) {
	return o[bucketID], nil
}
