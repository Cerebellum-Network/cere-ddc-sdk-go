package crypto

import (
	"github.com/ChainSafe/go-schnorrkel"
	log "github.com/sirupsen/logrus"
)

type sr25519Scheme struct {
	privateKey *schnorrkel.SecretKey
	publicKey  []byte
}

const Sr25519 SchemeName = "sr25519"

var signingContext = []byte("substrate")

func createSr25519Scheme(privateKey []byte) (Scheme, error) {
	key := [schnorrkel.SecretKeySize]byte{}
	nonce := [schnorrkel.PublicKeySize]byte{}

	copy(key[:], privateKey[:32])
	copy(nonce[:], privateKey[32:])

	secretKey := schnorrkel.NewSecretKey(key, nonce)
	public, err := secretKey.Public()
	if err != nil {
		return nil, err
	}

	publicKey := public.Encode()

	return &sr25519Scheme{privateKey: secretKey, publicKey: publicKey[:]}, nil
}

func (s *sr25519Scheme) Name() string {
	return string(Sr25519)
}

func (s *sr25519Scheme) PublicKey() []byte {
	return s.publicKey
}

func (s *sr25519Scheme) Sign(data []byte) ([]byte, error) {
	if err := validateSafeMessage(data); err != nil {
		return nil, err
	}
	transcript := schnorrkel.NewSigningContext(signingContext, data)
	signature, err := s.privateKey.Sign(transcript)
	if err != nil {
		return nil, err
	}

	result := signature.Encode()
	return result[:], nil
}

func (s *sr25519Scheme) Verify(data []byte, signature []byte) bool {
	return verifySr25519(s.publicKey, data, signature)
}

func verifySr25519(pubKey []byte, data []byte, signature []byte) bool {
	publicKey, err := getSchnorrkelPublicKey(pubKey)
	if err != nil {
		log.WithError(err).WithField("publicKey", pubKey).Info("Can't create Schnorrkel public key")
		return false
	}

	sign, err := getSchnorrkelSignature(signature)
	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't create Schnorrkel signature")
		return false
	}

	transcript := schnorrkel.NewSigningContext(signingContext, data)
	verified, _ := publicKey.Verify(sign, transcript)

	if !verified {
		// Try the Polkadot.js format (https://github.com/polkadot-js/wasm/issues/256#issuecomment-1002419850)
		wrappedContent := "<Bytes>" + string(data) + "</Bytes>"
		transcript = schnorrkel.NewSigningContext(signingContext, []byte(wrappedContent))
		verified, _ = publicKey.Verify(sign, transcript)
	}

	if !verified {
		log.WithField("appPubKey", pubKey).Info("Invalid content signature")
	}

	return verified
}

func getSchnorrkelPublicKey(pubKey []byte) (*schnorrkel.PublicKey, error) {
	in := [32]byte{}
	copy(in[:], pubKey)
	publicKey := &schnorrkel.PublicKey{}
	return publicKey, publicKey.Decode(in)
}

func getSchnorrkelSignature(signature []byte) (*schnorrkel.Signature, error) {
	in := [64]byte{}
	copy(in[:], signature)
	sign := &schnorrkel.Signature{}
	return sign, sign.Decode(in)
}
