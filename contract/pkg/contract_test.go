package pkg

import (
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestExtrinsic(t *testing.T) {
	act := CreateActivityCaptureContract("wss://rpc.devnet.cere.network/ws", "5Cm2DRrqS1e4FQin52Bv4eDrnmzWMy12UrUQTUQfceN4z4PH", "smoke rotate chef remember evoke joy sibling scheme document fetch inch fly")

	commit, err := act.SetCommit("1")

	if err != nil {
		log.WithError(err).Infof("!!!!!!")
		t.Error(err, "ERROR")
		return
	}

	log.Info(commit)
}

func TestRead(t *testing.T) {
	act := CreateActivityCaptureContract("wss://rpc.devnet.cere.network/ws", "5Cm2DRrqS1e4FQin52Bv4eDrnmzWMy12UrUQTUQfceN4z4PH", "smoke rotate chef remember evoke joy sibling scheme document fetch inch fly")

	commit, err := act.GetCommit()
	if err != nil {
		t.Error(err, "ERROR")
		return
	}

	log.Info(commit)
}
