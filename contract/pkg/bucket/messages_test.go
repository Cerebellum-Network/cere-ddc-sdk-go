package bucket

import (
	"encoding/hex"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"strings"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestBucketWriteAccess(t *testing.T) {
	//given
	ss58 := "5GmomkEekQQ3BipMvjDCG5bXKvzwhUDdXEcQqXRWmdkNCYkL"
	publicKey := "0xd049e851567f16d68523a645ee96465ceb678ad983fc08e6a38408ad10410c4d"
	publicKeyB, _ := hex.DecodeString(strings.TrimPrefix(publicKey, "0x"))

	accountID, _ := pkg.DecodeAccountIDFromSS58(ss58)
	bucketStatus := &BucketStatus{WriterIds: []types.AccountID{accountID}}

	//when
	hasWriteAccess, err := bucketStatus.HasWriteAccess(publicKeyB)

	//then
	assert.NoError(t, err)
	assert.True(t, hasWriteAccess)
}
