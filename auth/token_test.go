package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

const testBucket = uint64(1337)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type keypair struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func newKey(t *testing.T) keypair {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return keypair{pub: pub, priv: priv}
}

// signToken signs tok exactly the way pb.SignMessage does: marshal with an
// empty signature, sign the bytes, attach the signature. Done by hand so tests
// can also mint deliberately malformed tokens.
func signToken(t *testing.T, signer keypair, tok *pb.AuthToken) {
	t.Helper()
	tok.Signature = nil
	raw, err := proto.Marshal(tok)
	require.NoError(t, err)
	tok.Signature = &pb.Signature{
		Algorithm: pb.Signature_ED_25519,
		Signer:    signer.pub,
		Value:     ed25519.Sign(signer.priv, raw),
	}
}

// level describes one link of a chain, root first.
type level struct {
	signer      keypair
	subject     []byte
	canDelegate bool
	ops         []pb.Operation
	bucketID    uint64
	expiresIn   time.Duration
	noExpiry    bool
	pieceCid    []byte
}

// buildChain assembles and signs a chain. levels[0] is the root; the returned
// token is the leaf.
func buildChain(t *testing.T, levels ...level) *pb.AuthToken {
	t.Helper()
	var prev *pb.AuthToken
	for _, l := range levels {
		canDelegate := l.canDelegate
		payload := &pb.Payload{
			Prev:        prev,
			Subject:     l.subject,
			CanDelegate: &canDelegate,
			Operations:  l.ops,
			PieceCid:    l.pieceCid,
		}
		if l.bucketID != 0 {
			bucketID := l.bucketID
			payload.BucketId = &bucketID
		}
		if !l.noExpiry {
			expiresAt := time.Now().Add(l.expiresIn).Unix()
			payload.ExpiresAt = &expiresAt
		}
		tok := &pb.AuthToken{Payload: payload}
		signToken(t, l.signer, tok)
		prev = tok
	}
	return prev
}

func encode(t *testing.T, tok *pb.AuthToken) string {
	t.Helper()
	encoded, err := EncodeToken(tok)
	require.NoError(t, err)
	return encoded
}

func encodeBase64(t *testing.T, tok *pb.AuthToken) string {
	t.Helper()
	raw, err := proto.Marshal(tok)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(raw)
}

// bearerToken is the developer-direct shape: one level, no subject, signed by
// the bucket owner.
func bearerToken(t *testing.T, owner keypair) *pb.AuthToken {
	t.Helper()
	return buildChain(t, level{
		signer:    owner,
		ops:       []pb.Operation{pb.Operation_COMPUTE},
		bucketID:  testBucket,
		expiresIn: time.Hour,
	})
}

// delegationChain is the agent-execution shape: owner grant -> delegating
// service -> derived execution key.
func delegationChain(t *testing.T, owner, orch, exec keypair) *pb.AuthToken {
	t.Helper()
	return buildChain(t,
		level{
			signer:      owner,
			subject:     orch.pub,
			canDelegate: true,
			ops:         []pb.Operation{pb.Operation_COMPUTE},
			bucketID:    testBucket,
			expiresIn:   2 * time.Hour,
		},
		level{
			signer:    orch,
			subject:   exec.pub,
			ops:       []pb.Operation{pb.Operation_COMPUTE},
			bucketID:  testBucket,
			expiresIn: time.Hour,
		},
	)
}

// stubIndex is a canned BucketOwnerIndex.
type stubIndex struct {
	result LookupResult
	err    error
	calls  int
}

var _ BucketOwnerIndex = (*stubIndex)(nil)

func (s *stubIndex) Lookup(_ context.Context, _ uint64) (LookupResult, error) {
	s.calls++
	if s.err != nil {
		return LookupResult{}, s.err
	}
	return s.result, nil
}

func fundedIndex(owner keypair) *stubIndex {
	return &stubIndex{result: LookupResult{Owner: owner.pub, HasBalance: true}}
}

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

