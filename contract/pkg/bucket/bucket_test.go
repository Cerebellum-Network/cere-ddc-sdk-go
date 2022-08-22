package bucket

import (
	"fmt"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"testing"
)

var client = pkg.CreateBlockchainClient("wss://rpc.devnet.cere.network/ws")

//var ac = actcapture.CreateActivityCaptureContract(client, "5Cm2DRrqS1e4FQin52Bv4eDrnmzWMy12UrUQTUQfceN4z4PH", "smoke rotate chef remember evoke joy sibling scheme document fetch inch fly")
var b = CreateDdcBucketContract(client, "5GqwX528CHg1jAGuRsiwDwBVXruUvnPeLkEcki4YFbigfKsC")

func TestBucketContract(t *testing.T) {
	nodeGet, err := b.NodeGet(1)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(nodeGet)

	bucketGet, err := b.BucketGet(2)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(bucketGet)

	clusterGet, err := b.ClusterGet(0)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(clusterGet)

}
