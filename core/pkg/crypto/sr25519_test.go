package crypto

import (
	"encoding/hex"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	privKeySr25519               = "6e40d467e86ec447ae0088c81072feff8c860eebcff7dc44017b1b15746cce0d"
	pubKeyHexSr25519             = "0xc8393bc5fd86bccda163187c8c23cba9d63622a745ed59fedc51d51210d1884f"
	signatureSr25519             = "ba4a02f174694ee29a6a38b4ad2c16ff59f590da38694b556299197d1b352f464db889d93d1e3d86b068522aabb3585f50c129e1565a48b653336014a5fe158f"
	addressSr25519               = "5GbESExKAqUSer8YHHRpze6XDGjRaC2dpS7E44vMiXmEHoSe"
	addressForCereNetworkSr25519 = "6fcU48XxYuicqfo2xjkAHFpUG4jYLkqvTbs4fdvaCW64EVhQ"
)

var (
	signatureSr25519Bytes []byte
	pubKeyKeySr25519Bytes []byte
)

func init() {
	signatureSr25519Bytes, _ = hex.DecodeString(strings.TrimPrefix(signatureSr25519, "0x"))
	pubKeyKeySr25519Bytes, _ = hex.DecodeString(strings.TrimPrefix(pubKeyHexSr25519, "0x"))
}

var testSr25519Scheme = initTestSubjectSr25519()

func initTestSubjectSr25519() Scheme {
	decodeString, err := hex.DecodeString(privKeySr25519)
	if err != nil {
		log.Fatal("Failed decode private key sr25519")
	}

	scheme, err := createSr25519Scheme(decodeString)
	if err != nil {
		log.WithError(err).Fatal("Failed create scheme sr25519")
	}

	return scheme
}

func TestContentVerificationWhenSignatureIsValidSr25519(t *testing.T) {
	//when
	result := testSr25519Scheme.Verify([]byte(content), signatureSr25519Bytes)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidSr25519(t *testing.T) {
	//when
	result := testSr25519Scheme.Verify([]byte(content+"invalid"), signatureSr25519Bytes)

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

func TestAddressSr25519(t *testing.T) {
	address, err := testSr25519Scheme.Address()
	assert.NoError(t, err)
	assert.Equal(t, addressSr25519, address)
}

func TestAddressForCereNetworkSr25519(t *testing.T) {
	address, err := testSr25519Scheme.AddressForNetwork(56)
	assert.NoError(t, err)
	assert.Equal(t, address, addressForCereNetworkSr25519)
}

func TestPublicKeyHexSr25519(t *testing.T) {
	publicKeyHex := testSr25519Scheme.PublicKeyHex()
	assert.Equal(t, pubKeyHexSr25519, publicKeyHex)
}

func TestPublicKeySr25519(t *testing.T) {
	expPublicKey := testSr25519Scheme.PublicKey()
	assert.Equal(t, pubKeyKeySr25519Bytes, expPublicKey)
}
