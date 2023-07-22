package pkg

import (
	"bytes"
	"context"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/sdktypes"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/utils"
	"math"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

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

var (
	chainSubscriptionFactory = subscription.NewChainFactory()
	watchdogFactory          = subscription.NewWatchdogFactory()
	watchdogTimeout          = time.Minute
)

type (
	blockchainClient struct {
		*gsrpc.SubstrateAPI
		eventContractAccount types.AccountID
		eventDispatcher      map[types.Hash]sdktypes.ContractEventDispatchEntry
		eventContextCancel   []context.CancelFunc
		connectMutex         sync.Mutex
		eventDecoder         subscription.EventDecoder
	}
)

func CreateBlockchainClient(apiUrl string) sdktypes.BlockchainClient {
	substrateAPI, err := gsrpc.NewSubstrateAPI(apiUrl)
	if err != nil {
		log.WithError(err).WithField("apiUrl", apiUrl).Fatal("Can't connect to blockchainClient")
	}

	return &blockchainClient{
		SubstrateAPI: substrateAPI,
		eventDecoder: subscription.NewEventDecoder(),
	}
}

func (b *blockchainClient) SetEventDispatcher(contractAddressSS58 string, dispatcher map[types.Hash]sdktypes.ContractEventDispatchEntry) error {
	contract, err := utils.DecodeAccountIDFromSS58(contractAddressSS58)
	if err != nil {
		return err
	}
	b.eventContractAccount = contract
	b.eventDispatcher = dispatcher
	err = b.listenContractEvents()
	if err != nil {
		return err
	}
	return nil
}

func (b *blockchainClient) listenContractEvents() error {
	meta, err := b.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil, nil)
	if err != nil {
		return err
	}

	s, err := b.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
	if err != nil {
		return err
	}
	b.processChainSubscription(chainSubscriptionFactory.NewChainSubscription(s), key, meta)
	return nil
}

func (b *blockchainClient) processChainSubscription(sub subscription.ChainSubscription, key types.StorageKey, meta *types.Metadata) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	b.eventContextCancel = append(b.eventContextCancel, cancel)
	watchdog := watchdogFactory.NewWatchdog(watchdogTimeout)
	eventArrived := true
	var lastEventBlockHash types.Hash
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-ctx.Done():
				log.Info("Chain subscription context done")
				return

			case <-watchdog.C():
				if !eventArrived {
					log.WithField("block", lastEventBlockHash.Hex()).Warn("Watchdog event timeout")

					// read missed blocks
					lastEventBlock, err := b.RPC.Chain.GetBlock(lastEventBlockHash)
					if err != nil {
						log.WithError(err).Warn("Error fetching block")
						break
					}
					lastEventBlockNumber := lastEventBlock.Block.Header.Number
					headerLatest, err := b.RPC.Chain.GetHeaderLatest()
					if err != nil {
						log.WithError(err).Warn("Error fetching latest header")
					} else if headerLatest.Number > lastEventBlockNumber {
						for i := lastEventBlockNumber + 1; i <= headerLatest.Number; i++ {
							missedBlock, err := b.RPC.Chain.GetBlockHash(uint64(i))
							if err != nil {
								log.Println(err)
								continue
							}
							storageData, err := b.RPC.State.GetStorageRaw(key, missedBlock)
							if err != nil {
								log.WithError(err).Error("Error fetching storage data")
								continue
							}
							events, err := b.eventDecoder.DecodeEvents(*storageData, meta)
							if err != nil {
								log.WithError(err).Error("Error parsing events")
								continue
							}

							b.processEvents(events, missedBlock)
							lastEventBlockHash = missedBlock
						}
					}

					// try to resubscribe
					s, err := b.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
					if err != nil {
						log.WithError(err).Warn("Watchdog resubscribtion failed")
						break
					}
					log.Info("Watchdog event resubscribed")
					sub.Unsubscribe()
					sub = chainSubscriptionFactory.NewChainSubscription(s)
				}
				eventArrived = false

			case err := <-sub.Err():
				log.WithError(err).Warn("Subscription signaled an error")

			case evt := <-sub.Chan():
				if evt.Changes == nil {
					log.WithField("block", evt.Block.Hex()).Warn("Received nil event")
					break
				}
				eventArrived = true
				lastEventBlockHash = evt.Block

				// parse all events for this block
				for _, chng := range evt.Changes {
					if !bytes.Equal(chng.StorageKey[:], key) || !chng.HasStorageData {
						// skip, we are only interested in events with content
						continue
					}

					storageData := chng.StorageData
					events, err := b.eventDecoder.DecodeEvents(storageData, meta)
					if err != nil {
						log.WithError(err).Warnf("Error parsing event %x", storageData[:])
						continue
					}

					b.processEvents(events, evt.Block)
				}
			}
		}
	}()
}

func (b *blockchainClient) processEvents(events *types.EventRecords, blockHash types.Hash) {
	for _, e := range events.Contracts_ContractEmitted {
		if !b.eventContractAccount.Equal(&e.Contract) {
			continue
		}

		// Identify the event by matching one of its topics against known signatures. The topics are sorted so
		// the needed one may be in the arbitrary position.
		var dispatchEntry sdktypes.ContractEventDispatchEntry
		found := false
		for _, topic := range e.Topics {
			dispatchEntry, found = b.eventDispatcher[topic]
			if found {
				break
			}
		}
		if !found {

			log.WithField("block", blockHash.Hex()).
				Warnf("Unknown event emitted by our contract: %x", e.Data[:uint32(math.Min(16, float64(len(e.Data))))])
			continue
		}

		if dispatchEntry.Handler == nil {
			log.WithField("block", blockHash.Hex()).WithField("event", dispatchEntry.ArgumentType.Name()).
				Debug("Event unhandeled")
			continue
		}
		args := reflect.New(dispatchEntry.ArgumentType).Interface()
		if err := codec.Decode(e.Data[1:], args); err != nil {
			log.WithError(err).WithField("block", blockHash.Hex()).
				WithField("event", dispatchEntry.ArgumentType.Name()).
				Errorf("Cannot decode event data %x", e.Data)
		}
		log.WithField("block", blockHash.Hex()).WithField("event", dispatchEntry.ArgumentType.Name()).
			Debugf("Event args: %x", e.Data)
		dispatchEntry.Handler(args)
	}
}

