package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"strings"
)

type secp256k1Scheme struct {
	privateKey *ecdsa.PrivateKey
	publicKey  string
}

const Secp256k1 SchemeName = "secp256k1"

func (s *secp256k1Scheme) PublicKey() string {
	return s.publicKey
}

func (s *secp256k1Scheme) Name() string {
	return string(Secp256k1)
}

func (s *secp256k1Scheme) Sign(data []byte) (string, error) {
	sign, err := crypto.Sign(data, s.privateKey)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(sign), nil
}

func (s *secp256k1Scheme) Verify(publicKey string, data []byte, signature string) bool {
	hash := crypto.Keccak256Hash(data).Bytes()

	signatureBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't decode signature to hex")
		return false
	}

	publicKeyBytes, err := hex.DecodeString(strings.TrimPrefix(publicKey, "0x"))
	if err != nil {
		log.WithError(err).WithField("publicKey", publicKey).Info("Can't decode app pub key (without 0x prefix) to hex")
		return false
	}

	return crypto.VerifySignature(publicKeyBytes, hash, signatureBytes[:len(signatureBytes)-1])
}
