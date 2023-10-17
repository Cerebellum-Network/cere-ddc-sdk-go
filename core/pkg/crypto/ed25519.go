package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vedhavyas/go-subkey"
	ed25519Subkey "github.com/vedhavyas/go-subkey/ed25519"
)

type ed25519Scheme struct {
	keyPair   subkey.KeyPair
	publicKey []byte
}

const Ed25519 SchemeName = "ed25519"

func createEd25519Scheme(seed []byte) (Scheme, error) {
	return createEd25519SchemeFromString(hex.EncodeToString(seed))
}

func createEd25519SchemeFromString(seed string) (Scheme, error) {
	kyr, err := subkey.DeriveKeyPair(ed25519Subkey.Scheme{}, seed)
	if err != nil {
		return nil, err
	}

	return &ed25519Scheme{keyPair: kyr, publicKey: kyr.Public()[:]}, nil
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
	signature, err := e.keyPair.Sign(data)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func (e *ed25519Scheme) Verify(data []byte, signature []byte) bool {
	return verifyEd25519(e.publicKey, data, signature)
}

func (e *ed25519Scheme) Address() (string, error) {
	return subkey.SS58Address(e.publicKey, 42)
}

func (e *ed25519Scheme) AddressForNetwork(network uint8) (string, error) {
	return subkey.SS58Address(e.publicKey, network)
}

func (e *ed25519Scheme) PublicKeyHex() string {
	return fmt.Sprintf("0x%s", hex.EncodeToString(e.publicKey))
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
