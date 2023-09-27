package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	pubKeyEd25519  = "8f01969eb5244d853cc9c6ad73c46d8a1a091842c414cabd2377531f0832635f"
	privKeyEd25519 = "38a538d3d890bfe8f76dc9bf578e215af16fd3d684666f72db0bc0a22bc1d05b"
)

var pubKeyEd25519Bytes []byte

func init() {
	pubKeyEd25519Bytes, _ = hex.DecodeString(strings.TrimPrefix(pubKeyEd25519, "0x"))
}

var testEd25519Scheme = initTestSubjectEd25519()

func initTestSubjectEd25519() Scheme {
	decodeString, err := hex.DecodeString(privKeyEd25519)
	if err != nil {
		log.Fatal("Failed decode private key ed25519")
	}

	scheme, err := createEd25519Scheme(decodeString)
	if err != nil {
		log.WithError(err).Fatal("Failed create scheme ed25519")
	}

	return scheme
}

func TestPublicKeyEd25519(t *testing.T) {
	//when
	publicKey := testEd25519Scheme.PublicKey()

	//then
	assert.Equal(t, pubKeyEd25519Bytes, publicKey)
}

func TestNameEd25519(t *testing.T) {
	//when
	name := testEd25519Scheme.Name()

	//then
	assert.Equal(t, "ed25519", name)
}

func TestSignEd25519(t *testing.T) {
	//when
	signature, err := testEd25519Scheme.Sign([]byte(content))

	//then
	expected := "464fc53d45cc95e7bdbac954ae21bd8831cbe059f8f438c0f367f57a7ad7a47f56ca32b15c084b6ad81b91e6122984eaaff0f47280f3115294df8f83dd959e0a"
	expectedB, _ := hex.DecodeString(strings.TrimPrefix(expected, "0x"))
	assert.Equal(t, expectedB, signature)
	assert.NoError(t, err)
}

func TestContentVerificationWhenSignatureIsValidEd25519(t *testing.T) {
	//given
	signature := getContentSignatureEd25519(content)

	//when
	result := testEd25519Scheme.Verify([]byte(content), signature)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidEd25519(t *testing.T) {
	//given
	signature := getContentSignatureEd25519(content + "invalid")

	//when
	result := testEd25519Scheme.Verify([]byte(content), signature)

	//then
	assert.False(t, result)
}

func TestAddressEd25519Scheme(t *testing.T) {
	addr, err := testEd25519Scheme.Address()
	assert.NoError(t, err)
	assert.Equal(t, address, addr)
}

func TestAddressForCereNetworkEd25519Scheme(t *testing.T) {
	addr, err := testEd25519Scheme.AddressForNetwork(56)
	assert.NoError(t, err)
	assert.Equal(t, addressForCereNetwork, addr)
}

func TestPublicKeyHexEd25519Scheme(t *testing.T) {
	keyHex := testEd25519Scheme.PublicKeyHex()
	assert.Equal(t, pubKeyHex, keyHex)
}

func getContentSignatureEd25519(content string) []byte {
	privKeyAsBytes, _ := hex.DecodeString(strings.TrimPrefix(privKeyEd25519, "0x"))
	privKey := ed25519.NewKeyFromSeed(privKeyAsBytes)
	return ed25519.Sign(privKey, []byte(content))
}