func TestAuthorize(t *testing.T) {
	owner := newKey(t)
	orch := newKey(t)
	exec := newKey(t)
	attacker := newKey(t)
	stranger := newKey(t)

	tests := []struct {
		name     string
		token    func(t *testing.T) string
		op       pb.Operation
		bucketID uint64
		index    BucketOwnerIndex
		opts     Options
		wantErr  error
		check    func(t *testing.T, v *Verified)
	}{
		{
			// The headline case: eight lines of Go mint a self-consistent
			// token naming somebody else's bucket. Signature verification
			// passes. Only the bucket-owner assertion stops it.
			name: "self-signed forgery against a victim bucket is rejected",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer:    attacker,
					ops:       []pb.Operation{pb.Operation_COMPUTE},
					bucketID:  testBucket,
					expiresIn: 100 * 365 * 24 * time.Hour,
				}))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidOwnership,
		},
		{
			name: "self-signed two-level forgery is rejected",
			token: func(t *testing.T) string {
				// A delegating service's chain with L1 self-signed by the
				// service rather than rooted in the bucket owner.
				return encode(t, delegationChain(t, attacker, attacker, exec))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidOwnership,
		},
		{
			name: "valid one-level owner bearer token",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
				assert.Equal(t, testBucket, v.BucketID)
				assert.Nil(t, v.Subject, "a bearer token names no subject")
				assert.Equal(t, []pb.Operation{pb.Operation_COMPUTE}, v.Operations)
				assert.InDelta(t, time.Now().Add(time.Hour).Unix(), v.ExpiresAt, 5)
			},
		},
		{
			name: "valid two-level delegation chain",
			token: func(t *testing.T) string {
				return encode(t, delegationChain(t, owner, orch, exec))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer,
					"root issuer is the bucket owner, not the delegating service")
				assert.Equal(t, ed25519.PublicKey(exec.pub), v.Subject,
					"leaf subject is the derived execution key")
				assert.Equal(t, testBucket, v.BucketID)
				assert.InDelta(t, time.Now().Add(time.Hour).Unix(), v.ExpiresAt, 5,
					"effective expiry is the leaf's, not the root's")
			},
		},
		{
			name: "valid three-level chain with delegable intermediates",
			token: func(t *testing.T) string {
				mid := newKey(t)
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 3 * time.Hour,
					},
					level{
						signer: orch, subject: mid.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 2 * time.Hour,
					},
					level{
						signer: mid, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
				assert.Equal(t, ed25519.PublicKey(exec.pub), v.Subject)
			},
		},
		{
			name: "root issuer is not the bucket owner",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(stranger),
			wantErr:  ErrInvalidOwnership,
		},
		{
			name: "bucket has no known owner",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    &stubIndex{result: LookupResult{HasBalance: true}},
			wantErr:  ErrInvalidOwnership,
		},
		{
			name: "owner has zero balance",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    &stubIndex{result: LookupResult{Owner: owner.pub}},
			wantErr:  ErrInsufficientBalance,
		},
		{
			name: "removed bucket is rejected even for the funded owner",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index: &stubIndex{result: LookupResult{
				Owner: owner.pub, HasBalance: true, Removed: true,
			}},
			wantErr: ErrBucketRemoved,
		},
		{
			name: "owner lookup failure is a rejection, not an allow",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    &stubIndex{err: errors.New("indexer unreachable")},
			wantErr:  ErrOwnerLookup,
		},
		{
			name: "nil owner index fails closed",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    nil,
			wantErr:  ErrOwnerLookup,
		},
		{
			name: "expired leaf",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer:    owner,
					ops:       []pb.Operation{pb.Operation_COMPUTE},
					bucketID:  testBucket,
					expiresIn: -time.Minute,
				}))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrExpired,
		},
		{
			name: "expired root under a live leaf",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: -time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: -2 * time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrExpired,
		},
		{
			name: "missing expiry on the leaf reads as expired",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer:   owner,
					ops:      []pb.Operation{pb.Operation_COMPUTE},
					bucketID: testBucket,
					noExpiry: true,
				}))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrExpired,
		},
		{
			name: "missing expiry on the root reads as expired",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						noExpiry: true,
					},
					level{
						signer: orch, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrExpired,
		},
		{
			name: "child outliving its parent",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 24 * time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidExpirationTime,
		},
		{
			name: "bucket mismatch between chain levels",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 2 * time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket + 1,
						expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket + 1,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidPermissions,
		},
		{
			name: "token bucket does not match the requested bucket",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket + 99,
			index:    fundedIndex(owner),
			wantErr:  ErrBucketMismatch,
		},
		{
			name: "token naming no bucket is rejected",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer:    owner,
					ops:       []pb.Operation{pb.Operation_COMPUTE},
					expiresIn: time.Hour,
				}))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrBucketMismatch,
		},
		{
			name: "requested bucket zero is rejected",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: 0,
			index:    fundedIndex(owner),
			wantErr:  ErrBucketMismatch,
		},
		{
			name: "leaf lacks the COMPUTE operation",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer:    owner,
					ops:       []pb.Operation{pb.Operation_GET},
					bucketID:  testBucket,
					expiresIn: time.Hour,
				}))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidPermissions,
		},
		{
			name: "root lacks the COMPUTE operation",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops:      []pb.Operation{pb.Operation_GET},
						bucketID: testBucket, expiresIn: 2 * time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops:      []pb.Operation{pb.Operation_GET, pb.Operation_COMPUTE},
						bucketID: testBucket, expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidPermissions,
		},
		{
			// Documents the inverted permissionsValid check we deliberately
			// preserve: a parent granting MORE than the child is rejected.
			// This is why grants must be minted COMPUTE-only.
			name: "known quirk: broader parent above a narrower child is rejected",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						ops: []pb.Operation{
							pb.Operation_PUT, pb.Operation_GET,
							pb.Operation_DELETE, pb.Operation_COMPUTE,
						},
						bucketID: testBucket, expiresIn: 2 * time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops:      []pb.Operation{pb.Operation_COMPUTE},
						bucketID: testBucket, expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidPermissions,
		},
		{
			name: "empty operation set on the root",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: true,
						bucketID: testBucket, expiresIn: 2 * time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops:      []pb.Operation{pb.Operation_COMPUTE},
						bucketID: testBucket, expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidPermissions,
		},
		{
			name: "unknown requested operation",
			token: func(t *testing.T) string {
				return encode(t, bearerToken(t, owner))
			},
			op:       pb.Operation_UNKNOWN,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrUnknownOperation,
		},
		{
			name: "leaf signer is not the party the root delegated to",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: stranger.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 2 * time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidIssuer,
		},
		{
			name: "non-leaf level carries no subject",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 2 * time.Hour,
					},
					level{
						signer: owner, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrDelegatedToken,
		},
		{
			name: "non-delegable root two levels up",
			token: func(t *testing.T) string {
				mid := newKey(t)
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: orch.pub, canDelegate: false,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 3 * time.Hour,
					},
					level{
						signer: orch, subject: mid.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 2 * time.Hour,
					},
					level{
						signer: mid, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
				))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrDelegatedToken,
		},
		{
			name: "chain deeper than the configured bound",
			token: func(t *testing.T) string {
				return encode(t, delegationChain(t, owner, orch, exec))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			opts:     Options{MaxChainDepth: 1},
			wantErr:  ErrChainTooDeep,
		},
		{
			name: "tampered payload breaks the signature",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				bumped := testBucket + 1
				tok.Payload.BucketId = &bumped
				return encode(t, tok)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket + 1,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidSignature,
		},
		{
			name: "signature signed by a key other than the declared signer",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				tok.Signature.Signer = attacker.pub
				return encode(t, tok)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(attacker),
			wantErr:  ErrInvalidSignature,
		},
		{
			name: "unsupported signature algorithm",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				tok.Signature.Algorithm = pb.Signature_Algorithm(99)
				return encode(t, tok)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrUnsupportedAlgorithm,
		},
		{
			name: "undersized signer public key",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				tok.Signature.Signer = []byte{1, 2, 3}
				return encode(t, tok)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidSignature,
		},
		{
			name: "empty signature value",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				tok.Signature.Value = nil
				return encode(t, tok)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidSignature,
		},
		{
			name: "truncated signature value",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				tok.Signature.Value = tok.Signature.Value[:63]
				return encode(t, tok)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidSignature,
		},
		{
			name: "piece-scoped token cannot authorize a bucket operation",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer:    owner,
					ops:       []pb.Operation{pb.Operation_COMPUTE},
					bucketID:  testBucket,
					pieceCid:  []byte("some-piece-cid"),
					expiresIn: time.Hour,
				}))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrPieceScopedToken,
		},
		{
			name:     "empty token",
			token:    func(*testing.T) string { return "" },
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrTokenRequired,
		},
		{
			name:     "undecodable token",
			token:    func(*testing.T) string { return "!!!not-a-token!!!" },
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrInvalidToken,
		},
		{
			name: "encoded token longer than the bound is rejected before decoding",
			token: func(*testing.T) string {
				// All-'z' is valid base58, so this is the shape that costs the
				// most to decode. It must never reach a decoder.
				return strings.Repeat("z", DefaultMaxEncodedTokenLength+1)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			wantErr:  ErrTokenTooLong,
		},
		{
			name: "a real token stays well inside the length bound",
			token: func(t *testing.T) string {
				encoded := encode(t, delegationChain(t, owner, orch, exec))
				require.Less(t, len(encoded), DefaultMaxEncodedTokenLength/4,
					"the two-level agent chain must have ample headroom under the bound")
				return encoded
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
			},
		},
		{
			name: "base58 encoding is accepted",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				raw, err := proto.Marshal(tok)
				require.NoError(t, err)
				return base58.Encode(raw)
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
			},
		},
		{
			name: "base64 encoding is accepted transitionally",
			token: func(t *testing.T) string {
				return encodeBase64(t, bearerToken(t, owner))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
			},
		},
		{
			name: "base64 two-level chain is accepted transitionally",
			token: func(t *testing.T) string {
				return encodeBase64(t, delegationChain(t, owner, orch, exec))
			},
			op:       pb.Operation_COMPUTE,
			bucketID: testBucket,
			index:    fundedIndex(owner),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
				assert.Equal(t, ed25519.PublicKey(exec.pub), v.Subject)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authorizer := NewWithOptions(tc.index, tc.opts)

			verified, err := authorizer.Authorize(context.Background(), tc.token(t), tc.op, tc.bucketID)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, verified, "no Verified may escape a rejection")

				var typed *Error
				require.ErrorAs(t, err, &typed, "every rejection must be typed")
				assert.NotZero(t, typed.Kind, "every rejection must carry a Kind")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, verified)
			if tc.check != nil {
				tc.check(t, verified)
			}
		})
	}
}

