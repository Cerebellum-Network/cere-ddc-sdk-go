package verification

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"strings"
)

func VerifyContentSecp256k1(publicKey string, content string, signature string) bool {
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
