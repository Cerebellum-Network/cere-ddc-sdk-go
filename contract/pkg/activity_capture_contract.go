package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/abi"
	"github.com/patractlabs/go-patract/metadata"
	"github.com/patractlabs/go-patract/rpc"
	log "github.com/sirupsen/logrus"
	"time"
)

type (
	ActivityCaptureContract interface {
		GetCommit() string
		SetCommit(data string) error
	}

	activityCaptureContract struct {
		contract       Contract
		lastAccessTime time.Time
		apiUrl         string
		accountId      string
		keyringPair    signature.KeyringPair
	}
)

func CreateActivityCaptureContract(apiUrl string, contractAccountId string, secret string) ActivityCaptureContract {
	keyringPair, err := KeyringPairFromSecret(secret)
	if err != nil {
		log.WithError(err).Fatal("Can't initialize keyring pair for activity capture contract")
	}

	smartContract, err := rpc.NewContractAPI(apiUrl)
	if err != nil {
		log.WithError(err).WithField("apiUrl", apiUrl).Fatal("Can't initialize activity capture contract api")
	}

	if err := smartContract.WithMetaData(abi.ActivityCapture); err != nil {
		log.WithError(err).Fatal("Can't initialize activity capture contract metadata")
	}

	contractMetadata, _ := metadata.New(abi.ActivityCapture)

	return &activityCaptureContract{
		contract:    CreateContract(smartContract, contractAccountId, contractMetadata),
		apiUrl:      apiUrl,
		accountId:   contractAccountId,
		keyringPair: keyringPair,
	}
}

func (a *activityCaptureContract) GetCommit() string {
	//TODO implement me
	panic("implement me")
}

func (a *activityCaptureContract) SetCommit(data string) error {
	//TODO implement me
	panic("implement me")
}

func (a *activityCaptureContract) GetApiUrl() string {
	return a.apiUrl
}

func (a *activityCaptureContract) GetAccountId() string {
	return a.accountId
}

func (a *activityCaptureContract) GetLastAccessTime() time.Time {
	return a.lastAccessTime
}
