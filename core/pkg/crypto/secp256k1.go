package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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

func createSecp256k1Scheme(privateKey []byte) (Scheme, error) {
	privKey, err := crypto.ToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	publicKey := hex.EncodeToString(elliptic.MarshalCompressed(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y))

	return &secp256k1Scheme{privateKey: privKey, publicKey: publicKey}, nil
}

func (s *secp256k1Scheme) PublicKey() string {
	return s.publicKey
}

func (s *secp256k1Scheme) Name() string {
	return string(Secp256k1)
}

func (s *secp256k1Scheme) Sign(data []byte) (string, error) {
	sign, err := crypto.Sign(crypto.Keccak256Hash(data).Bytes(), s.privateKey)
	if err != nil {
		return "", err
	}

	return encodeSignature(sign), nil
}

func (s *secp256k1Scheme) Verify(data []byte, signature string) bool {
	return verifySecp256k1(s.publicKey, data, signature)
}

func verifySecp256k1(publicKey string, data []byte, signature string) bool {
	hash := crypto.Keccak256Hash(data).Bytes()

	signatureBytes, err := decodeSignature(signature)
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
