package crypto

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	pubKeyEthereum           = "0x048e66b3e549818ea2cb354fb70749f6c8de8fa484f7530fc447d5fe80a1c424e4f5ae648d648c980ae7095d1efad87161d83886ca4b6c498ac22a93da5099014a"
	compressedPubKeyEthereum = "0x028e66b3e549818ea2cb354fb70749f6c8de8fa484f7530fc447d5fe80a1c424e4"
	content                  = "Hello world!"
)

var testSecp256k1Scheme = &secp256k1Scheme{}

func TestContentVerificationWhenSignatureIsValidEthereum(t *testing.T) {
	//given
	signature := "0x7655c23866338f3a73bc0d416504424715328828f082b8e162d5670a84103e6508e3a68f14bdc6ec27c4b364d819f744fcfbf006d12eb6dd5b7109615ee273c701"

	//when
	result := testSecp256k1Scheme.Verify(pubKeyEthereum, content, signature)

	//then
	assert.True(t, result)
}

func TestContentWithCompressedPubLeyVerificationWhenSignatureIsValidEthereum(t *testing.T) {
	//given
	signature := "0x7655c23866338f3a73bc0d416504424715328828f082b8e162d5670a84103e6508e3a68f14bdc6ec27c4b364d819f744fcfbf006d12eb6dd5b7109615ee273c701"

	//when
	result := testSecp256k1Scheme.Verify(compressedPubKeyEthereum, content, signature)

	//then
	assert.True(t, result)
}

func TestContentVerificationWhenSignatureIsInvalidEthereum(t *testing.T) {
	//given
	signature := "0x789a80053e4927d0a898db8e065e948f5cf086e32f9ccaa54c1908e22ac430c62621578113ddbb62d509bf6049b8fb544ab06d36f916685a2eb8e57ffadde02301"

	//when
	result := testSecp256k1Scheme.Verify(pubKeyEthereum, content, signature)

	//then
	assert.False(t, result)
}