// TestAuthorizeDelegateRoot covers the one authorization change of proxy write
// delegation: a token chain may root in the bucket owner OR in one of the
// owner's honored proxy delegates, as vouched for by the index
// (LookupResult.Delegates).
//
// The proxy-type distinction is resolved UPSTREAM of this layer: NonTransfer and
// Any grants both land in Delegates, Governance and Staking never do. So the
// NonTransfer and Any rows have the same shape here, and the Governance row
// degenerates to "a signer the index did not vouch for" — which pins the
// filtering contract at the consuming end: a delegate that arrives without an
// index entry is nobody.
func TestAuthorizeDelegateRoot(t *testing.T) {
	owner := newKey(t)
	delegateNonTransfer := newKey(t)
	delegateAny := newKey(t)
	delegateRevoked := newKey(t)
	delegateGovernance := newKey(t)
	forger := newKey(t)
	agent := newKey(t)
	exec := newKey(t)

	// delegateIndex is an index where owner owns the bucket with balance and
	// vouches for exactly the listed delegates. No arguments means the index
	// knows no delegates — the state after revocation, and the state a
	// Governance-only "delegate" collapses to after upstream filtering.
	delegateIndex := func(delegates ...[]byte) *stubIndex {
		return &stubIndex{result: LookupResult{
			Owner:      owner.pub,
			HasBalance: true,
			Delegates:  delegates,
		}}
	}

	tests := []struct {
		name    string
		token   func(t *testing.T) string
		index   BucketOwnerIndex
		wantErr error
		check   func(t *testing.T, v *Verified)
	}{
		{
			name:  "owner root is still allowed (regression)",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, owner)) },
			index: delegateIndex(),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(owner.pub), v.RootIssuer)
			},
		},
		{
			// NonTransfer is one of the two honored proxy types; upstream
			// filtering has already placed it in Delegates by the time the
			// verifier looks.
			name:  "NonTransfer-honored delegate root is allowed",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, delegateNonTransfer)) },
			index: delegateIndex(delegateNonTransfer.pub),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(delegateNonTransfer.pub), v.RootIssuer,
					"the delegate is the root signer; the owner is not on the chain at all")
			},
		},
		{
			name:  "Any-honored delegate root is allowed",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, delegateAny)) },
			index: delegateIndex(delegateAny.pub),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(delegateAny.pub), v.RootIssuer)
			},
		},
		{
			// Delegates is a LIST and Authorize scans it (slices.ContainsFunc),
			// so a root signed by the SECOND listed delegate must pass. Pins the
			// scan against a first-element-only regression.
			name:  "root signed by the second of multiple delegates is allowed",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, delegateAny)) },
			index: delegateIndex(delegateNonTransfer.pub, delegateAny.pub),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(delegateAny.pub), v.RootIssuer,
					"the second listed delegate is the root signer; the owner is not on the chain at all")
			},
		},
		{
			// A Governance-only proxy is filtered out upstream and never
			// reaches Delegates, so at this layer it is indistinguishable from
			// any other unvouched key — and must deny exactly like one.
			name:    "Governance-only delegate is not honored: empty Delegates denies",
			token:   func(t *testing.T) string { return encode(t, bearerToken(t, delegateGovernance)) },
			index:   delegateIndex(),
			wantErr: ErrInvalidOwnership,
		},
		{
			// Revocation upstream removes the delegate from the index's answer;
			// a token the delegate minted before revocation stops authorizing
			// with no change here.
			name:    "revoked delegate root is denied",
			token:   func(t *testing.T) string { return encode(t, bearerToken(t, delegateRevoked)) },
			index:   delegateIndex(),
			wantErr: ErrInvalidOwnership,
		},
		{
			// The headline forgery, now against a bucket that DOES have a
			// delegate: being a delegate of one key authorizes nobody else's
			// bucket.
			name:    "non-delegate self-sign forgery is denied while another key IS a delegate",
			token:   func(t *testing.T) string { return encode(t, bearerToken(t, forger)) },
			index:   delegateIndex(delegateNonTransfer.pub),
			wantErr: ErrInvalidOwnership,
		},
		{
			// The grant is CHAINS rooted at a delegate, not only one-level
			// tokens: a delegate mints a further level beneath its own root
			// exactly as an owner can, and the root-issuer assertion is what
			// changes, not the chain walk.
			name: "two-level chain rooted at a delegate is allowed",
			token: func(t *testing.T) string {
				return encode(t, delegationChain(t, delegateNonTransfer, agent, exec))
			},
			index: delegateIndex(delegateNonTransfer.pub),
			check: func(t *testing.T, v *Verified) {
				assert.Equal(t, ed25519.PublicKey(delegateNonTransfer.pub), v.RootIssuer,
					"the delegate roots the chain")
				assert.Equal(t, ed25519.PublicKey(exec.pub), v.Subject)
			},
		},
		{
			// Balance is the OWNER's, on the delegate path exactly as on the
			// owner path: a delegate of a bankrupt owner authorizes nothing.
			name:  "delegate root with an owner that cannot pay is denied",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, delegateNonTransfer)) },
			index: &stubIndex{result: LookupResult{
				Owner:      owner.pub,
				HasBalance: false,
				Delegates:  [][]byte{delegateNonTransfer.pub},
			}},
			wantErr: ErrInsufficientBalance,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authorizer := New(tc.index)

			verified, err := authorizer.Authorize(
				context.Background(), tc.token(t), pb.Operation_COMPUTE, testBucket)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, verified, "no Verified may escape a rejection")

				var typed *Error
				require.ErrorAs(t, err, &typed, "every rejection must be typed")
				assert.NotZero(t, typed.Kind, "every rejection must carry a Kind")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, verified)
			if tc.check != nil {
				tc.check(t, verified)
			}
		})
	}
}

