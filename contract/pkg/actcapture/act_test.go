package actcapture

import (
	"context"
	"fmt"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"testing"
)

var client = pkg.CreateBlockchainClient("wss://rpc.devnet.cere.network/ws")

var ac = CreateActivityCaptureContract(client, "5Cm2DRrqS1e4FQin52Bv4eDrnmzWMy12UrUQTUQfceN4z4PH", "smoke rotate chef remember evoke joy sibling scheme document fetch inch fly")

func TestActContract(t *testing.T) {
	/*	commit, err := ac.GetCommit()
		if err != nil {
			t.Error(err)
			return
		}

		fmt.Println(commit)*/

	setCommit, err := ac.SetCommit(context.Background(), "123")
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(setCommit)
}
