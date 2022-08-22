package pkg

import (
	"context"
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	log "github.com/sirupsen/logrus"
)

const getCommitMethod = "5329f551"
const setCommitMethod = "e445e1fd"

type (
	ActivityCaptureContract interface {
		GetContractAddress() string

		GetCommit() (string, error)
		SetCommit(ctx context.Context, data string) (string, error)
	}

	activityCaptureContract struct {
		client              BlockchainClient
		account             types.AccountID
		keyringPair         signature.KeyringPair
		contractAddress     types.AccountID
		contractAddressSS58 string
		getCommitMethodId   []byte
		setCommitMethodId   []byte
	}
)

func CreateActivityCaptureContract(client BlockchainClient, contractAddressSS58 string, secret string) ActivityCaptureContract {
	keyringPair, err := signature.KeyringPairFromSecret(secret, 42)
	if err != nil {
		log.WithError(err).Fatal("Can't initialize keyring pair for activity capture contract")
	}

	account, err := DecodeAccountIDFromSS58(keyringPair.Address)
	if err != nil {
		log.WithError(err).WithField("account", keyringPair.Address).Fatal("Can't decode accountIDSS58")
	}

	getCommitMethodId, err := hex.DecodeString(getCommitMethod)
	if err != nil {
		log.WithError(err).WithField("method", getCommitMethod).Fatal("Can't decode method getCommitMethod")
	}

	setCommitMethodId, err := hex.DecodeString(setCommitMethod)
	if err != nil {
		log.WithError(err).WithField("method", setCommitMethod).Fatal("Can't decode method setCommitMethod")
	}

	contractAddress, err := DecodeAccountIDFromSS58(contractAddressSS58)
	if err != nil {
		log.WithError(err).WithField("contractAddressSS58", contractAddressSS58).Fatal("Can't decode contract address SS58")
	}

	return &activityCaptureContract{
		client:              client,
		keyringPair:         keyringPair,
		account:             account,
		contractAddress:     contractAddress,
		contractAddressSS58: contractAddressSS58,
		getCommitMethodId:   getCommitMethodId,
		setCommitMethodId:   setCommitMethodId,
	}
}

func (a *activityCaptureContract) GetCommit() (string, error) {
	return a.client.CallToReadEncoded(a.contractAddressSS58, a.keyringPair.Address, a.getCommitMethodId, a.account)
}

func (a *activityCaptureContract) SetCommit(ctx context.Context, data string) (string, error) {
	call := ContractCall{
		ContractAddress: a.contractAddress,
		From:            a.keyringPair,
		Value:           0,
		GasLimit:        -1,
		Method:          a.setCommitMethodId,
		Args:            []interface{}{a.account, types.Text(data)},
	}

	hash, err := a.client.CallToExec(ctx, call)
	if err != nil {
		return "", err
	}

	return hash.Hex(), nil
}

func (a *activityCaptureContract) GetContractAddress() string {
	return a.contractAddressSS58
}