// TestRejectionKinds pins the 401 / 403 / 503 split callers map onto responses.
// The scenarios are real Authorize calls, not direct calls to the classifier.
func TestRejectionKinds(t *testing.T) {
	owner := newKey(t)
	orch := newKey(t)
	exec := newKey(t)
	attacker := newKey(t)
	stranger := newKey(t)

	tests := []struct {
		name       string
		token      func(t *testing.T) string
		index      BucketOwnerIndex
		wantKind   Kind
		wantStatus int
	}{
		{
			name:  "no token",
			token: func(*testing.T) string { return "" },
			index: fundedIndex(owner), wantKind: KindUnauthenticated, wantStatus: 401,
		},
		{
			name:  "undecodable token",
			token: func(*testing.T) string { return "!!!nope!!!" },
			index: fundedIndex(owner), wantKind: KindUnauthenticated, wantStatus: 401,
		},
		{
			name: "expired token",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer: owner, ops: []pb.Operation{pb.Operation_COMPUTE},
					bucketID: testBucket, expiresIn: -time.Minute,
				}))
			},
			index: fundedIndex(owner), wantKind: KindUnauthenticated, wantStatus: 401,
		},
		{
			name: "tampered signature",
			token: func(t *testing.T) string {
				tok := bearerToken(t, owner)
				tok.Signature.Value[0] ^= 0xff
				return encode(t, tok)
			},
			index: fundedIndex(owner), wantKind: KindUnauthenticated, wantStatus: 401,
		},
		{
			name: "broken chain invariant",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t,
					level{
						signer: owner, subject: stranger.pub, canDelegate: true,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: 2 * time.Hour,
					},
					level{
						signer: orch, subject: exec.pub,
						ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
						expiresIn: time.Hour,
					},
				))
			},
			index: fundedIndex(owner), wantKind: KindUnauthenticated, wantStatus: 401,
		},
		{
			name:  "forged token against a victim bucket",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, attacker)) },
			index: fundedIndex(owner), wantKind: KindForbidden, wantStatus: 403,
		},
		{
			name:     "owner cannot pay",
			token:    func(t *testing.T) string { return encode(t, bearerToken(t, owner)) },
			index:    &stubIndex{result: LookupResult{Owner: owner.pub}},
			wantKind: KindForbidden, wantStatus: 403,
		},
		{
			name:  "bucket removed",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, owner)) },
			index: &stubIndex{result: LookupResult{
				Owner: owner.pub, HasBalance: true, Removed: true,
			}},
			wantKind: KindForbidden, wantStatus: 403,
		},
		{
			name: "operation not granted",
			token: func(t *testing.T) string {
				return encode(t, buildChain(t, level{
					signer: owner, ops: []pb.Operation{pb.Operation_GET},
					bucketID: testBucket, expiresIn: time.Hour,
				}))
			},
			index: fundedIndex(owner), wantKind: KindForbidden, wantStatus: 403,
		},
		{
			name:     "index unavailable",
			token:    func(t *testing.T) string { return encode(t, bearerToken(t, owner)) },
			index:    &stubIndex{err: errors.New("indexer unreachable")},
			wantKind: KindUnavailable, wantStatus: 503,
		},
		{
			name:  "no index configured",
			token: func(t *testing.T) string { return encode(t, bearerToken(t, owner)) },
			index: nil, wantKind: KindUnavailable, wantStatus: 503,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.index).Authorize(
				context.Background(), tc.token(t), pb.Operation_COMPUTE, testBucket)
			require.Error(t, err)
			assert.Equal(t, tc.wantKind, KindOf(err))
			assert.Equal(t, tc.wantStatus, KindOf(err).HTTPStatus())
		})
	}
}

