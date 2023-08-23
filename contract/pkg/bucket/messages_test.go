package bucket

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/stretchr/testify/assert"
)

func TestBucketWriteAccess(t *testing.T) {
	//given
	ss58 := "5GmomkEekQQ3BipMvjDCG5bXKvzwhUDdXEcQqXRWmdkNCYkL"
	publicKey := "0xd049e851567f16d68523a645ee96465ceb678ad983fc08e6a38408ad10410c4d"
	publicKeyB, _ := hex.DecodeString(strings.TrimPrefix(publicKey, "0x"))

	accountID, _ := pkg.DecodeAccountIDFromSS58(ss58)
	bucketStatus := &BucketInfo{WriterIds: []types.AccountID{accountID}}

	//when
	hasWriteAccess := bucketStatus.HasWriteAccess(publicKeyB)

	//then
	assert.True(t, hasWriteAccess)
}

func TestClusterStatus_ReplicationFactor(t *testing.T) {
	type fields struct {
		ClusterId   ClusterId
		Cluster     Cluster
		NodesVNodes []NodeVNodesInfo
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
				Cluster: Cluster{
					Params: `{"ReplicationFactor": 3}`,
				},
			},
			want: 3,
		},
		{
			name: "ReplicationFactor as string",
			fields: fields{
				ClusterId: 1,
				Cluster: Cluster{
					Params: `{"ReplicationFactor": "3"}`,
				},
			},
			want: 3,
		},
		{
			name: "ReplicationFactor not set",
			fields: fields{
				ClusterId: 1,
				Cluster: Cluster{
					Params: `{}`,
				},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterInfo{
				ClusterId:   tt.fields.ClusterId,
				Cluster:     tt.fields.Cluster,
				NodesVNodes: tt.fields.NodesVNodes,
			}
			assert.Equalf(t, tt.want, c.ReplicationFactor(), "ReplicationFactor()")
		})
	}
}
