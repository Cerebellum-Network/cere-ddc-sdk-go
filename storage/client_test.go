package storage_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/authtest"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cnsapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/dagapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/storagetest"
)

const testBucket uint64 = 4242

// writeToken is what a storage node demands for a write: signed by the bucket
// owner, naming the bucket, granting PUT and GET.
func writeToken(t *testing.T, owner authtest.Keypair, bucket uint64) string {
	t.Helper()
	return authtest.Encode58(t, authtest.BuildChain(t, authtest.Level{
		Signer:    owner,
		Ops:       []pb.Operation{pb.Operation_PUT, pb.Operation_GET},
		BucketID:  bucket,
		ExpiresIn: time.Hour,
	}))
}

func newClient(t *testing.T, conn grpc.ClientConnInterface, token string) *storage.Client {
	t.Helper()
	return storage.NewClient(conn,
		storage.WithToken(token),
		storage.WithRecordSigner(auth.NewEd25519Signer(authtest.NewKey(t).Priv)))
}

// The whole surface end to end over the storage node's gRPC contracts: a raw
// piece, a DAG node linking it, a CNS name over the node, then resolving the
// name and reading the node back.
func TestClientWritesANodeAndANameAndReadsThemBackByName(t *testing.T) {
	ctx := context.Background()
	owner := authtest.NewKey(t)
	node := storagetest.Start(t, owner.Pub, testBucket)
	client := newClient(t, node.Conn(), writeToken(t, owner, testBucket))

	body := []byte("export default {}\n")
	pieceCID, err := client.PutPiece(ctx, testBucket, bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.Equal(t, body, node.Piece(testBucket, pieceCID))

	pieceBytes, err := storage.CIDBytes(pieceCID)
	require.NoError(t, err)
	want := &dagapi.Node{
		Links: []*dagapi.Link{{Cid: pieceBytes, Name: "bundle.js", Size: uint64(len(body))}},
		Tags:  []*dagapi.Tag{{Key: "k", Value: "v"}},
	}
	nodeCID, err := client.PutNode(ctx, testBucket, want)
	require.NoError(t, err)
	require.NoError(t, client.PutName(ctx, testBucket, "agents", nodeCID))

	resolved, err := client.ResolveName(ctx, testBucket, "agents")
	require.NoError(t, err)
	assert.Equal(t, nodeCID, resolved)
	got, err := client.GetNode(ctx, testBucket, resolved)
	require.NoError(t, err)
	assert.True(t, proto.Equal(want, got), "node read back by name differs")
	assert.Equal(t, "bundle.js", got.SearchLinkByName("bundle.js").GetName())
}

// An unknown name is the only resolve failure that means "no such name";
// a missing bucket is NotFound too, and must stay a plain error.
func TestResolveNameDistinguishesAMissingNameFromAMissingBucket(t *testing.T) {
	ctx := context.Background()
	owner := authtest.NewKey(t)
	node := storagetest.Start(t, owner.Pub, testBucket)
	client := newClient(t, node.Conn(), writeToken(t, owner, testBucket))

	_, err := client.ResolveName(ctx, testBucket, "agents")
	require.ErrorIs(t, err, storage.ErrNameNotFound)

	_, err = client.ResolveName(ctx, testBucket+1, "agents")
	require.Error(t, err)
	require.NotErrorIs(t, err, storage.ErrNameNotFound)
}

// A token signed by somebody who does not own the bucket writes nothing.
func TestAWriteWithAForeignTokenIsRefused(t *testing.T) {
	ctx := context.Background()
	owner, stranger := authtest.NewKey(t), authtest.NewKey(t)
	node := storagetest.Start(t, owner.Pub, testBucket)
	client := newClient(t, node.Conn(), writeToken(t, stranger, testBucket))

	_, err := client.PutNode(ctx, testBucket, &dagapi.Node{})
	require.Error(t, err)
	assert.Zero(t, node.Writes(testBucket))
}

// A storage node verifies a CNS record's signature over the record marshaled
// without its signature, signed as-is. Pinned with a fixed key and record
// against a hand-computed ed25519 signature.
func TestPutNameSignsTheRawRecordBytes(t *testing.T) {
	seed, _ := hex.DecodeString("0101010101010101010101010101010101010101010101010101010101010101")
	key := ed25519.NewKeyFromSeed(seed)
	capture := &capturingCNS{}
	client := storage.NewClient(capture.serve(t),
		storage.WithToken("token"), storage.WithRecordSigner(auth.NewEd25519Signer(key)))

	raw := sumCID(t, []byte("record"))
	cidString, err := storage.CIDString(raw)
	require.NoError(t, err)
	require.NoError(t, client.PutName(context.Background(), testBucket, "agents", cidString))

	rec := capture.put.GetRecord()
	require.NotNil(t, rec)
	unsigned, err := proto.Marshal(&cnsapi.Record{Cid: raw, Name: "agents"})
	require.NoError(t, err)
	sig := rec.GetSignature()
	assert.Equal(t, pb.Signature_ED_25519, sig.GetAlgorithm())
	assert.Equal(t, []byte(key.Public().(ed25519.PublicKey)), sig.GetSigner())
	assert.Equal(t, ed25519.Sign(key, unsigned), sig.GetValue(), "signature over proto.Marshal of the unsigned record")
	assert.Equal(t, testBucket, capture.put.GetBucketId())
	assert.Equal(t, []string{"token"}, capture.tokens)
}

// sr25519 record signers work too, as on a storage node.
func TestPutNameWithAnSr25519Signer(t *testing.T) {
	ctx := context.Background()
	owner := authtest.NewKey(t)
	node := storagetest.Start(t, owner.Pub, testBucket)
	signer, err := auth.NewSr25519Signer([32]byte{7})
	require.NoError(t, err)
	client := storage.NewClient(node.Conn(), storage.WithToken(writeToken(t, owner, testBucket)), storage.WithRecordSigner(signer))

	nodeCID, err := client.PutNode(ctx, testBucket, &dagapi.Node{Data: []byte("x")})
	require.NoError(t, err)
	require.NoError(t, client.PutName(ctx, testBucket, "agents", nodeCID))
	assert.Equal(t, pb.Signature_SR_25519, node.Name(testBucket, "agents").GetSignature().GetAlgorithm())
}

func TestPutPieceStreamsMetadataThenOneMiBChunks(t *testing.T) {
	capture := &capturingFile{}
	client := storage.NewClient(capture.serve(t), storage.WithToken("token"))
	body := bytes.Repeat([]byte{0xab}, storage.PieceChunkSize*2+512)

	_, err := client.PutPiece(context.Background(), testBucket, bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	require.Len(t, capture.messages, 4)
	meta := capture.messages[0].GetMetadata()
	require.NotNil(t, meta)
	assert.Equal(t, testBucket, meta.GetBucketId())
	assert.Equal(t, uint64(len(body)), meta.GetSize())
	assert.Len(t, capture.messages[1].GetContent().GetData(), storage.PieceChunkSize)
	assert.Len(t, capture.messages[2].GetContent().GetData(), storage.PieceChunkSize)
	assert.Len(t, capture.messages[3].GetContent().GetData(), 512)
	assert.Equal(t, []string{"token"}, capture.tokens)
}

func TestPutPieceChecksTheSize(t *testing.T) {
	capture := &capturingFile{}
	client := storage.NewClient(capture.serve(t), storage.WithToken("token"))
	ctx := context.Background()

	_, err := client.PutPiece(ctx, testBucket, bytes.NewReader([]byte("abc")), 4)
	require.ErrorContains(t, err, "content is 3 bytes, announced 4")
	_, err = client.PutPiece(ctx, testBucket, bytes.NewReader([]byte("abcde")), 4)
	require.ErrorContains(t, err, "longer than the announced 4 bytes")
	_, err = client.PutPiece(ctx, testBucket, bytes.NewReader(nil), 0)
	require.ErrorContains(t, err, "size must be positive")
	_, err = client.PutPiece(ctx, testBucket, bytes.NewReader(nil), storage.MaxRawPieceSize+1)
	require.ErrorContains(t, err, "raw piece limit")
}

// A reader that fails mid-body surfaces the reader's error, not the node's
// answer to the truncated stream.
func TestPutPieceSurfacesTheReadersError(t *testing.T) {
	capture := &capturingFile{}
	client := storage.NewClient(capture.serve(t), storage.WithToken("token"))
	boom := errors.New("disk on fire")

	_, err := client.PutPiece(context.Background(), testBucket, io.MultiReader(bytes.NewReader([]byte("ab")), errReader{boom}), 10)
	require.ErrorIs(t, err, boom)
}

func TestCallsWithoutATokenOrSignerFailFast(t *testing.T) {
	ctx := context.Background()
	owner := authtest.NewKey(t)
	node := storagetest.Start(t, owner.Pub, testBucket)

	_, err := storage.NewClient(node.Conn()).PutNode(ctx, testBucket, &dagapi.Node{})
	require.ErrorIs(t, err, storage.ErrNoToken)

	nodeCID, err := storage.NewClient(node.Conn(), storage.WithToken(writeToken(t, owner, testBucket))).
		PutNode(ctx, testBucket, &dagapi.Node{Data: []byte("x")})
	require.NoError(t, err)
	err = storage.NewClient(node.Conn(), storage.WithToken(writeToken(t, owner, testBucket))).
		PutName(ctx, testBucket, "agents", nodeCID)
	require.ErrorIs(t, err, storage.ErrNoRecordSigner)
}

func TestWithTokenSourceIsAskedPerCall(t *testing.T) {
	capture := &capturingFile{}
	calls := 0
	client := storage.NewClient(capture.serve(t), storage.WithTokenSource(func(context.Context) (string, error) {
		calls++
		return "token-" + string(rune('0'+calls)), nil
	}))
	for range 2 {
		_, err := client.PutPiece(context.Background(), testBucket, bytes.NewReader([]byte("a")), 1)
		require.NoError(t, err)
	}
	assert.Equal(t, []string{"token-1", "token-2"}, capture.tokens)
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
