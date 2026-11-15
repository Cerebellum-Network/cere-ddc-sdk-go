// Package authtest mints DDC access-token chains and stubs the bucket-owner
// index, for tests of anything that authorizes one.
//
// The fixtures are shared because the forgeries are: a self-signed token naming
// somebody else's bucket, a chain whose leaf outlives its parent, a token with no
// expiry, a tampered payload. Every component that enforces the one rule has to
// reject all of them, and building each attack once means a new component
// inherits the whole set rather than re-deriving the interesting half.
package authtest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

// Keypair is one signing identity: a bucket owner, a delegating service, or an
// attacker.
type Keypair struct {
	Pub  ed25519.PublicKey
	Priv ed25519.PrivateKey
}

// NewKey generates a signing identity.
func NewKey(t testing.TB) Keypair {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "generate signing key")
	return Keypair{Pub: pub, Priv: priv}
}

// Level describes one link of a chain to mint.
type Level struct {
	Signer      Keypair
	Subject     ed25519.PublicKey
	CanDelegate bool
	Ops         []pb.Operation
	BucketID    uint64
	ExpiresIn   time.Duration
}

// BuildChain assembles and signs a chain. levels[0] is the root; the returned
// token is the leaf, which is what a caller presents.
//
// Operation sets must be identical at every level: the chain's attenuation check
// is inverted (see the auth package doc), so a parent granting a
// superset of the child's operations is rejected. Every real minter has to obey
// this, and so does every fixture.
func BuildChain(t testing.TB, levels ...Level) *pb.AuthToken {
	t.Helper()
	var prev *pb.AuthToken
	for _, level := range levels {
		canDelegate := level.CanDelegate
		expiresAt := time.Now().Add(level.ExpiresIn).Unix()
		payload := &pb.Payload{
			Prev:        prev,
			Subject:     level.Subject,
			CanDelegate: &canDelegate,
			Operations:  level.Ops,
			ExpiresAt:   &expiresAt,
		}
		if level.BucketID != 0 {
			bucketID := level.BucketID
			payload.BucketId = &bucketID
		}
		prev = Sign(t, level.Signer, payload)
	}
	return prev
}

// Sign signs a payload as-is, for the fixtures BuildChain cannot express — a
// token with no expiry at all, say. The signature covers the marshaled token with
// Signature unset, which is the envelope auth.VerifySignature reconstructs.
func Sign(t testing.TB, signer Keypair, payload *pb.Payload) *pb.AuthToken {
	t.Helper()
	token := &pb.AuthToken{Payload: payload}
	raw, err := proto.Marshal(token)
	require.NoError(t, err, "marshal token for signing")
	token.Signature = &pb.Signature{
		Algorithm: pb.Signature_ED_25519,
		Signer:    signer.Pub,
		Value:     ed25519.Sign(signer.Priv, raw),
	}
	return token
}

// OwnerToken is the developer-direct shape: one level, no subject, signed by
// signer, granting COMPUTE on bucket for an hour.
//
// Signed by the bucket owner it is the honest credential; signed by anybody else
// it is the forgery the one rule exists to stop — well-formed, correctly signed,
// unexpired, and worthless because its root issuer owns nothing.
func OwnerToken(t testing.TB, signer Keypair, bucket uint64) *pb.AuthToken {
	t.Helper()
	return BuildChain(t, Level{
		Signer:    signer,
		Ops:       []pb.Operation{pb.Operation_COMPUTE},
		BucketID:  bucket,
		ExpiresIn: time.Hour,
	})
}

// DelegatedToken is the two-level agent shape: owner grants COMPUTE on bucket to
// delegate, delegate signs a leaf naming subject.
func DelegatedToken(t testing.TB, owner, delegate Keypair, subject ed25519.PublicKey, bucket uint64) *pb.AuthToken {
	t.Helper()
	return BuildChain(t,
		Level{
			Signer:      owner,
			Subject:     delegate.Pub,
			CanDelegate: true,
			Ops:         []pb.Operation{pb.Operation_COMPUTE},
			BucketID:    bucket,
			ExpiresIn:   2 * time.Hour,
		},
		Level{
			Signer:    delegate,
			Subject:   subject,
			Ops:       []pb.Operation{pb.Operation_COMPUTE},
			BucketID:  bucket,
			ExpiresIn: time.Hour,
		},
	)
}

