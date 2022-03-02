package verification

import (
	"encoding/hex"
	"github.com/ChainSafe/go-schnorrkel"
	log "github.com/sirupsen/logrus"
	"strings"
)

var SigningContext = []byte("substrate")

func VerifyContentSr25519(appPubKey string, content string, signature string) bool {
	publicKey, err := getSchnorrkelPublicKey(appPubKey)
	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't create Schnorrkel public key")
		return false
	}

	s, err := getSchnorrkelSignature(signature)
	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't create Schnorrkel signature")
		return false
	}

	transcript := schnorrkel.NewSigningContext(SigningContext, []byte(content))
	verified, _ := publicKey.Verify(s, transcript)

	if !verified {
		wrappedContent := "<Bytes>" + content + "</Bytes>"
		transcript = schnorrkel.NewSigningContext(SigningContext, []byte(wrappedContent))
		verified, _ = publicKey.Verify(s, transcript)
	}

	if !verified {
		log.WithField("appPubKey", appPubKey).Info("Invalid content signature")
	}

	return verified
}

func getSchnorrkelPublicKey(appPubKey string) (*schnorrkel.PublicKey, error) {
	hexPublicKey, err := hex.DecodeString(strings.TrimPrefix(appPubKey, "0x"))
	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't decode app pub key (without 0x prefix) to hex")
		return nil, err
	}

	in := [32]byte{}
	copy(in[:], hexPublicKey)
	publicKey := &schnorrkel.PublicKey{}
	return publicKey, publicKey.Decode(in)
}

func getSchnorrkelSignature(signature string) (*schnorrkel.Signature, error) {
	hexSignature, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't decode signature (without 0x prefix) to hex")
		return nil, err
	}

	in := [64]byte{}
	copy(in[:], hexSignature)
	s := &schnorrkel.Signature{}
	return s, s.Decode(in)
}
