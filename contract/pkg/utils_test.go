package pkg

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecodeAccountIDFromSS58(t *testing.T) {
	//given
	ss58 := "5GmomkEekQQ3BipMvjDCG5bXKvzwhUDdXEcQqXRWmdkNCYkL"
	publicKey, _ := hex.DecodeString("d049e851567f16d68523a645ee96465ceb678ad983fc08e6a38408ad10410c4d")

	//when
	accountID, _ := DecodeAccountIDFromSS58(ss58)

	//then
	assert.Equal(t, publicKey, accountID[:])
}
