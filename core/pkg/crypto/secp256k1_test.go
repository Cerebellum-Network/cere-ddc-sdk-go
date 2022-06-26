package crypto

import (
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	privKeySecp256k1   = "fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19"
	signatureSecp256k1 = "0xdc5b199958dc51bd2924754f1c2d4908ec7a7bd2b8ff7b55cf6c88e5315adbba0c033d77414650f17ffd702863f33709972d647aef2c3b3dd0378a5d39c4685801"
)

var testSecp256k1Scheme = initTestSubjectSecp256k1()

func initTestSubjectSecp256k1() Scheme {
	decodeString, err := hex.DecodeString(privKeySecp256k1)
	if err != nil {
		log.Fatal("Failed decode private key secp256k1")
	}

	scheme, err := createSecp256k1Scheme(decodeString)
	if err != nil {
		log.Fatal("Failed create scheme secp256k1")
	}

	return scheme
}

func TestContentVerificationWhenSignatureIsValidSecp256k1(t *testing.T) {
	//when
	result := testSecp256k1Scheme.Verify([]byte(content), signatureSecp256k1)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidSecp256k1(t *testing.T) {
	//when
	result := testSecp256k1Scheme.Verify([]byte(content+"invalid"), signatureSecp256k1)

	//then
	assert.False(t, result)
}

func TestSignSecp256k1(t *testing.T) {
	//when
	sign, err := testSecp256k1Scheme.Sign([]byte(content))

	//then
	assert.NoError(t, err)
	assert.Equal(t, signatureSecp256k1, sign)
}