func TestKindOfNonAuthzErrorDenies(t *testing.T) {
	// A caller that hands KindOf something unrelated — or nil — must not get a
	// permissive answer back.
	assert.Equal(t, KindForbidden, KindOf(errors.New("something else")))
	assert.Equal(t, KindForbidden, KindOf(nil))
	assert.Equal(t, 403, Kind(0).HTTPStatus(), "an unset Kind still denies")
}

// TestDecodeTokenAgreeingDecodersAreOneReading guards the flip side: standard
// and URL base64 of the same bytes must not read as an ambiguity.
func TestDecodeTokenAgreeingDecodersAreOneReading(t *testing.T) {
	owner := newKey(t)
	raw, err := proto.Marshal(bearerToken(t, owner))
	require.NoError(t, err)

	std := base64.StdEncoding.EncodeToString(raw)
	if std != base64.URLEncoding.EncodeToString(raw) {
		t.Skip("payload happens to contain + or /; both alphabets do not coincide")
	}

	decoded, err := DecodeToken(std)
	require.NoError(t, err)
	assert.Equal(t, []byte(owner.pub), decoded.GetSignature().GetSigner())
}

func TestAuthorizeDoesNotConsultIndexOnAMalformedChain(t *testing.T) {
	owner := newKey(t)
	index := fundedIndex(owner)
	authorizer := New(index)

	expired := encode(t, buildChain(t, level{
		signer:    owner,
		ops:       []pb.Operation{pb.Operation_COMPUTE},
		bucketID:  testBucket,
		expiresIn: -time.Hour,
	}))

	_, err := authorizer.Authorize(context.Background(), expired, pb.Operation_COMPUTE, testBucket)
	require.ErrorIs(t, err, ErrExpired)
	assert.Zero(t, index.calls, "a chain that fails locally must not reach the indexer")
}

func TestAuthorizeUsesInjectedClock(t *testing.T) {
	owner := newKey(t)
	authorizer := NewWithOptions(fundedIndex(owner), Options{
		Now: func() time.Time { return time.Now().Add(48 * time.Hour) },
	})

	token := encode(t, bearerToken(t, owner))

	_, err := authorizer.Authorize(context.Background(), token, pb.Operation_COMPUTE, testBucket)
	assert.ErrorIs(t, err, ErrExpired)
}

func TestDecodeToken(t *testing.T) {
	owner := newKey(t)
	token := bearerToken(t, owner)
	raw, err := proto.Marshal(token)
	require.NoError(t, err)

	payloadOnly, err := proto.Marshal(&pb.AuthToken{Payload: token.Payload})
	require.NoError(t, err)

	tests := []struct {
		name    string
		encoded string
		wantErr error
	}{
		{name: "base58", encoded: base58.Encode(raw)},
		{name: "base64 std", encoded: base64.StdEncoding.EncodeToString(raw)},
		{name: "not an encoding", encoded: "!!!not-base58-or-base64!!!", wantErr: ErrInvalidToken},
		{name: "not a token", encoded: base58.Encode([]byte("hello there")), wantErr: ErrInvalidToken},
		// Decodable protobuf with no signature at all: shaped like a token, not a
		// credential.
		{name: "payload only", encoded: base58.Encode(payloadOnly), wantErr: ErrInvalidToken},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decoded, decodedRaw, err := DecodeTokenBytes(tc.encoded)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, decoded)
				assert.Nil(t, decodedRaw)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, decoded)
			assert.Equal(t, testBucket, decoded.GetPayload().GetBucketId())
			assert.Equal(t, []byte(owner.pub), decoded.GetSignature().GetSigner())
			// The bytes come back verbatim whichever encoding they arrived in:
			// a component forwarding the credential to another verifier has to
			// hand over the bytes the signature was computed over.
			assert.Equal(t, raw, decodedRaw)
		})
	}
}

func TestEncodeTokenRoundTrip(t *testing.T) {
	owner := newKey(t)
	orch := newKey(t)
	exec := newKey(t)

	encoded, err := EncodeToken(delegationChain(t, owner, orch, exec))
	require.NoError(t, err)

	decoded, err := DecodeToken(encoded)
	require.NoError(t, err)

	assert.Equal(t, []byte(exec.pub), decoded.GetPayload().GetSubject())
	assert.Equal(t, []byte(owner.pub), decoded.GetPayload().GetPrev().GetSignature().GetSigner())

	// The round trip must preserve verifiability of every level.
	verified, err := VerifySignature(decoded)
	require.NoError(t, err)
	assert.True(t, verified)
}

