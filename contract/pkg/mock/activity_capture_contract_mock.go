package mock

import (
	"context"
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/actcapture"
	"math/big"
	"math/rand"
	"time"
)

type (
	activityCaptureContractMock struct {
		commit *actcapture.Commit
	}
)

var _ actcapture.ActivityCaptureContract = (*activityCaptureContractMock)(nil)

func CreateActivityCaptureContractMock() actcapture.ActivityCaptureContract {
	return &activityCaptureContractMock{}
}

func (a *activityCaptureContractMock) GetCommit() (*actcapture.Commit, error) {
	return a.commit, nil
}

func (a *activityCaptureContractMock) SetCommit(ctx context.Context, hash []byte, gas, from, to uint64) (string, error) {
	a.commit = &actcapture.Commit{Hash: types.NewHash(hash), Gas: types.NewU128(*new(big.Int).SetUint64(gas)), From: types.U64(from), To: types.U64(to)}

	token := make([]byte, 32)
	rand.Read(token)

	return hex.EncodeToString(token), nil
}

func (a *activityCaptureContractMock) GetEraSettings() (*actcapture.EraConfig, error) {
	return &actcapture.EraConfig{
		Start:    types.U64(time.Now().UTC().UnixMilli()),
		Interval: 20_000,
	}, nil
}

func (a *activityCaptureContractMock) GetContractAddress() string {
	return "mock_activity_capture"
}
