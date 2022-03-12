package crypto

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"strings"
)

type secp256k1Scheme struct {
}

const Secp256k1 SchemeName = "secp256k1"

func (s *secp256k1Scheme) Name() SchemeName {
	return Secp256k1
}

func (s *secp256k1Scheme) Verify(publicKey string, content string, signature string) bool {
	contentBytes := []byte(content)
	hash := crypto.Keccak256Hash(contentBytes).Bytes()

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
