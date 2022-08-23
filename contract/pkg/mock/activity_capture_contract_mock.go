package mock

import (
	"context"
	"encoding/hex"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/actcapture"
	"math/rand"
)

type (
	activityCaptureContractMock struct {
		commit string
	}
)

var _ actcapture.ActivityCaptureContract = (*activityCaptureContractMock)(nil)

func CreateActivityCaptureContractMock() actcapture.ActivityCaptureContract {
	return &activityCaptureContractMock{}
}

func (a *activityCaptureContractMock) GetCommit() (string, error) {
	return a.commit, nil
}

func (a *activityCaptureContractMock) SetCommit(ctx context.Context, data string) (string, error) {
	a.commit = data

	token := make([]byte, 32)
	rand.Read(token)

	return hex.EncodeToString(token), nil
}

func (a *activityCaptureContractMock) GetContractAddress() string {
	return "mock_activity_capture"
}
