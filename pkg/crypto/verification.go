package verification

import "errors"

type (
	SignatureVerifyAlgorithm func(pubKey string, content string, signature string) bool
	SignatureScheme          string

	Verifier interface {
		//TODO use domain.Signature
		Verify(algorithm SignatureScheme, pubKey string, content string, signature string) (bool, error)
	}
	verifier struct {
		verifyAlgorithms map[SignatureScheme]SignatureVerifyAlgorithm
	}
)

const (
	Ed25519   SignatureScheme = "ed25519"
	Secp256k1 SignatureScheme = "secp256k1"
	Sr25519   SignatureScheme = "sr25519"
)

var ErrVerifierNotSupportedAlgorithm = errors.New("signature verifier doesn't support current algorithm")

func CreateVerifier(algorithms ...SignatureScheme) (Verifier, error) {
	verifyAlgorithms := map[SignatureScheme]SignatureVerifyAlgorithm{}

	for _, algorithmName := range algorithms {
		verifyAlgorithm := algorithm(algorithmName)
		if verifyAlgorithm == nil {
			return nil, ErrVerifierNotSupportedAlgorithm
		}
		verifyAlgorithms[algorithmName] = verifyAlgorithm
	}

	return &verifier{verifyAlgorithms: verifyAlgorithms}, nil
}

func (v verifier) Verify(algorithm SignatureScheme, pubKey string, content string, signature string) (bool, error) {
	verifyAlgorithms, ok := v.verifyAlgorithms[algorithm]
	if !ok {
		return false, ErrVerifierNotSupportedAlgorithm
	}

	return verifyAlgorithms(pubKey, content, signature), nil
}

func algorithm(signatureScheme SignatureScheme) SignatureVerifyAlgorithm {
	switch signatureScheme {
	case Ed25519:
		return VerifyContentEd25519
	case Secp256k1:
		return VerifyContentSecp256k1
	case Sr25519:
		return VerifyContentSr25519
	default:
		return nil
	}
}
