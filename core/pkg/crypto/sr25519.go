package crypto

import (
	"github.com/ChainSafe/go-schnorrkel"
	log "github.com/sirupsen/logrus"
)

type sr25519Scheme struct {
	privateKey *schnorrkel.SecretKey
	publicKey  string
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

	return &sr25519Scheme{privateKey: secretKey, publicKey: encodeKey(publicKey[:])}, nil
}

func (s *sr25519Scheme) Name() string {
	return string(Sr25519)
}

func (s *sr25519Scheme) PublicKey() string {
	return s.publicKey
}

func (s *sr25519Scheme) Sign(data []byte) (string, error) {
	if err := validateSafeMessage(data); err != nil {
		return "", err
	}
	transcript := schnorrkel.NewSigningContext(signingContext, data)
	signature, err := s.privateKey.Sign(transcript)
	if err != nil {
		return "", err
	}

	result := signature.Encode()
	return encodeSignature(result[:]), nil
}

func (s *sr25519Scheme) Verify(data []byte, signature string) bool {
	return verifySr25519(s.publicKey, data, signature)
}

func verifySr25519(appPubKey string, data []byte, signature string) bool {
	publicKey, err := getSchnorrkelPublicKey(appPubKey)
	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't create Schnorrkel public key")
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
		wrappedContent := "<Bytes>" + string(data) + "</Bytes>"
		transcript = schnorrkel.NewSigningContext(signingContext, []byte(wrappedContent))
		verified, _ = publicKey.Verify(sign, transcript)
	}

	if !verified {
		log.WithField("appPubKey", appPubKey).Info("Invalid content signature")
	}

	return verified
}

func getSchnorrkelPublicKey(appPubKey string) (*schnorrkel.PublicKey, error) {
	publicKeyBytes, err := decodeKey(appPubKey)
	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't decode app pub key (without 0x prefix) to hex")
		return nil, err
	}

	in := [32]byte{}
	copy(in[:], publicKeyBytes)
	publicKey := &schnorrkel.PublicKey{}
	return publicKey, publicKey.Decode(in)
}

func getSchnorrkelSignature(signature string) (*schnorrkel.Signature, error) {
	sigBytes, err := decodeSignature(signature)
	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't decode signature (without 0x prefix) to hex")
		return nil, err
	}

	in := [64]byte{}
	copy(in[:], sigBytes)
	sign := &schnorrkel.Signature{}
	return sign, sign.Decode(in)
}
