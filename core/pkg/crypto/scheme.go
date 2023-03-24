package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type (
	SchemeName string

	Scheme interface {
		Verify(data []byte, signature []byte) bool
		Sign(data []byte) ([]byte, error)
		Name() string
		PublicKey() []byte
		Address() (string, error)
		PublicKeyHex() string
	}
)

var ErrSchemeNotExist = errors.New("scheme doesn't exist")

func CreateScheme(schemeName SchemeName, seed string) (Scheme, error) {
	switch schemeName {
	case Sr25519, "": // Default.
		return createSr25519SchemeFromString(seed)
	case Ed25519:
		return createEd25519SchemeFromString(seed)
	case Secp256k1:
		privateKey, err := hex.DecodeString(strings.TrimPrefix(seed, "0x"))
		if err != nil {
			return nil, err
		}
		return createSecp256k1Scheme(privateKey)
	default:
		return nil, ErrSchemeNotExist
	}
}

func Verify(schemeName SchemeName, publicKey []byte, content []byte, signature []byte) (bool, error) {
	switch schemeName {
	case Sr25519, "": // Default.
		return verifySr25519(publicKey, content, signature), nil
	case Ed25519:
		return verifyEd25519(publicKey, content, signature), nil
	case Secp256k1:
		return verifySecp256k1(publicKey, content, signature), nil
	default:
		return false, ErrSchemeNotExist
	}
}

// Validate that the signed data does not conflict with the blockchain extrinsics.
func validateSafeMessage(data []byte) error {
	// Encoded extrinsics start with the pallet index; reserve up to 48 pallets.
	// Make ASCII "0" the smallest first valid byte.
	if len(data) > 0 && data[0] < 48 {
		return fmt.Errorf("data unsafe to sign")
	}
	return nil
}