// TestOwnershipCheckIsTheOnlyThingStoppingAForgery is the mutation guard for the
// headline case. It asserts that the forged chain is otherwise COMPLETELY VALID —
// signatures verify at every level, it is unexpired, it names the billed bucket
// and it grants COMPUTE — so the only check standing between it and an allow is
// the root-issuer-equals-bucket-owner comparison in Authorize.
//
// If that comparison were deleted, the "self-signed forgery against a victim
// bucket" case in TestAuthorize would not merely report a different error: the
// call would SUCCEED, because everything else about the token passes. This test
// proves that by running the identical token against an index that names the
// attacker as the owner and showing it is then authorized.
func TestOwnershipCheckIsTheOnlyThingStoppingAForgery(t *testing.T) {
	victimOwner := newKey(t)
	attacker := newKey(t)

	forged := encode(t, buildChain(t, level{
		signer:    attacker,
		ops:       []pb.Operation{pb.Operation_COMPUTE},
		bucketID:  testBucket,
		expiresIn: 100 * 365 * 24 * time.Hour,
	}))

	// Every local check the chain walk performs passes: the walk itself accepts
	// the token and resolves the attacker as the root issuer.
	decoded, err := DecodeToken(forged)
	require.NoError(t, err)
	expanded := &expandedToken{}
	require.NoError(t,
		NewWithOptions(nil, Options{}).(*authorizer).expandAndVerifyToken(decoded, expanded, 0),
		"the forged chain is internally valid — nothing about the token itself is wrong")
	require.NoError(t, verifyRequestedAction(expanded, pb.Operation_COMPUTE, testBucket),
		"the forged chain names the billed bucket and grants COMPUTE")
	require.Equal(t, []byte(attacker.pub), expanded.rootIssuer)

	// Against the real owner index it is refused, and only on ownership.
	_, err = New(fundedIndex(victimOwner)).Authorize(
		context.Background(), forged, pb.Operation_COMPUTE, testBucket)
	require.ErrorIs(t, err, ErrInvalidOwnership)
	assert.Equal(t, KindForbidden, KindOf(err))

	// Swap the index so the attacker IS the owner: the very same bytes are now
	// authorized. Nothing but the ownership comparison changed.
	verified, err := New(fundedIndex(attacker)).Authorize(
		context.Background(), forged, pb.Operation_COMPUTE, testBucket)
	require.NoError(t, err, "the token is valid in every respect except who owns the bucket")
	require.NotNil(t, verified)
	assert.Equal(t, ed25519.PublicKey(attacker.pub), verified.RootIssuer)
}

// TestPreservedQuirkEmptyLeafOperationsInheritParent pins quirk 3 from the
// package doc. A leaf minted with no operations does not grant nothing — it
// adopts its parent's set, because the attenuation check is skipped while the
// accumulated set is still empty.
//
// Only the holder of the parent grant can mint such a leaf, so no authority is
// widened; the point of the test is that "operations: []" is not a way to
// express "no authority", and that a future fix must be deliberate.
func TestPreservedQuirkEmptyLeafOperationsInheritParent(t *testing.T) {
	owner, orch, exec := newKey(t), newKey(t), newKey(t)

	token := encode(t, buildChain(t,
		level{
			signer: owner, subject: orch.pub, canDelegate: true,
			ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
			expiresIn: 2 * time.Hour,
		},
		level{
			signer: orch, subject: exec.pub,
			ops:       nil, // leaf grants nothing explicitly
			bucketID:  testBucket,
			expiresIn: time.Hour,
		},
	))

	verified, err := New(fundedIndex(owner)).Authorize(
		context.Background(), token, pb.Operation_COMPUTE, testBucket)
	require.NoError(t, err, "documented quirk: an empty leaf set inherits the parent's")
	assert.Equal(t, []pb.Operation{pb.Operation_COMPUTE}, verified.Operations)
}

// TestPreservedQuirkCanDelegateUnenforcedAboveLeaf pins quirk 4 from the package
// doc: the CanDelegate check is `depth > 1`, so a grant minted with
// canDelegate=false still accepts exactly one level signed beneath it.
func TestPreservedQuirkCanDelegateUnenforcedAboveLeaf(t *testing.T) {
	owner, orch, exec := newKey(t), newKey(t), newKey(t)

	token := encode(t, buildChain(t,
		level{
			signer: owner, subject: orch.pub, canDelegate: false,
			ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
			expiresIn: 2 * time.Hour,
		},
		level{
			signer: orch, subject: exec.pub,
			ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
			expiresIn: time.Hour,
		},
	))

	verified, err := New(fundedIndex(owner)).Authorize(
		context.Background(), token, pb.Operation_COMPUTE, testBucket)
	require.NoError(t, err, "documented quirk: canDelegate is only consulted at depth > 1")
	assert.Equal(t, ed25519.PublicKey(exec.pub), verified.Subject)

	// A third level is where the flag finally bites.
	mid := newKey(t)
	deeper := encode(t, buildChain(t,
		level{
			signer: owner, subject: orch.pub, canDelegate: false,
			ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
			expiresIn: 3 * time.Hour,
		},
		level{
			signer: orch, subject: mid.pub, canDelegate: true,
			ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
			expiresIn: 2 * time.Hour,
		},
		level{
			signer: mid, subject: exec.pub,
			ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
			expiresIn: time.Hour,
		},
	))
	_, err = New(fundedIndex(owner)).Authorize(
		context.Background(), deeper, pb.Operation_COMPUTE, testBucket)
	assert.ErrorIs(t, err, ErrDelegatedToken)
}

// TestStolenGrantCannotBeExtended: an L1 owner grant to a service is durable
// authorization state, not a bearer secret, because whoever steals it cannot
// sign a level beneath it without the service's private key.
func TestStolenGrantCannotBeExtended(t *testing.T) {
	owner, orch, attacker := newKey(t), newKey(t), newKey(t)

	// The owner's L1 grant, delegating to a service.
	grant := buildChain(t, level{
		signer: owner, subject: orch.pub, canDelegate: true,
		ops: []pb.Operation{pb.Operation_COMPUTE}, bucketID: testBucket,
		expiresIn: 2 * time.Hour,
	})

	// The attacker has the grant bytes and tries to sign a leaf beneath it with
	// their own key.
	expiresAt := time.Now().Add(time.Hour).Unix()
	canDelegate := false
	forgedLeaf := &pb.AuthToken{Payload: &pb.Payload{
		Prev:        grant,
		Subject:     attacker.pub,
		CanDelegate: &canDelegate,
		Operations:  []pb.Operation{pb.Operation_COMPUTE},
		BucketId:    &[]uint64{testBucket}[0],
		ExpiresAt:   &expiresAt,
	}}
	signToken(t, attacker, forgedLeaf)

	_, err := New(fundedIndex(owner)).Authorize(
		context.Background(), encode(t, forgedLeaf), pb.Operation_COMPUTE, testBucket)
	require.ErrorIs(t, err, ErrInvalidIssuer,
		"the leaf signer must equal the grant's Subject, which the attacker does not hold")
}

