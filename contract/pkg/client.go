package pkg

import (
	"bytes"
	"context"
	"reflect"
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	CERE = 10_000_000_000
)

type (
	BlockchainClient interface {
		CallToReadEncoded(contractAddressSS58 string, fromAddress string, method []byte, args ...interface{}) (string, error)
		CallToExec(ctx context.Context, contractCall ContractCall) (types.Hash, error)
		SetEventDispatcher(dispatcher map[types.Hash]ContractEventDispatchEntry)
		ListenContractEvents(contractAddressSS58 string) error
	}

	blockchainClient struct {
		*gsrpc.SubstrateAPI
		eventDispatcher map[types.Hash]ContractEventDispatchEntry
		connectMutex    sync.Mutex
	}

	ContractCall struct {
		ContractAddress     types.AccountID
		ContractAddressSS58 string
		From                signature.KeyringPair
		Value               float64
		GasLimit            float64
		Method              []byte
		Args                []interface{}
	}

	ContractEventDispatchEntry struct {
		ArgumentType reflect.Type
		Handler      ContractEventHandler
	}

	ContractEventDispatchDescription struct {
		Topic string
		ContractEventDispatchEntry
	}

	ContractEventHandler func(interface{})

	Response struct {
		DebugMessage string `json:"debugMessage"`
		GasConsumed  int    `json:"gasConsumed"`
		Result       struct {
			Ok struct {
				Data  string `json:"data"`
				Flags int    `json:"flags"`
			} `json:"Ok"`
		} `json:"result"`
	}

	Request struct {
		Origin    string `json:"origin"`
		Dest      string `json:"dest"`
		GasLimit  uint   `json:"gasLimit"`
		InputData string `json:"inputData"`
		Value     int    `json:"value"`
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

func (b *blockchainClient) SetEventDispatcher(dispatcher map[types.Hash]ContractEventDispatchEntry) {
	b.eventDispatcher = dispatcher
}

func (b *blockchainClient) ListenContractEvents(contractAddressSS58 string) error {
	meta, err := b.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}

	contract, err := DecodeAccountIDFromSS58(contractAddressSS58)
	if err != nil {
		return err
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil, nil)
	if err != nil {
		return err
	}

	sub, err := b.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
	if err != nil {
		return err
	}

	go func() {
		defer sub.Unsubscribe()
		for {
			evt := <-sub.Chan()

			// parse all events for this block
			for _, chng := range evt.Changes {
				if !bytes.Equal(chng.StorageKey[:], key) || !chng.HasStorageData {
					// skip, we are only interested in events with content
					continue
				}

				events := types.EventRecords{}
				err = types.EventRecordsRaw(chng.StorageData).DecodeEventRecords(meta, &events)
				if err != nil {
					log.Warnf("Error parsing event %x", chng.StorageData[:])
					continue
				}

				for _, e := range events.Contracts_ContractEmitted {
					if !contract.Equal(&e.Contract) {
						continue
					}
					dispatchEntry, found := b.eventDispatcher[e.Topics[0]]
					if !found {
						log.WithField("topic", e.Topics[0].Hex()).WithField("hash", evt.Block.Hex()).Warn("Unknown event emitted by our contract")
						continue
					}
					if dispatchEntry.Handler == nil {
						log.WithField("hash", evt.Block.Hex()).WithField("event", dispatchEntry.ArgumentType.Name()).Info("Event unhandeled")
						continue
					}
					args := reflect.New(dispatchEntry.ArgumentType).Interface()
					if err := codec.Decode(e.Data[1:], args); err != nil {
						log.WithError(err).WithField("hash", evt.Block.Hex()).WithField("event", dispatchEntry.ArgumentType.Name()).
							Errorf("Cannot decode event data %x", e.Data)
					}
					log.WithField("hash", evt.Block.Hex()).WithField("event", dispatchEntry.ArgumentType.Name()).Infof("Event args: %x", e.Data)
					dispatchEntry.Handler(args)
				}
			}
		}
	}()
	return nil
}

func (b *blockchainClient) CallToReadEncoded(contractAddressSS58 string, fromAddress string, method []byte, args ...interface{}) (string, error) {
	data, err := GetContractData(method, args...)
	if err != nil {
		return "", errors.Wrap(err, "getMessagesData")
	}

	res, err := b.callToRead(contractAddressSS58, fromAddress, data)
	if err != nil {
		return "", err
	}

	return res.Result.Ok.Data, nil
}

func (b *blockchainClient) callToRead(contractAddressSS58 string, fromAddress string, data []byte) (Response, error) {
	params := Request{
		Origin:    fromAddress,
		Dest:      contractAddressSS58,
		GasLimit:  500_000_000_000,
		InputData: codec.HexEncodeToString(data),
	}

	res, err := withRetryOnClosedNetwork(b, func() (Response, error) {
		res := Response{}
		return res, b.Client.Call(&res, "contracts_call", params)
	})
	if err != nil {
		return Response{}, errors.Wrap(err, "call")
	}

	return res, nil
}

func (b *blockchainClient) CallToExec(ctx context.Context, contractCall ContractCall) (types.Hash, error) {
	data, err := GetContractData(contractCall.Method, contractCall.Args...)
	if err != nil {
		return types.Hash{}, err
	}

	valueRaw := types.NewUCompactFromUInt(uint64(contractCall.Value * CERE))
	var gasLimitRaw types.UCompact
	if contractCall.GasLimit > 0 {
		gasLimitRaw = types.NewUCompactFromUInt(uint64(contractCall.GasLimit * CERE))
	} else {
		resp, err := b.callToRead(contractCall.ContractAddressSS58, contractCall.From.Address, data)
		if err != nil {
			return types.Hash{}, err
		}

		gasLimitRaw = types.NewUCompactFromUInt(uint64(resp.GasConsumed))
	}

	multiAddress := types.MultiAddress{IsID: true, AsID: contractCall.ContractAddress}
	extrinsic, err := withRetryOnClosedNetwork(b, func() (types.Extrinsic, error) {
		return b.createExtrinsic(contractCall.From, multiAddress, valueRaw, gasLimitRaw, types.NewOptionBoolEmpty(), data)
	})
	if err != nil {
		return types.Hash{}, err
	}

	hash, err := withRetryOnClosedNetwork(b, func() (types.Hash, error) {
		return b.submitAndWaitExtrinsic(ctx, extrinsic)
	})
	if err != nil {
		return types.Hash{}, err
	}

	return hash, err
}

func (b *blockchainClient) createExtrinsic(authKey signature.KeyringPair, args ...interface{}) (types.Extrinsic, error) {
	meta, err := b.RPC.State.GetMetadataLatest()
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "get metadata lastest error")
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

	call, err := types.NewCall(meta, "Contracts.call", args...)
	if err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "new call error")
	}
	ext := types.NewExtrinsic(call)

	if err := ext.Sign(authKey, o); err != nil {
		return types.Extrinsic{}, errors.Wrap(err, "sign extrinsic error")
	}

	return ext, nil
}

func (b *blockchainClient) submitAndWaitExtrinsic(ctx context.Context, extrinsic types.Extrinsic) (types.Hash, error) {
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

func withRetryOnClosedNetwork[T any](b *blockchainClient, f func() (T, error)) (T, error) {
	result, err := f()
	if isClosedNetworkError(err) {
		if b.reconnect() != nil {
			return result, err
		}

		result, err = f()
	}
	return result, err
}

func (b *blockchainClient) reconnect() error {
	b.connectMutex.Lock()
	defer b.connectMutex.Unlock()
	_, err := b.RPC.State.GetRuntimeVersionLatest()
	if !isClosedNetworkError(err) {
		return nil
	}

	substrateAPI, err := gsrpc.NewSubstrateAPI(b.Client.URL())
	if err != nil {
		log.WithError(err).Warningf("Blockchain client can't reconnect to %s", b.Client.URL())
		return err
	}
	b.SubstrateAPI = substrateAPI

	return nil
}
