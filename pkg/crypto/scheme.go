package crypto

import (
	"encoding/hex"
	"errors"
)

type (
	SchemeName string

	Scheme interface {
		Verify(pubKey string, data []byte, signature string) bool
		Sign(data []byte) (string, error)
		Name() string
		PublicKey() string
	}
)

var ErrSchemeNotExist = errors.New("scheme doesn't exist")

func CreateScheme(schemeName SchemeName, privateKeyHex string) (Scheme, error) {
	privateKey, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, err
	}

	switch schemeName {
	case Ed25519:
		return createEd25519Scheme(privateKey), nil
	case Secp256k1:
		return nil, nil
	case Sr25519:
		return nil, nil
	default:
		return nil, ErrSchemeNotExist
	}
}
