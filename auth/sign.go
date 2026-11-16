package auth

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/ChainSafe/go-schnorrkel"
	"google.golang.org/protobuf/proto"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

// sr25519 keys and signatures have fixed sizes, the same as ed25519's.
const (
	publicKeySize = 32
	signatureSize = 64
)

// substrateContext is the sr25519 signing context Substrate and polkadot.js use.
var substrateContext = []byte("substrate")

// ErrUnusableSignature is returned for a signature that cannot be verified: no
// signature, an unknown algorithm, or a signer or value of the wrong size.
var ErrUnusableSignature = errors.New("signature is unusable")

// Signer signs messages for one identity.
type Signer interface {
	// Algorithm is recorded in the signature, so verifiers pick the right scheme.
	Algorithm() pb.Signature_Algorithm
	// PublicKey is recorded in the signature as its signer.
	PublicKey() []byte
	// Sign signs message as-is.
	Sign(message []byte) ([]byte, error)
}

// NewEd25519Signer returns a Signer for an ed25519 private key.
func NewEd25519Signer(key ed25519.PrivateKey) Signer {
	return &funcSigner{
		algorithm: pb.Signature_ED_25519,
		publicKey: key.Public().(ed25519.PublicKey),
		sign:      func(message []byte) ([]byte, error) { return ed25519.Sign(key, message), nil },
	}
}

// NewSr25519Signer returns a Signer for an sr25519 key, given its 32-byte
// mini secret (the seed Substrate derives a keypair from). Keys are expanded the
// way Substrate expands them, and messages are signed in the "substrate"
// context, so signatures verify on chain and in polkadot.js.
func NewSr25519Signer(miniSecret [32]byte) (Signer, error) {
	msk, err := schnorrkel.NewMiniSecretKeyFromRaw(miniSecret)
	if err != nil {
		return nil, fmt.Errorf("sr25519 mini secret: %w", err)
	}
	secret := msk.ExpandEd25519()
	public, err := secret.Public()
	if err != nil {
		return nil, fmt.Errorf("sr25519 public key: %w", err)
	}
	publicKey := public.Encode()
	return &funcSigner{
		algorithm: pb.Signature_SR_25519,
		publicKey: publicKey[:],
		sign: func(message []byte) ([]byte, error) {
			sig, err := secret.Sign(schnorrkel.NewSigningContext(substrateContext, message))
			if err != nil {
				return nil, err
			}
			encoded := sig.Encode()
			return encoded[:], nil
		},
	}, nil
}

// NewSigner adapts any key holder to a Signer: a hardware key, a remote signer,
// or a keypair from another library. sign must produce a signature of the given
// algorithm over the message exactly as passed.
func NewSigner(algorithm pb.Signature_Algorithm, publicKey []byte, sign func(message []byte) ([]byte, error)) Signer {
	return &funcSigner{algorithm: algorithm, publicKey: publicKey, sign: sign}
}

type funcSigner struct {
	algorithm pb.Signature_Algorithm
	publicKey []byte
	sign      func([]byte) ([]byte, error)
}

func (s *funcSigner) Algorithm() pb.Signature_Algorithm   { return s.algorithm }
func (s *funcSigner) PublicKey() []byte                   { return s.publicKey }
func (s *funcSigner) Sign(message []byte) ([]byte, error) { return s.sign(message) }

// SignMessage signs msg, which must not be signed yet, and sets its signature.
// The signature covers msg marshaled with its Signature unset.
func SignMessage(signer Signer, msg pb.SignableMessage) (*pb.Signature, error) {
	if msg.GetSignature() != nil {
		return nil, pb.ErrSignatureNotEmpty
	}
	raw, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	value, err := signer.Sign(raw)
	if err != nil {
		return nil, err
	}
	sig := &pb.Signature{Algorithm: signer.Algorithm(), Signer: signer.PublicKey(), Value: value}
	msg.SetSignature(sig)
	return sig, nil
}

// SignToken signs one level of a token chain with an ed25519 key. It is
// SignMessage with NewEd25519Signer.
func SignToken(key ed25519.PrivateKey, token *pb.AuthToken) error {
	_, err := SignMessage(NewEd25519Signer(key), token)
	return err
}

// VerifySignature checks msg's signature against the signer it names. It
// reports false, with no error, for a well-formed signature that does not
// verify, and ErrUnusableSignature for one that cannot be checked at all. It
// never panics on what arrives off the wire.
//
// This is a self-consistency check, not an authorization: anyone can sign a
// message with their own key. Use Authorizer to check that a token roots in the
// bucket owner.
//
// Like polkadot.js wallets, a signature also verifies when it was made over the
// message wrapped as "<Bytes>" + message + "</Bytes>".
func VerifySignature(msg pb.SignableMessage) (bool, error) {
	if msg == nil {
		return false, ErrUnusableSignature
	}
	sig := msg.GetSignature()
	if sig == nil {
		return false, fmt.Errorf("%w: %w", ErrUnusableSignature, pb.ErrSignatureEmpty)
	}
	if err := checkSignatureUsable(sig); err != nil {
		return false, err
	}

	// Marshal a copy with the signature unset. Clearing it on msg itself would
	// race with any other goroutine reading or verifying the same message.
	unsigned, ok := proto.Clone(msg).(pb.SignableMessage)
	if !ok {
		return false, fmt.Errorf("%w: %T does not clone to a signable message", ErrUnusableSignature, msg)
	}
	unsigned.SetSignature(nil)
	raw, err := proto.Marshal(unsigned)
	if err != nil {
		return false, err
	}

	verify := verifyEd25519
	if sig.GetAlgorithm() == pb.Signature_SR_25519 {
		verify = verifySr25519
	}
	if verify(sig.GetSigner(), raw, sig.GetValue()) {
		return true, nil
	}
	return verify(sig.GetSigner(), wrapBytes(raw), sig.GetValue()), nil
}

func checkSignatureUsable(sig *pb.Signature) error {
	switch sig.GetAlgorithm() {
	case pb.Signature_ED_25519, pb.Signature_SR_25519:
	default:
		return fmt.Errorf("%w: unknown algorithm %d", ErrUnusableSignature, sig.GetAlgorithm())
	}
	if len(sig.GetSigner()) != publicKeySize {
		return fmt.Errorf("%w: signer is %d bytes, want %d", ErrUnusableSignature, len(sig.GetSigner()), publicKeySize)
	}
	if len(sig.GetValue()) != signatureSize {
		return fmt.Errorf("%w: value is %d bytes, want %d", ErrUnusableSignature, len(sig.GetValue()), signatureSize)
	}
	return nil
}

// wrapBytes is how polkadot.js wallets wrap raw bytes before signing them.
// https://github.com/polkadot-js/wasm/issues/256#issuecomment-1002419850
func wrapBytes(message []byte) []byte {
	wrapped := make([]byte, 0, len(message)+len("<Bytes></Bytes>"))
	wrapped = append(wrapped, "<Bytes>"...)
	wrapped = append(wrapped, message...)
	return append(wrapped, "</Bytes>"...)
}

func verifyEd25519(publicKey, message, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}

func verifySr25519(publicKey, message, signature []byte) bool {
	var key [publicKeySize]byte
	copy(key[:], publicKey)
	public := &schnorrkel.PublicKey{}
	if err := public.Decode(key); err != nil {
		return false
	}
	var value [signatureSize]byte
	copy(value[:], signature)
	sig := &schnorrkel.Signature{}
	if err := sig.Decode(value); err != nil {
		return false
	}
	ok, err := public.Verify(sig, schnorrkel.NewSigningContext(substrateContext, message))
	return err == nil && ok
}
