package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	patractTypes "github.com/patractlabs/go-patract/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBucketWriteAccess(t *testing.T) {
	//given
	ss58 := "5GmomkEekQQ3BipMvjDCG5bXKvzwhUDdXEcQqXRWmdkNCYkL"
	publicKey := "0xd049e851567f16d68523a645ee96465ceb678ad983fc08e6a38408ad10410c4d"

	accountID, _ := patractTypes.DecodeAccountIDFromSS58(ss58)
	bucketStatus := &BucketStatus{WriterIds: []types.AccountID{accountID}}

	//when
	hasWriteAccess, err := bucketStatus.HasWriteAccess(publicKey)

	//then
	assert.NoError(t, err)
	assert.True(t, hasWriteAccess)
}
