package pkg

import (
	"fmt"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"testing"
)

var client = CreateBlockchainClient("wss://rpc.devnet.cere.network/ws")

//var ac = actcapture.CreateActivityCaptureContract(client, "5Cm2DRrqS1e4FQin52Bv4eDrnmzWMy12UrUQTUQfceN4z4PH", "smoke rotate chef remember evoke joy sibling scheme document fetch inch fly")
var b = bucket.CreateDdcBucketContract(client, "5GqwX528CHg1jAGuRsiwDwBVXruUvnPeLkEcki4YFbigfKsC")

func TestBucketContract(t *testing.T) {
	nodeGet, err := b.NodeGet(0)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(nodeGet)

	clusterGet, err := b.ClusterGet(1)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(clusterGet)

	bucketGet, err := b.BucketGet(12)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(bucketGet)
}

func TestExtrinsic(t *testing.T) {
}

func TestRead(t *testing.T) {
}
