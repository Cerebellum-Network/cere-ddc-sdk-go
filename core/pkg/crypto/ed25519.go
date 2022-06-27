package crypto

import (
	"crypto/ed25519"
	log "github.com/sirupsen/logrus"
)

type ed25519Scheme struct {
	privateKey ed25519.PrivateKey
	publicKey  []byte
}

const Ed25519 SchemeName = "ed25519"

func createEd25519Scheme(privateKey []byte) Scheme {
	privKey := ed25519.NewKeyFromSeed(privateKey)
	publicKey := privKey.Public().(ed25519.PublicKey)

	return &ed25519Scheme{privateKey: privKey, publicKey: publicKey}
}

func (e *ed25519Scheme) PublicKey() []byte {
	return e.publicKey
}

func (e *ed25519Scheme) Name() string {
	return string(Ed25519)
}

func (e *ed25519Scheme) Sign(data []byte) ([]byte, error) {
	if err := validateSafeMessage(data); err != nil {
		return nil, err
	}
	return ed25519.Sign(e.privateKey, data), nil
}

func (e *ed25519Scheme) Verify(data []byte, signature []byte) bool {
	return verifyEd25519(e.publicKey, data, signature)
}

func verifyEd25519(publicKey []byte, data []byte, signature []byte) bool {
	verified := ed25519.Verify(publicKey, data, signature)

	if !verified {
		// Try the Polkadot.js format (https://github.com/polkadot-js/wasm/issues/256#issuecomment-1002419850)
		wrappedContent := "<Bytes>" + string(data) + "</Bytes>"
		verified = ed25519.Verify(publicKey, []byte(wrappedContent), signature)
	}

	if !verified {
		log.WithField("publicKey", publicKey).Info("Invalid content signature")
	}

	return verified
}
