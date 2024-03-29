package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vedhavyas/go-subkey"
)

type secp256k1Scheme struct {
	privateKey *ecdsa.PrivateKey
	publicKey  []byte
}

func (s *secp256k1Scheme) Address() (string, error) {
	return subkey.SS58Address(s.publicKey, 42)
}

func (s *secp256k1Scheme) AddressForNetwork(network uint8) (string, error) {
	return subkey.SS58Address(s.publicKey, network)
}

func (s *secp256k1Scheme) PublicKeyHex() string {
	return fmt.Sprintf("0x%s", hex.EncodeToString(s.publicKey))
}

const Secp256k1 SchemeName = "secp256k1"

func createSecp256k1Scheme(privateKey []byte) (Scheme, error) {
	privKey, err := crypto.ToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	publicKey := elliptic.MarshalCompressed(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)

	return &secp256k1Scheme{privateKey: privKey, publicKey: publicKey}, nil
}

func (s *secp256k1Scheme) PublicKey() []byte {
	return s.publicKey
}

func (s *secp256k1Scheme) Name() string {
	return string(Secp256k1)
}

func (s *secp256k1Scheme) Sign(data []byte) ([]byte, error) {
	return crypto.Sign(crypto.Keccak256Hash(data).Bytes(), s.privateKey)
}

func (s *secp256k1Scheme) Verify(data []byte, signature []byte) bool {
	return verifySecp256k1(s.publicKey, data, signature)
}

func verifySecp256k1(publicKey []byte, data []byte, signature []byte) bool {
	hash := crypto.Keccak256Hash(data).Bytes()

	return crypto.VerifySignature(publicKey, hash, signature[:len(signature)-1])
}
