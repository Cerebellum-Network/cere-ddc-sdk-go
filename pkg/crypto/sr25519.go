package crypto

import (
	"encoding/hex"
	"github.com/ChainSafe/go-schnorrkel"
	log "github.com/sirupsen/logrus"
	"strings"
)

type sr25519Scheme struct {
}

const Sr25519 SchemeName = "sr25519"

var signingContext = []byte("substrate")

func (s sr25519Scheme) Name() string {
	return string(Sr25519)
}

func (s sr25519Scheme) Sign(data []byte) (string, error) {
	panic("implement me")
}

func (s sr25519Scheme) Verify(appPubKey string, data []byte, signature string) bool {
	publicKey, err := getSchnorrkelPublicKey(appPubKey)
	if err != nil {
		log.WithError(err).WithField("appPubKey", appPubKey).Info("Can't create Schnorrkel public key")
		return false
	}

	sign, err := getSchnorrkelSignature(signature)
	if err != nil {
		log.WithError(err).WithField("signature", signature).Info("Can't create Schnorrkel signature")
		return false
	}

	transcript := schnorrkel.NewSigningContext(signingContext, data)
	verified, _ := publicKey.Verify(sign, transcript)

	if !verified {
		wrappedContent := "<Bytes>" + string(data) + "</Bytes>"
		transcript = schnorrkel.NewSigningContext(signingContext, []byte(wrappedContent))
		verified, _ = publicKey.Verify(sign, transcript)
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
	sign := &schnorrkel.Signature{}
	return sign, sign.Decode(in)
}