// TestStolenBearerTokenCannotBeExtended is the same property for the
// developer-direct one-level token: it names no Subject, so nothing may be
// chained beneath it at all.
func TestStolenBearerTokenCannotBeExtended(t *testing.T) {
	owner, attacker := newKey(t), newKey(t)

	stolen := bearerToken(t, owner)

	expiresAt := time.Now().Add(time.Minute).Unix()
	canDelegate := false
	forgedLeaf := &pb.AuthToken{Payload: &pb.Payload{
		Prev:        stolen,
		Subject:     attacker.pub,
		CanDelegate: &canDelegate,
		Operations:  []pb.Operation{pb.Operation_COMPUTE},
		BucketId:    &[]uint64{testBucket}[0],
		ExpiresAt:   &expiresAt,
	}}
	signToken(t, attacker, forgedLeaf)

	_, err := New(fundedIndex(owner)).Authorize(
		context.Background(), encode(t, forgedLeaf), pb.Operation_COMPUTE, testBucket)
	require.ErrorIs(t, err, ErrDelegatedToken,
		"a token with no Subject cannot have anything chained beneath it")
}

// TestDecodeTokenLengthBoundIsCheapForHostileInput asserts the bound exists and
// that oversized input is refused without paying the quadratic decode cost.
//
// The timing assertion is deliberately loose — it is asserting the absence of a
// second-scale operation, not measuring performance. Unbounded, this input takes
// ~18s to base58-decode on one core; bounded, it never reaches a decoder.
func TestDecodeTokenLengthBoundIsCheapForHostileInput(t *testing.T) {
	hostile := strings.Repeat("z", 256<<10) // all-'z' is valid base58

	start := time.Now()
	decoded, err := DecodeToken(hostile)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrTokenTooLong)
	assert.Nil(t, decoded)
	assert.Less(t, elapsed, time.Second,
		"an oversized credential must be refused without being decoded")
}

func TestDecodeTokenLengthBoundIsConfigurable(t *testing.T) {
	owner := newKey(t)
	token := encode(t, bearerToken(t, owner))

	// A bound below the real token's length rejects it...
	_, err := NewWithOptions(fundedIndex(owner), Options{MaxEncodedTokenLength: 10}).
		Authorize(context.Background(), token, pb.Operation_COMPUTE, testBucket)
	require.ErrorIs(t, err, ErrTokenTooLong)

	// ...and the default admits it.
	_, err = New(fundedIndex(owner)).
		Authorize(context.Background(), token, pb.Operation_COMPUTE, testBucket)
	require.NoError(t, err)
}

func TestPermissionsValidInversionIsPreserved(t *testing.T) {
	// Pins the documented quirk so a future "cleanup" cannot silently flip
	// live storage semantics. Argument order at the call site is
	// permissionsValid(childOps, parentOps).
	compute := []pb.Operation{pb.Operation_COMPUTE}
	full := []pb.Operation{pb.Operation_PUT, pb.Operation_GET, pb.Operation_DELETE}

	assert.True(t, permissionsValid(compute, compute), "identical sets pass")
	assert.False(t, permissionsValid(compute, full), "parent broader than child fails")
	assert.True(t, permissionsValid(full, compute[:0]), "empty parent set passes")
	assert.False(t, permissionsValid(nil, compute), "empty child set always fails")
}

// TestExpiresAtUnitNormalisation pins the fix for a confirmed fleet-wide bug:
// wallet-minted access tokens never expired.
//
// The JS SDK writes ExpiresAt in MILLISECONDS (AuthToken.create sets
// `expiresAt: Date.now() + expiresIn`, 30-day default). Comparing that against a
// SECONDS clock computes `1.8e9 > 1.8e12`, false until roughly the year 58,000.
//
// Go minters write SECONDS. Both must verify correctly.
func TestExpiresAtUnitNormalisation(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	nowSec := now.Unix()

	for _, tt := range []struct {
		name      string
		expiresAt int64
		expired   bool
	}{
		// SDK shape: milliseconds.
		{"millis, 30 days ahead (SDK default)", (nowSec + 30*24*3600) * 1000, false},
		{"millis, one hour ahead", (nowSec + 3600) * 1000, false},
		{"millis, one hour ago", (nowSec - 3600) * 1000, true},
		// Orchestrator shape: seconds.
		{"seconds, one hour ahead", nowSec + 3600, false},
		{"seconds, one hour ago", nowSec - 3600, true},
		// Absent expiry reads as expired (storage semantics).
		{"zero", 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			exp := tt.expiresAt
			token := &pb.AuthToken{Payload: &pb.Payload{ExpiresAt: &exp}}
			a := &authorizer{now: func() time.Time { return now }}
			require.Equal(t, tt.expired, a.isExpired(token))
		})
	}
}

// TestExpiresAtMillisIsNotAnImmortalityBypass guards the normalisation against
// being read as a loophole: a millisecond expiry that has genuinely passed must
// still be rejected, and it must be rejected for the same reason a seconds one
// is. Without normalisation the first case here is accepted, which is the bug.
func TestExpiresAtMillisIsNotAnImmortalityBypass(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	pastMillis := (now.Unix() - 24*3600) * 1000

	exp := pastMillis
	token := &pb.AuthToken{Payload: &pb.Payload{ExpiresAt: &exp}}
	a := &authorizer{now: func() time.Time { return now }}

	require.True(t, a.isExpired(token),
		"a millisecond expiry a day in the past must be expired; reading it as seconds makes it immortal")

	// This deliberately says nothing about pb.AuthToken.IsExpired. It was unit-blind
	// when expiresAtSeconds was written, and the same normalisation is going upstream
	// in ddc-dac (Cerebellum-Network/ddc-dac#46) — so asserting the upstream bug here
	// would turn that dependency bump into a red build on dev. Once the bump lands,
	// expiresAtSeconds and this test can both go.
}

