package pkg

import (
	"context"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v2"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	CERE            = 10_000_000_000
	DefaultGasLimit = 5_000_000_000_000
)

type (
	BlockchainClient interface {
		CallToReadEncoded(contractAddressSS58 string, fromAddress string, method []byte, args ...interface{}) (string, error)
		CallToExec(ctx context.Context, contractCall ContractCall) (types.Hash, error)
	}

	blockchainClient struct {
		*gsrpc.SubstrateAPI
	}

	ContractCall struct {
		ContractAddress types.AccountID
		From            signature.KeyringPair
		Value           float64
		GasLimit        float64
		Method          []byte
		Args            []interface{}
	}
)

func CreateBlockchainClient(apiUrl string) BlockchainClient {
	substrateAPI, err := gsrpc.NewSubstrateAPI(apiUrl)
	if err != nil {
		log.WithError(err).WithField("apiUrl", apiUrl).Fatal("Can't connect to blockchainClient")
	}

	return &blockchainClient{
		SubstrateAPI: substrateAPI,
	}
}

func (b *blockchainClient) CallToReadEncoded(contractAddressSS58 string, fromAddress string, method []byte, args ...interface{}) (string, error) {
	params := struct {
		Origin    string `json:"origin"`
		Dest      string `json:"dest"`
		GasLimit  uint   `json:"gasLimit"`
		InputData string `json:"inputData"`
		Value     int    `json:"value"`
	}{
		Origin:   fromAddress,
		Dest:     contractAddressSS58,
		GasLimit: DefaultGasLimit,
	}

	data, err := GetContractData(method, args...)
	if err != nil {
		return "", errors.Wrap(err, "getMessagesData")
	}

	params.InputData = types.HexEncodeToString(data)

	res := struct {
		Success struct {
			Data  string `json:"data"`
			Flags int    `json:"flags"`
		} `json:"success"`
	}{}

	err = b.Client.Call(&res, "contracts_call", params)
	if err != nil {
		return "", errors.Wrap(err, "call")
	}

	return res.Success.Data, nil
}

func (b *blockchainClient) CallToExec(ctx context.Context, contractCall ContractCall) (types.Hash, error) {
	valueRaw := types.NewUCompactFromUInt(uint64(contractCall.Value * CERE))

	var gasLimitRaw types.UCompact
	if contractCall.GasLimit < 0 {
		gasLimitRaw = types.NewUCompactFromUInt(uint64(contractCall.GasLimit * CERE))
	} else {
		gasLimitRaw = types.NewUCompactFromUInt(DefaultGasLimit)
	}

	data, err := GetContractData(contractCall.Method, contractCall.Args...)
	if err != nil {
		return types.Hash{}, err
	}

	extrinsic, err := b.CreateExtrinsic(contractCall.From, contractCall.ContractAddress, valueRaw, gasLimitRaw, data)
	if err != nil {
		return types.Hash{}, err
	}

	return b.SubmitAndWaitExtrinsic(ctx, extrinsic)
}

func (b *blockchainClient) SubmitAndWaitExtrinsic(ctx context.Context, extrinsic types.Extrinsic) (types.Hash, error) {
	sub, err := b.RPC.Author.SubmitAndWatchExtrinsic(extrinsic)
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "submit error")
	}
	defer sub.Unsubscribe()

	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				return status.AsInBlock, nil
			}
		case err := <-sub.Err():
			return types.Hash{}, errors.Wrap(err, "subscribe error")
		case <-ctx.Done():
			return types.Hash{}, ctx.Err()
		}
	}
}

func (b *blockchainClient) CreateExtrinsic(authKey signature.KeyringPair, args ...interface{}) (types.Extrinsic, error) {
	meta, err := b.RPC.State.GetMetadataLatest()
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "get metadata lastest error")
	}

	call, err := types.NewCall(meta, "Contracts.call", args...)
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "new call error")
	}

	genesisHash, err := b.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "get block hash error")
	}

	rv, err := b.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "get runtime version lastest error")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", authKey.PublicKey, nil)
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "create storage key error")
	}

	var accountInfo types.AccountInfo
	ok, err := b.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return types.Extrinsic{}, errors.Wrapf(err, "create storage key error by %s", authKey.Address)
	} else if !ok {
		return types.Extrinsic{}, errors.Errorf("no accountInfo found by %s", authKey.Address)
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	ext := types.NewExtrinsic(call)
	if err := ext.Sign(authKey, o); err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "sign extrinsic error")
	}

	return ext, nil
}
