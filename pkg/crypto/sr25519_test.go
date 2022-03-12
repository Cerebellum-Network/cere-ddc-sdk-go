package crypto

import (
	"encoding/hex"
	"github.com/ChainSafe/go-schnorrkel"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	pubKeySr25519  = "0xfa741f6e23be965b5f316247c2e83449acf1330bd9430036dc0f9e42750bff72"
	privKeySr25519 = "0x6e40d467e86ec447ae0088c81072feff8c860eebcff7dc44017b1b15746cce0d"
)

var testSr25519Scheme = &sr25519Scheme{}

func TestContentVerificationWhenSignatureIsValidSr25519(t *testing.T) {
	//given
	signature := getContentSignatureSr25519(content)

	//when
	result := testSr25519Scheme.Verify(pubKeySr25519, content, signature)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidSr25519(t *testing.T) {
	//given
	signature := getContentSignatureSr25519(content + "invalid")

	//when
	result := testSr25519Scheme.Verify(pubKeySr25519, content, signature)

	//then
	assert.False(t, result)
}

func getContentSignatureSr25519(content string) string {
	hexPrivateKey, _ := hex.DecodeString(strings.TrimPrefix(privKeySr25519, "0x"))
	in := [32]byte{}
	copy(in[:], hexPrivateKey)
	secretKey := &schnorrkel.SecretKey{}
	_ = secretKey.Decode(in)

	transcript := schnorrkel.NewSigningContext(signingContext, []byte(content))
	signature, _ := secretKey.Sign(transcript)
	s := make([]byte, 0)

	for _, b := range signature.Encode() {
		s = append(s, b)
	}
	return "0x" + hex.EncodeToString(s)
}
