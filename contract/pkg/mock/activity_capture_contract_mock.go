package mock

import (
	"encoding/hex"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"math/rand"
)

type (
	activityCaptureContractMock struct {
		commit string
	}
)

var _ pkg.ActivityCaptureContract = (*activityCaptureContractMock)(nil)

func CreateActivityCaptureContractMock() pkg.ActivityCaptureContract {
	return &activityCaptureContractMock{}
}

func (a *activityCaptureContractMock) GetCommit() (string, error) {
	return a.commit, nil
}

func (a *activityCaptureContractMock) SetCommit(data string) (string, error) {
	a.commit = data

	token := make([]byte, 32)
	rand.Read(token)

	return hex.EncodeToString(token), nil
}
