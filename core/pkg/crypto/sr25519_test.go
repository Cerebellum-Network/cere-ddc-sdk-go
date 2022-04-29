package crypto

import (
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	privKeySr25519 = "6e40d467e86ec447ae0088c81072feff8c860eebcff7dc44017b1b15746cce0d"

	signatureSr25519 = "04803a00dfbf383d146251dc898167b78719c23a1e2f0b2b20ba2b4b5a714a242042b377c829129d4cfbf5eb51d0ac97ece10e54a0b0d9c2149def4c77f87489"
)

var testSr25519Scheme = initTestSubjectSr25519()

func initTestSubjectSr25519() Scheme {
	decodeString, err := hex.DecodeString(privKeySr25519)
	if err != nil {
		log.Fatal("Failed decode private key sr25519")
	}

	scheme, err := createSr25519Scheme(decodeString)
	if err != nil {
		log.WithError(err).Info("ERROR")
		log.Fatal("Failed create scheme sr25519")
	}

	return scheme
}

func TestContentVerificationWhenSignatureIsValidSr25519(t *testing.T) {
	//when
	result := testSr25519Scheme.Verify([]byte(content), signatureSr25519)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidSr25519(t *testing.T) {
	//when
	result := testSr25519Scheme.Verify([]byte(content+"invalid"), signatureSr25519)

	//then
	assert.False(t, result)
}

func TestSignSr25519(t *testing.T) {
	//when
	sign, err := testSr25519Scheme.Sign([]byte(content))

	//then
	verify := testSr25519Scheme.Verify([]byte(content), sign)
	assert.NoError(t, err)
	assert.True(t, verify)
}
