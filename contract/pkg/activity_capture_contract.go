package pkg

import (
	"context"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/abi"
	"github.com/patractlabs/go-patract/metadata"
	"github.com/patractlabs/go-patract/rpc"
	"github.com/patractlabs/go-patract/utils"
	log "github.com/sirupsen/logrus"
)

type (
	ActivityCaptureContract interface {
		GetCommit() (string, error)
		SetCommit(data string) (string, error)
	}

	activityCaptureContract struct {
		contract    Contract
		apiUrl      string
		accountId   string
		account     struct{ Account types.AccountID }
		keyringPair signature.KeyringPair
	}
)

func CreateActivityCaptureContract(apiUrl string, contractAccountId string, secret string) ActivityCaptureContract {
	keyringPair, err := KeyringPairFromSecret(secret)
	if err != nil {
		log.WithError(err).Fatal("Can't initialize keyring pair for activity capture contract")
	}

	account, err := utils.DecodeAccountIDFromSS58(keyringPair.Address)
	if err != nil {
		log.WithError(err).WithField("account", keyringPair.Address).Fatal("Can't decode accountIDSS58")
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
		account:     struct{ Account types.AccountID }{Account: account},
	}
}

func (a *activityCaptureContract) GetCommit() (string, error) {
	result := ""
	ctx := rpc.NewCtx(context.Background()).WithFrom(a.keyringPair)
	err := a.contract.CallToRead(ctx, &result, []string{"get_commit"}, a.account)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (a *activityCaptureContract) SetCommit(data string) (string, error) {
	ctx := rpc.NewCtx(context.Background()).WithFrom(a.keyringPair)
	hash, err := a.contract.CallToExec(ctx, 0, -1, []string{"set_commit"}, a.account, types.Text(data))
	if err != nil {
		return "", err
	}

	return hash.Hex(), nil
}

func (a *activityCaptureContract) GetApiUrl() string {
	return a.apiUrl
}

func (a *activityCaptureContract) GetAccountId() string {
	return a.accountId
}