func (b *blockchainClient) CallToReadEncoded(contractAddressSS58 string, fromAddress string, method []byte, args ...interface{}) (string, error) {
	data, err := utils.GetContractData(method, args...)
	if err != nil {
		return "", errors.Wrap(err, "getMessagesData")
	}

	res, err := b.callToRead(contractAddressSS58, fromAddress, data)
	if err != nil {
		return "", err
	}

	return res.Result.Ok.Data, nil
}

func (b *blockchainClient) callToRead(contractAddressSS58 string, fromAddress string, data []byte) (sdktypes.Response, error) {
	params := sdktypes.Request{
		Origin:    fromAddress,
		Dest:      contractAddressSS58,
		GasLimit:  500_000_000_000,
		InputData: codec.HexEncodeToString(data),
	}

	res, err := withRetryOnClosedNetwork(b, func() (sdktypes.Response, error) {
		res := sdktypes.Response{}
		return res, b.Client.Call(&res, "contracts_call", params)
	})
	if err != nil {
		return sdktypes.Response{}, errors.Wrap(err, "call")
	}

	return res, nil
}

func (b *blockchainClient) CallToExec(ctx context.Context, contractCall sdktypes.ContractCall) (types.Hash, error) {
	data, err := utils.GetContractData(contractCall.Method, contractCall.Args...)
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
		return b.createExtrinsic("Contracts.call", contractCall.From, multiAddress, valueRaw, gasLimitRaw, types.NewOptionBoolEmpty(), data)
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

func (b *blockchainClient) Deploy(ctx context.Context, deployCall sdktypes.DeployCall) (types.AccountID, error) {
	deployer, err := types.NewAccountID(deployCall.From.PublicKey)
	if err != nil {
		return types.AccountID{}, err
	}

	data, err := utils.GetContractData(deployCall.Method, deployCall.Args...)
	if err != nil {
		return types.AccountID{}, err
	}

	extrinsic, err := withRetryOnClosedNetwork(b, func() (types.Extrinsic, error) {
		return b.createExtrinsic(
			"Contracts.instantiate_with_code",
			deployCall.From,
			types.NewUCompactFromUInt(uint64(deployCall.Value*CERE)),
			types.NewUCompactFromUInt(uint64(deployCall.GasLimit*CERE)),
			types.NewOptionBoolEmpty(),
			deployCall.Code,
			data,
			deployCall.Salt)
	})
	if err != nil {
		return types.AccountID{}, err
	}

	hash, err := withRetryOnClosedNetwork(b, func() (types.Hash, error) {
		return b.submitAndWaitExtrinsic(ctx, extrinsic)
	})
	if err != nil {
		return types.AccountID{}, err
	}

	return withRetryOnClosedNetwork(b, func() (types.AccountID, error) {
		return b.grabContractInstantiated(hash, deployer)
	})
}

func (b *blockchainClient) grabContractInstantiated(hash types.Hash, deployer *types.AccountID) (types.AccountID, error) {
	meta, err := b.RPC.State.GetMetadataLatest()
	if err != nil {
		return types.AccountID{}, errors.Wrap(err, "get metadata lastest")
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil, nil)
	if err != nil {
		return types.AccountID{}, errors.Wrap(err, "create storage key")
	}

	storage, err := b.RPC.State.QueryStorageAt([]types.StorageKey{key}, hash)
	if err != nil {
		return types.AccountID{}, errors.Wrap(err, "query storage at block "+hash.Hex())
	}

	for _, st := range storage {
		for _, chng := range st.Changes {
			events := types.EventRecords{}
			err = types.EventRecordsRaw(chng.StorageData).DecodeEventRecords(meta, &events)
			if err != nil {
				log.WithError(err).Warnf("Error parsing event %x", chng.StorageData[:])
				continue
			}
			for _, e := range events.Contracts_Instantiated {
				if !e.Deployer.Equal(deployer) {
					log.Warnf("Deployers mismatch %s and %s", e.Deployer.ToHexString(), deployer.ToHexString())
					continue
				}
				return e.Contract, nil
			}
		}
	}

	return types.AccountID{}, errors.New("Contract not instantiated at block " + hash.Hex())
}

func (b *blockchainClient) createExtrinsic(cmd string, authKey signature.KeyringPair, args ...interface{}) (types.Extrinsic, error) {
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

	call, err := types.NewCall(meta, cmd, args...)
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
	if utils.IsClosedNetworkError(err) {
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
	if !utils.IsClosedNetworkError(err) {
		return nil
	}

	b.unsubscribeAll()

	substrateAPI, err := gsrpc.NewSubstrateAPI(b.Client.URL())
	if err != nil {
		log.WithError(err).Warningf("Blockchain client can't reconnect to %s", b.Client.URL())
		return err
	}
	b.SubstrateAPI = substrateAPI
	if b.eventDispatcher != nil {
		err = b.listenContractEvents()
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *blockchainClient) unsubscribeAll() {
	for _, c := range b.eventContextCancel {
		c()
	}
	b.eventContextCancel = nil
}
