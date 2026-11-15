package auth

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

// Golden tokens signed by the implementation this package replaces
// (cerebellum-network/ddc-dac pb with cere-ddc-sdk-go/core crypto), from the
// seed 0x0102…20. Each payload is bucket 1337, operations {GET, COMPUTE},
// expiresAt 4102444800. "Wrapped" tokens are signed over
// "<Bytes>" + message + "</Bytes>", as polkadot.js wallets sign raw bytes.
const (
	goldenEd25519PublicKey = "79b5562e8fe654f94078b112e8a98ba7901f853ae695bed7e0e3910bad049664"
	goldenSr25519PublicKey = "10c68432943c68a6e1be650818b5e08db79e57823de9f34df7ba36d404d91e1d"

	goldenEd25519Token        = "yM6K6KGjf46TEr8t3gHNUr2VPajiRAmcGM9UCV78JaShAtoAs2XBdB8QU4Swe8zvnNTs9U73gMcyWkZBt3cNc6ZVu2n6vCW6eGSoiXwRj7qdEayfDiXdM9Rfn2kv131mk4YVdQVjVZ9iVYFybM6m6iw5uKr9YEr"
	goldenEd25519WrappedToken = "yM6K6KGjf46TEr8t3gHNUr2VPajiRAmcGM9UCV78JaShAtoAs2XSwdSUdWdkiLwT1mXz3FbhigQDskvyQAKbqd7xUWemgwzYB47xWkFkE2oZyEX8z8hB9EoJJWpDKaR8frxQPDntZ1oor9ARjD1xhS2qLwjFENi"
	goldenSr25519Token        = "KwXUcvJpoTNdBiDFJJAjj8mnkumQkfaKdaejwYRrbav5EHV3GEdLEt2dDijL2XrpzkqjVXLS3tyRkPECjmhc9uMLefWqPTkM6qnN6Gc244BybKKUR3HhTqaNvgLqQcq8wvDZ63svASSzWxR2kHZFpJKd6uQ8sAjw98"
	goldenSr25519WrappedToken = "KwXUcvJpoTNdBiDFJJAjj8mnkumQkfaKdaejwYRrbav5EHV3GEdLEt4d29wQWAnd8GP1SBcBWC5QH41cXAsS5u1FBz9hPkMKA5BrdEMGpFnx9oVCXpvkXa8FXsqtgqG4YeDBQsgiKt4kWq18fapaLnY9GAnLrKe1Z8"
)

func goldenSeed() [32]byte {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	return seed
}

func goldenPayload() *pb.Payload {
	bucket, expires := uint64(1337), int64(4102444800)
	return &pb.Payload{
		BucketId:   &bucket,
		ExpiresAt:  &expires,
		Operations: []pb.Operation{pb.Operation_GET, pb.Operation_COMPUTE},
	}
}

func TestVerifySignatureAcceptsGoldenTokens(t *testing.T) {
	for _, tt := range []struct {
		name, token string
		algorithm   pb.Signature_Algorithm
		publicKey   string
	}{
		{"ed25519", goldenEd25519Token, pb.Signature_ED_25519, goldenEd25519PublicKey},
		{"ed25519 wrapped", goldenEd25519WrappedToken, pb.Signature_ED_25519, goldenEd25519PublicKey},
		{"sr25519", goldenSr25519Token, pb.Signature_SR_25519, goldenSr25519PublicKey},
		{"sr25519 wrapped", goldenSr25519WrappedToken, pb.Signature_SR_25519, goldenSr25519PublicKey},
	} {
		t.Run(tt.name, func(t *testing.T) {
			token, err := DecodeToken(tt.token)
			require.NoError(t, err)
			require.Equal(t, tt.algorithm, token.GetSignature().GetAlgorithm())
			require.Equal(t, tt.publicKey, hex.EncodeToString(token.GetSignature().GetSigner()))
			require.True(t, proto.Equal(goldenPayload(), token.GetPayload()), "golden payload")

			ok, err := VerifySignature(token)
			require.NoError(t, err)
			assert.True(t, ok)

			// The same signature over a different payload must not verify.
			token.Payload.Operations = []pb.Operation{pb.Operation_GET}
			ok, err = VerifySignature(token)
			require.NoError(t, err)
			assert.False(t, ok, "tampered payload")
		})
	}
}

