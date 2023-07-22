package pkg

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/mock"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/sdktypes"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func Test_blockchainClient_processChainSubscription(t *testing.T) {
	// given
	mockController := gomock.NewController(t)
	chainSubscriptionMock := mock.NewMockChainSubscription(mockController)
	chainSubscriptionMock.EXPECT().Err().Return(make(chan error)).AnyTimes()
	storageChangeSetsChan := make(chan types.StorageChangeSet)
	chainSubscriptionMock.EXPECT().Chan().Return(storageChangeSetsChan).AnyTimes()
	chainSubscriptionMock.EXPECT().Unsubscribe().AnyTimes()

	chainSubscriptionFactoryMock := mock.NewMockChainSubscriptionFactory(mockController)
	chainSubscriptionFactoryMock.EXPECT().NewChainSubscription(gomock.Any()).Return(chainSubscriptionMock).AnyTimes()
	chainSubscriptionFactory = chainSubscriptionFactoryMock

	stateMock := mock.NewMockState(mockController)
	stateMock.EXPECT().SubscribeStorageRaw(gomock.Any()).Return(nil, nil).AnyTimes()
	chainMock := mock.NewMockChain(mockController)
	//chainMock.EXPECT().GetBlock(gomock.Any()).Return(nil, nil).AnyTimes()

	watchdogMock := mock.NewMockWatchdog(mockController)
	watchdogChan := make(chan time.Time)
	watchdogMock.EXPECT().C().Return(watchdogChan).AnyTimes()
	watchdogFactoryMock := mock.NewMockWatchdogFactory(mockController)
	watchdogFactoryMock.EXPECT().NewWatchdog(gomock.Any()).Return(watchdogMock).AnyTimes()
	watchdogFactory = watchdogFactoryMock

	eventDecoderMock := mock.NewMockEventDecoder(mockController)

	substrateAPI := &gsrpc.SubstrateAPI{
		RPC: &rpc.RPC{
			State: stateMock,
			Chain: chainMock,
		},
	}

	c := blockchainClient{
		SubstrateAPI: substrateAPI,
		eventDecoder: eventDecoderMock,
	}
	c.eventDispatcher = make(map[types.Hash]sdktypes.ContractEventDispatchEntry)

	t.Run("should resubscribe when timeout", func(t *testing.T) {
		h := types.NewHash([]byte{1})
		handlerChan := make(chan bool)
		ticker := time.NewTicker(1 * time.Second)
		go func() {
			<-ticker.C
			handlerChan <- false
		}()
		c.eventDispatcher[h] = sdktypes.ContractEventDispatchEntry{
			Handler: func(_ interface{}) {
				handlerChan <- true
			},
			ArgumentType: reflect.TypeOf(interface{}("")),
		}
		storageKey := h[:]
		events := &types.EventRecords{
			Contracts_ContractEmitted: []types.EventContractsContractEmitted{
				{
					Data: []byte("test"),
					Topics: []types.Hash{
						h,
					},
				},
			},
		}
		eventDecoderMock.EXPECT().DecodeEvents(gomock.Any(), gomock.Any()).Return(events, nil).Times(1)

		c.processChainSubscription(chainSubscriptionMock, storageKey, createMetadata())
		watchdogChan <- time.Now()
		storageChangeSetsChan <- types.StorageChangeSet{
			Changes: []types.KeyValueOption{
				{
					HasStorageData: true,
					StorageKey:     storageKey,
				},
			},
		}
		assert.True(t, <-handlerChan)
		c.unsubscribeAll()
	})
	t.Run("should read missed events from the blockchain before resubscribing", func(t *testing.T) {

		h := types.NewHash([]byte{2})
		storageKey := h[:]

		metadata := createMetadata()
		c.processChainSubscription(chainSubscriptionMock, storageKey, metadata)

		// trigger to save last block
		events := &types.EventRecords{
			Contracts_ContractEmitted: []types.EventContractsContractEmitted{
				{
					Data: []byte("test"),
					Topics: []types.Hash{
						h,
					},
				},
			},
		}
		eventDecoderMock.EXPECT().DecodeEvents(gomock.Any(), gomock.Any()).Return(events, nil).Times(1)
		lastBlockHash := types.NewHash([]byte{123})
		storageChangeSetsChan <- types.StorageChangeSet{
			Block: lastBlockHash,
			Changes: []types.KeyValueOption{
				{
					HasStorageData: true,
					StorageKey:     storageKey,
				},
			},
		}

		// nothing after first watchdog tick
		watchdogChan <- time.Now()

		// Set up mocks
		// getting block number by hash
		lastSignedBlock := &types.SignedBlock{
			Block: types.Block{
				Header: types.Header{
					Number: 1,
				},
			},
		}
		chainMock.EXPECT().GetBlock(lastBlockHash).Return(lastSignedBlock, nil).Times(1)
		// getting current block number
		currentSignedBlock := &types.Header{
			Number: 3,
		}
		chainMock.EXPECT().GetHeaderLatest().Return(currentSignedBlock, nil).Times(1)

		// return block hash by number
		missedBlockHash2 := types.NewHash([]byte{22})
		missedBlockHash3 := types.NewHash([]byte{33})
		chainMock.EXPECT().GetBlockHash(uint64(2)).Return(missedBlockHash2, nil).Times(1)
		chainMock.EXPECT().GetBlockHash(uint64(3)).Return(missedBlockHash3, nil).Times(1)

		// return storage data for missed blocks
		storageDataRaw1 := types.StorageDataRaw{1}
		storageDataRaw2 := types.StorageDataRaw{2}
		stateMock.EXPECT().GetStorageRaw(storageKey, missedBlockHash2).Return(&storageDataRaw1, nil).Times(1)
		stateMock.EXPECT().GetStorageRaw(storageKey, missedBlockHash3).Return(&storageDataRaw2, nil).Times(1)

		// return events for missed blocks
		events2 := &types.EventRecords{
			Contracts_ContractEmitted: []types.EventContractsContractEmitted{
				{
					Data: []byte("test2"),
					Topics: []types.Hash{
						h,
					},
				},
			},
		}
		events3 := &types.EventRecords{
			Contracts_ContractEmitted: []types.EventContractsContractEmitted{
				{
					Data: []byte("test3"),
					Topics: []types.Hash{
						h,
					},
				},
			},
		}
		eventDecoderMock.EXPECT().DecodeEvents([]byte{1}, metadata).Return(events2, nil).Times(1)
		eventDecoderMock.EXPECT().DecodeEvents([]byte{2}, metadata).Return(events3, nil).Times(1)

		// add subscriber to event
		handlerChan := make(chan bool)
		ticker := time.NewTicker(1 * time.Second)
		go func() {
			<-ticker.C
			handlerChan <- false
		}()
		c.eventDispatcher[h] = sdktypes.ContractEventDispatchEntry{
			Handler: func(_ interface{}) {
				handlerChan <- true
			},
			ArgumentType: reflect.TypeOf(interface{}("")),
		}

		// trigger second time, because we resubscribe if there is no event between two watchdog ticks
		watchdogChan <- time.Now()
		assert.True(t, <-handlerChan)

		c.unsubscribeAll()
	})

}

func createMetadata() *types.Metadata {
	metadata := &types.Metadata{
		Version: 14,
		AsMetadataV14: types.MetadataV14{
			Pallets: []types.PalletMetadataV14{
				{
					HasStorage: true,
					Storage: types.StorageMetadataV14{
						Prefix: "System",
						Items: []types.StorageEntryMetadataV14{
							{
								Name: "Events",
								Type: types.StorageEntryTypeV14{
									IsPlainType: true,
								},
							},
						},
					},
				},
			},
		},
	}
	return metadata
}
