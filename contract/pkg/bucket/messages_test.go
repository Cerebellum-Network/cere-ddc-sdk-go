package bucket

import (
	"encoding/hex"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/utils"
	"strings"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/stretchr/testify/assert"
)

func TestBucketWriteAccess(t *testing.T) {
	//given
	ss58 := "5GmomkEekQQ3BipMvjDCG5bXKvzwhUDdXEcQqXRWmdkNCYkL"
	publicKey := "0xd049e851567f16d68523a645ee96465ceb678ad983fc08e6a38408ad10410c4d"
	publicKeyB, _ := hex.DecodeString(strings.TrimPrefix(publicKey, "0x"))

	accountID, _ := utils.DecodeAccountIDFromSS58(ss58)
	bucketStatus := &BucketStatus{WriterIds: []types.AccountID{accountID}}

	//when
	hasWriteAccess := bucketStatus.HasWriteAccess(publicKeyB)

	//then
	assert.True(t, hasWriteAccess)
}

func TestClusterStatus_ReplicationFactor(t *testing.T) {
	type fields struct {
		ClusterId ClusterId
		Cluster   Cluster
		Params    Params
	}
	tests := []struct {
		name   string
		fields fields
		want   uint
	}{
		{
			name: "ReplicationFactor as integer",
			fields: fields{
				ClusterId: 1,
				Params:    `{"ReplicationFactor": 3}`,
				Cluster:   Cluster{},
			},
			want: 3,
		},
		{
			name: "ReplicationFactor as string",
			fields: fields{
				ClusterId: 1,
				Params:    `{"ReplicationFactor": "3"}`,
				Cluster:   Cluster{},
			},
			want: 3,
		},
		{
			name: "ReplicationFactor not set",
			fields: fields{
				ClusterId: 1,
				Params:    `{}`,
				Cluster:   Cluster{},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterStatus{
				ClusterId: tt.fields.ClusterId,
				Cluster:   tt.fields.Cluster,
				Params:    tt.fields.Params,
			}
			assert.Equalf(t, tt.want, c.ReplicationFactor(), "ReplicationFactor()")
		})
	}
}