// TestVerifySignatureRejectsUnusableSignaturesWithoutPanicking covers inputs
// that arrive off the wire and that a naive verifier panics on or misreads: an
// unknown algorithm, and a signer or value of the wrong size. Each must return
// ErrUnusableSignature rather than panic.
func TestVerifySignatureRejectsUnusableSignaturesWithoutPanicking(t *testing.T) {
	for _, tt := range []struct {
		name string
		sig  *pb.Signature
	}{
		{"nil signature", nil},
		{"algorithm 2", &pb.Signature{Algorithm: pb.Signature_Algorithm(2), Signer: make([]byte, 32), Value: make([]byte, 64)}},
		{"algorithm 9999", &pb.Signature{Algorithm: pb.Signature_Algorithm(9999), Signer: make([]byte, 32), Value: make([]byte, 64)}},
		{"ed25519 signer 31 bytes", &pb.Signature{Algorithm: pb.Signature_ED_25519, Signer: make([]byte, 31), Value: make([]byte, 64)}},
		{"ed25519 signer 33 bytes", &pb.Signature{Algorithm: pb.Signature_ED_25519, Signer: make([]byte, 33), Value: make([]byte, 64)}},
		{"ed25519 signer empty", &pb.Signature{Algorithm: pb.Signature_ED_25519, Value: make([]byte, 64)}},
		{"sr25519 signer empty", &pb.Signature{Algorithm: pb.Signature_SR_25519, Value: make([]byte, 64)}},
		{"sr25519 signer 31 bytes", &pb.Signature{Algorithm: pb.Signature_SR_25519, Signer: make([]byte, 31), Value: make([]byte, 64)}},
		{"empty signature value", &pb.Signature{Algorithm: pb.Signature_ED_25519, Signer: make([]byte, 32)}},
		{"ed25519 value 63 bytes", &pb.Signature{Algorithm: pb.Signature_ED_25519, Signer: make([]byte, 32), Value: make([]byte, 63)}},
		{"sr25519 value 65 bytes", &pb.Signature{Algorithm: pb.Signature_SR_25519, Signer: make([]byte, 32), Value: make([]byte, 65)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			token := &pb.AuthToken{Payload: &pb.Payload{}, Signature: tt.sig}
			require.NotPanics(t, func() {
				ok, err := VerifySignature(token)
				require.False(t, ok)
				require.ErrorIs(t, err, ErrUnusableSignature)
			})
		})
	}
}

func TestVerifySignatureRejectsNilMessage(t *testing.T) {
	require.NotPanics(t, func() {
		ok, err := VerifySignature(nil)
		require.False(t, ok)
		require.ErrorIs(t, err, ErrUnusableSignature)
	})
}

// TestVerifySignatureAcceptsWellFormedSignatures guards against the guard being
// over-broad: a structurally valid signature of either supported algorithm must
// reach the crypto rather than being rejected up front. The signatures are not
// cryptographically valid, so verification returns false, with no error.
func TestVerifySignatureAcceptsWellFormedSignatures(t *testing.T) {
	for _, algo := range []pb.Signature_Algorithm{pb.Signature_ED_25519, pb.Signature_SR_25519} {
		token := &pb.AuthToken{
			Payload:   &pb.Payload{},
			Signature: &pb.Signature{Algorithm: algo, Signer: make([]byte, 32), Value: make([]byte, 64)},
		}
		ok, err := VerifySignature(token)
		require.False(t, ok)
		require.NoError(t, err, "a well-formed signature must reach the crypto, not be rejected by the guard")
	}
}

func TestBucketID(t *testing.T) {
	owner, orch, exec := newKey(t), newKey(t), newKey(t)

	assert.Equal(t, uint64(0), BucketID(nil), "nil token names no bucket")
	assert.Equal(t, uint64(0), BucketID(&pb.AuthToken{}), "empty token names no bucket")
	assert.Equal(t, testBucket, BucketID(bearerToken(t, owner)), "one-level bearer token")
	assert.Equal(t, testBucket, BucketID(delegationChain(t, owner, orch, exec)), "two-level agent chain")
}

func TestBucketIDReadsThroughALeafThatNamesNoBucket(t *testing.T) {
	// The storage node's walk tolerates a level with no bucket_id as long as some level
	// names one, so the bridge has to look past the leaf.
	owner, orch := newKey(t), newKey(t)
	chain := buildChain(t,
		level{
			signer:      owner,
			subject:     orch.pub,
			canDelegate: true,
			ops:         []pb.Operation{pb.Operation_COMPUTE},
			bucketID:    testBucket,
			expiresIn:   2 * time.Hour,
		},
		level{
			signer:    orch,
			ops:       []pb.Operation{pb.Operation_COMPUTE},
			expiresIn: time.Hour,
		},
	)
	assert.Equal(t, testBucket, BucketID(chain))
}

func TestBucketIDIsBoundedByChainDepth(t *testing.T) {
	// A chain deeper than the walk will ever accept must not be walked
	// indefinitely here either.
	signer := newKey(t)
	levels := make([]level, 0, DefaultMaxChainDepth+4)
	for i := 0; i < DefaultMaxChainDepth+3; i++ {
		levels = append(levels, level{
			signer:    signer,
			subject:   signer.pub,
			ops:       []pb.Operation{pb.Operation_COMPUTE},
			expiresIn: time.Hour,
		})
	}
	// Only the ROOT names the bucket, and the root sits past the depth bound.
	levels[0].bucketID = testBucket

	assert.Equal(t, uint64(0), BucketID(buildChain(t, levels...)),
		"the walk stops at DefaultMaxChainDepth, so a bucket buried deeper is not reported")
}

func TestBucketIDClaimingSomebodyElsesBucketIsStillRejected(t *testing.T) {
	// The value BucketID returns is attacker-controlled. This pins the reason
	// that is safe: feeding it back into Authorize still fails ownership.
	victimOwner, forger := newKey(t), newKey(t)
	forged := bearerToken(t, forger) // names testBucket, signed by the forger

	bucketID := BucketID(forged)
	require.Equal(t, testBucket, bucketID)

	authorizer := New(fundedIndex(victimOwner))
	verified, err := authorizer.Authorize(context.Background(), encode(t, forged), pb.Operation_COMPUTE, bucketID)
	assert.Nil(t, verified)
	assert.ErrorIs(t, err, ErrInvalidOwnership)
}
