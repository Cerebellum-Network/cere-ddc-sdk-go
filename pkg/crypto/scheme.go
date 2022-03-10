package crypto

import "errors"

type (
	SchemeName string

	Scheme interface {
		Verify(pubKey string, content string, signature string) bool
		Name() SchemeName
	}
)

var ErrSchemeNotExist = errors.New("scheme doesn't exist")

func CreateScheme(schemeName SchemeName) (Scheme, error) {
	switch schemeName {
	case Ed25519:
		return &ed25519Scheme{}, nil
	case Secp256k1:
		return &secp256k1Scheme{}, nil
	case Sr25519:
		return &sr25519Scheme{}, nil
	default:
		return nil, ErrSchemeNotExist
	}
}
