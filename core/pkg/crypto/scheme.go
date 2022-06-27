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
		Verify(data []byte, signature string) bool
		Sign(data []byte) (string, error)
		Name() string
		PublicKey() string
	}
)

var ErrSchemeNotExist = errors.New("scheme doesn't exist")

func CreateScheme(schemeName SchemeName, privateKeyHex string) (Scheme, error) {
	privateKey, err := hex.DecodeString(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, err
	}

	switch schemeName {
	case Sr25519, "": // Default.
		return createSr25519Scheme(privateKey)
	case Ed25519:
		return createEd25519Scheme(privateKey), nil
	case Secp256k1:
		return createSecp256k1Scheme(privateKey)
	default:
		return nil, ErrSchemeNotExist
	}
}

func Verify(schemeName SchemeName, publicKeyHex string, content []byte, signature string) (bool, error) {
	switch schemeName {
	case Ed25519:
		return verifyEd25519(publicKeyHex, content, signature), nil
	case Secp256k1:
		return verifySecp256k1(publicKeyHex, content, signature), nil
	case Sr25519:
		return verifySr25519(publicKeyHex, content, signature), nil
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

func encodeSignature(sig []byte) string {
	return "0x" + hex.EncodeToString(sig)
}

func decodeSignature(sig string) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(sig, "0x"))
}

func encodeKey(key []byte) string {
	return "0x" + hex.EncodeToString(key)
}

func decodeKey(key string) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(key, "0x"))
}
