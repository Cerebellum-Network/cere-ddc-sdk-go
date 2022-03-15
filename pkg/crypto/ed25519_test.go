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

var testEd25519Scheme = initTestSubjectEd25519()

func initTestSubjectEd25519() Scheme {
	decodeString, err := hex.DecodeString(privKeyEd25519)
	if err != nil {
		log.Fatal("Failed decode private key ed25519")
	}
	return createEd25519Scheme(decodeString)
}

func TestPublicKeyEd25519(t *testing.T) {
	//when
	publicKey := testEd25519Scheme.PublicKey()

	//then
	assert.Equal(t, pubKeyEd25519, publicKey)
}

func TestNameEd25519(t *testing.T) {
	//when
	name := testEd25519Scheme.Name()

	//then
	assert.Equal(t, "ed25519", name)
}

func TestSignEd25519(t *testing.T) {
	//when
	signature := testEd25519Scheme.Sign([]byte(content))

	//then
	expected := "464fc53d45cc95e7bdbac954ae21bd8831cbe059f8f438c0f367f57a7ad7a47f56ca32b15c084b6ad81b91e6122984eaaff0f47280f3115294df8f83dd959e0a"
	assert.Equal(t, expected, signature)
}

func TestContentVerificationWhenSignatureIsValidEd25519(t *testing.T) {
	//given
	signature := getContentSignatureEd25519(content)

	//when
	result := testEd25519Scheme.Verify(pubKeyEd25519, []byte(content), signature)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidEd25519(t *testing.T) {
	//given
	signature := getContentSignatureEd25519(content + "invalid")

	//when
	result := testEd25519Scheme.Verify(pubKeyEd25519, []byte(content), signature)

	//then
	assert.False(t, result)
}

func getContentSignatureEd25519(content string) string {
	privKeyAsBytes, _ := hex.DecodeString(strings.TrimPrefix(privKeyEd25519, "0x"))
	privKey := ed25519.NewKeyFromSeed(privKeyAsBytes)
	return hex.EncodeToString(ed25519.Sign(privKey, []byte(content)))
}
