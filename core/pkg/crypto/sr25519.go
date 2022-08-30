package crypto

import (
	"encoding/hex"
	"github.com/ChainSafe/go-schnorrkel"
	log "github.com/sirupsen/logrus"
	"github.com/vedhavyas/go-subkey"
	"github.com/vedhavyas/go-subkey/sr25519"
)

type sr25519Scheme struct {
	keyPair   subkey.KeyPair
	publicKey []byte
}

const Sr25519 SchemeName = "sr25519"

var signingContext = []byte("substrate")

func createSr25519Scheme(seed []byte) (Scheme, error) {
	return createSr25519SchemeFromString(hex.EncodeToString(seed))
}

func createSr25519SchemeFromString(seed string) (Scheme, error) {
	kyr, err := subkey.DeriveKeyPair(sr25519.Scheme{}, seed)
	if err != nil {
		return nil, err
	}

	return &sr25519Scheme{keyPair: kyr, publicKey: kyr.Public()[:]}, nil
}

func (s *sr25519Scheme) Name() string {
	return string(Sr25519)
}

func (s *sr25519Scheme) PublicKey() []byte {
	return s.publicKey[:]
}

func (s *sr25519Scheme) Sign(data []byte) ([]byte, error) {
	if err := validateSafeMessage(data); err != nil {
		return nil, err
	}
	signature, err := s.keyPair.Sign(data)
	if err != nil {
		return nil, err
	}

	return signature, nil
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
