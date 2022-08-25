package actcapture

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	log "github.com/sirupsen/logrus"
	"math/big"
)

const getCommitMethod = "5329f551"
const setCommitMethod = "e445e1fd"

type (
	ActivityCaptureContract interface {
		GetContractAddress() string

		GetCommit() (*Commit, error)
		SetCommit(ctx context.Context, hash []byte, resources uint64) (string, error)
	}

	activityCaptureContract struct {
		client              pkg.BlockchainClient
		account             types.AccountID
		keyringPair         signature.KeyringPair
		contractAddress     types.AccountID
		contractAddressSS58 string
		getCommitMethodId   []byte
		setCommitMethodId   []byte
	}
)

func CreateActivityCaptureContract(client pkg.BlockchainClient, contractAddressSS58 string, secret string) ActivityCaptureContract {
	keyringPair, err := signature.KeyringPairFromSecret(secret, 42)
	if err != nil {
		log.WithError(err).Fatal("Can't initialize keyring pair for activity capture contract")
	}

	account, err := pkg.DecodeAccountIDFromSS58(keyringPair.Address)
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

	contractAddress, err := pkg.DecodeAccountIDFromSS58(contractAddressSS58)
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

func (a *activityCaptureContract) GetCommit() (*Commit, error) {
	encoded, err := a.client.CallToReadEncoded(a.contractAddressSS58, a.keyringPair.Address, a.getCommitMethodId, a.account)
	if err != nil {
		return nil, err
	} else if len(encoded) == 0 {
		return nil, errors.New("commit doesn't exist")
	}

	result := &Commit{}
	if err = types.DecodeFromHexString(encoded, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *activityCaptureContract) SetCommit(ctx context.Context, hash []byte, resources uint64) (string, error) {
	call := pkg.ContractCall{
		ContractAddress: a.contractAddress,
		From:            a.keyringPair,
		Value:           0,
		GasLimit:        -1,
		Method:          a.setCommitMethodId,
		Args:            []interface{}{a.account, types.NewHash(hash), types.NewU128(*new(big.Int).SetUint64(resources))},
	}

	blockHash, err := a.client.CallToExec(ctx, call)
	if err != nil {
		return "", err
	}

	return blockHash.Hex(), nil
}

func (a *activityCaptureContract) GetContractAddress() string {
	return a.contractAddressSS58
}
