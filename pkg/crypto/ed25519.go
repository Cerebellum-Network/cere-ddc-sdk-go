package verification

import (
	"crypto/ed25519"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"strings"
)

func VerifyContentEd25519(appPubKey string, content string, signature string) bool {
	hexSignature, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))

	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't decode signature to hex")
		return false
	}

	publicKey, err := hex.DecodeString(strings.TrimPrefix(appPubKey, "0x"))

	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't decode app pub key (without 0x prefix) to hex")
		return false
	}

	verified := ed25519.Verify(publicKey, []byte(content), hexSignature)

	if !verified {
		wrappedContent := "<Bytes>" + content + "</Bytes>"
		verified = ed25519.Verify(publicKey, []byte(wrappedContent), hexSignature)
	}

	if !verified {
		log.WithField("appPubKey", appPubKey).Info("Invalid content signature")
	}

	return verified
}
