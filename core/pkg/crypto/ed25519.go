package crypto

import (
	"crypto/ed25519"
	log "github.com/sirupsen/logrus"
)

type ed25519Scheme struct {
	privateKey ed25519.PrivateKey
	publicKey  string
}

const Ed25519 SchemeName = "ed25519"

func createEd25519Scheme(privateKey []byte) Scheme {
	privKey := ed25519.NewKeyFromSeed(privateKey)
	publicKey := encodeKey(privKey.Public().(ed25519.PublicKey))

	return &ed25519Scheme{privateKey: privKey, publicKey: publicKey}
}

func (e *ed25519Scheme) PublicKey() string {
	return e.publicKey
}

func (e *ed25519Scheme) Name() string {
	return string(Ed25519)
}

func (e *ed25519Scheme) Sign(data []byte) (string, error) {
	if err := validateSafeMessage(data); err != nil {
		return "", err
	}
	return encodeSignature(ed25519.Sign(e.privateKey, data)), nil
}

func (e *ed25519Scheme) Verify(data []byte, signature string) bool {
	return verifyEd25519(e.publicKey, data, signature)
}

func verifyEd25519(appPubKey string, data []byte, signature string) bool {
	hexSignature, err := decodeSignature(signature)

	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't decode signature to hex")
		return false
	}

	publicKey, err := decodeKey(appPubKey)

	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't decode app pub key (without 0x prefix) to hex")
		return false
	}

	verified := ed25519.Verify(publicKey, data, hexSignature)

	if !verified {
		wrappedContent := "<Bytes>" + string(data) + "</Bytes>"
		verified = ed25519.Verify(publicKey, []byte(wrappedContent), hexSignature)
	}

	if !verified {
		log.WithField("appPubKey", appPubKey).Info("Invalid content signature")
	}

	return verified
}
