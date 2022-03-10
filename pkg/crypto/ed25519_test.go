package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	pubKeyEd25519  = "0x8f01969eb5244d853cc9c6ad73c46d8a1a091842c414cabd2377531f0832635f"
	privKeyEd25519 = "0x38a538d3d890bfe8f76dc9bf578e215af16fd3d684666f72db0bc0a22bc1d05b"
)

var testEd25519Scheme = &ed25519Scheme{}

func TestContentVerificationWhenSignatureIsValidEd25519(t *testing.T) {
	//given
	signature := getContentSignatureEd25519(content)

	//when
	result := testEd25519Scheme.Verify(pubKeyEd25519, content, signature)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidEd25519(t *testing.T) {
	//given
	signature := getContentSignatureEd25519(content + "invalid")

	//when
	result := testEd25519Scheme.Verify(pubKeyEd25519, content, signature)

	//then
	assert.False(t, result)
}

func getContentSignatureEd25519(content string) string {
	privKeyAsBytes, _ := hex.DecodeString(strings.TrimPrefix(privKeyEd25519, "0x"))
	privKey := ed25519.NewKeyFromSeed(privKeyAsBytes)
	return hex.EncodeToString(ed25519.Sign(privKey, []byte(content)))
}