func TestVerifySignatureRestoresTheSignature(t *testing.T) {
	token, err := DecodeToken(goldenSr25519Token)
	require.NoError(t, err)
	sig := token.GetSignature()

	_, err = VerifySignature(token)
	require.NoError(t, err)
	assert.Same(t, sig, token.GetSignature())
}

func TestSignersDeriveTheGoldenPublicKeys(t *testing.T) {
	seed := goldenSeed()
	ed := NewEd25519Signer(ed25519.NewKeyFromSeed(seed[:]))
	sr, err := NewSr25519Signer(seed)
	require.NoError(t, err)

	assert.Equal(t, goldenEd25519PublicKey, hex.EncodeToString(ed.PublicKey()))
	assert.Equal(t, pb.Signature_ED_25519, ed.Algorithm())
	assert.Equal(t, goldenSr25519PublicKey, hex.EncodeToString(sr.PublicKey()))
	assert.Equal(t, pb.Signature_SR_25519, sr.Algorithm())
}

func TestSignMessageRoundTrip(t *testing.T) {
	seed := goldenSeed()
	sr, err := NewSr25519Signer(seed)
	require.NoError(t, err)

	for name, signer := range map[string]Signer{
		"ed25519": NewEd25519Signer(ed25519.NewKeyFromSeed(seed[:])),
		"sr25519": sr,
	} {
		t.Run(name, func(t *testing.T) {
			token := &pb.AuthToken{Payload: goldenPayload()}
			sig, err := SignMessage(signer, token)
			require.NoError(t, err)
			assert.Same(t, sig, token.GetSignature())
			assert.Equal(t, signer.Algorithm(), sig.GetAlgorithm(), "the signature records the signer's algorithm")

			encoded, err := EncodeToken(token)
			require.NoError(t, err)
			decoded, err := DecodeToken(encoded)
			require.NoError(t, err)
			ok, err := VerifySignature(decoded)
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}
}

// ed25519 signatures are deterministic, so the SDK must reproduce the golden
// token byte for byte.
func TestSignMessageReproducesGoldenEd25519Token(t *testing.T) {
	seed := goldenSeed()
	token := &pb.AuthToken{Payload: goldenPayload()}
	require.NoError(t, SignToken(ed25519.NewKeyFromSeed(seed[:]), token))
	encoded, err := EncodeToken(token)
	require.NoError(t, err)
	assert.Equal(t, goldenEd25519Token, encoded)
}

func TestSignMessageRefusesASignedMessage(t *testing.T) {
	seed := goldenSeed()
	token, err := DecodeToken(goldenEd25519Token)
	require.NoError(t, err)
	_, err = SignMessage(NewEd25519Signer(ed25519.NewKeyFromSeed(seed[:])), token)
	require.ErrorIs(t, err, pb.ErrSignatureNotEmpty)
}

func TestVerifySignatureReportsAMissingSignature(t *testing.T) {
	ok, err := VerifySignature(&pb.AuthToken{Payload: goldenPayload()})
	assert.False(t, ok)
	require.ErrorIs(t, err, ErrUnusableSignature)
	require.ErrorIs(t, err, pb.ErrSignatureEmpty)
}

func TestNewSignerAdaptsAnyKeyHolder(t *testing.T) {
	seed := goldenSeed()
	key := ed25519.NewKeyFromSeed(seed[:])
	signer := NewSigner(pb.Signature_ED_25519, key.Public().(ed25519.PublicKey), func(m []byte) ([]byte, error) {
		return ed25519.Sign(key, m), nil
	})
	token := &pb.AuthToken{Payload: goldenPayload()}
	_, err := SignMessage(signer, token)
	require.NoError(t, err)
	ok, err := VerifySignature(token)
	require.NoError(t, err)
	assert.True(t, ok)
}
