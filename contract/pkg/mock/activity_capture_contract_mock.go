package mock

import (
	"context"
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/actcapture"
	"math/big"
	"math/rand"
)

type (
	activityCaptureContractMock struct {
		commit actcapture.Commit
	}
)

var _ actcapture.ActivityCaptureContract = (*activityCaptureContractMock)(nil)

func CreateActivityCaptureContractMock() actcapture.ActivityCaptureContract {
	return &activityCaptureContractMock{}
}

func (a *activityCaptureContractMock) GetCommit() (*actcapture.Commit, error) {
	commit := a.commit
	return &commit, nil
}

func (a *activityCaptureContractMock) SetCommit(ctx context.Context, hash []byte, resources uint64) (string, error) {
	a.commit = actcapture.Commit{Hash: types.NewHash(hash), Resources: types.NewU128(*new(big.Int).SetUint64(resources))}

	token := make([]byte, 32)
	rand.Read(token)

	return hex.EncodeToString(token), nil
}

func (a *activityCaptureContractMock) GetContractAddress() string {
	return "mock_activity_capture"
}