// Encode58 encodes a token in the DDC-native encoding.
func Encode58(t testing.TB, token *pb.AuthToken) string {
	t.Helper()
	return base58.Encode(Marshal(t, token))
}

// Encode64 encodes a token in standard base64, which DecodeToken also accepts.
func Encode64(t testing.TB, token *pb.AuthToken) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(Marshal(t, token))
}

// Marshal returns a token's protobuf bytes — what a component forwarding a
// credential onward has to reproduce byte-for-byte.
func Marshal(t testing.TB, token *pb.AuthToken) []byte {
	t.Helper()
	raw, err := proto.Marshal(token)
	require.NoError(t, err, "marshal token")
	return raw
}

// Index is a canned auth.BucketOwnerIndex. A bucket absent from Owners has no
// owner and no balance, which denies.
type Index struct {
	Owners map[uint64]auth.LookupResult

	// Err makes every lookup fail, standing in for an indexer outage. The
	// distinction that matters: an outage must deny, and must be distinguishable
	// from a forgery.
	Err error
}

var _ auth.BucketOwnerIndex = (*Index)(nil)

func (i *Index) Lookup(_ context.Context, bucketID uint64) (auth.LookupResult, error) {
	if i.Err != nil {
		return auth.LookupResult{}, i.Err
	}
	return i.Owners[bucketID], nil
}

// OwnedBy returns an index in which owner owns every listed bucket, with
// balance.
func OwnedBy(owner Keypair, buckets ...uint64) *Index {
	owners := make(map[uint64]auth.LookupResult, len(buckets))
	for _, bucket := range buckets {
		owners[bucket] = auth.LookupResult{Owner: owner.Pub, HasBalance: true}
	}
	return &Index{Owners: owners}
}

// Unfunded returns an index in which owner owns bucket but cannot pay.
func Unfunded(owner Keypair, bucket uint64) *Index {
	return &Index{Owners: map[uint64]auth.LookupResult{
		bucket: {Owner: owner.Pub, HasBalance: false},
	}}
}

// Down returns an index that cannot answer.
func Down(reason string) *Index {
	return &Index{Err: Error(reason)}
}

// Error is a plain error, standing in for anything that is not a typed authz
// rejection — those must deny, not pass.
type Error string

func (e Error) Error() string { return string(e) }

// RecordingAuthorizer is a canned auth.Authorizer that records what it was
// asked. Use it to assert WHICH bucket a middleware chose to authorize against,
// which a real Authorizer cannot report.
type RecordingAuthorizer struct {
	Verified *auth.Verified
	Err      error

	Calls     int
	GotOp     pb.Operation
	GotBucket uint64
	GotToken  string
}

var _ auth.Authorizer = (*RecordingAuthorizer)(nil)

func (r *RecordingAuthorizer) Authorize(
	_ context.Context, encodedToken string, op pb.Operation, bucketID uint64,
) (*auth.Verified, error) {
	r.Calls++
	r.GotToken, r.GotOp, r.GotBucket = encodedToken, op, bucketID
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Verified, nil
}

// Allowing returns a RecordingAuthorizer that accepts everything and reports
// bucketID as the payer.
func Allowing(bucketID uint64) *RecordingAuthorizer {
	return &RecordingAuthorizer{Verified: &auth.Verified{BucketID: bucketID}}
}

// Rejecting returns a RecordingAuthorizer that fails with a typed rejection of
// the given kind.
func Rejecting(kind auth.Kind, err error) *RecordingAuthorizer {
	return &RecordingAuthorizer{Err: &auth.Error{Kind: kind, Err: err}}
}
