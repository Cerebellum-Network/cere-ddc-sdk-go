package blockchain

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pallets"
)

type EventsListener func(events *pallets.Events, blockNumber types.BlockNumber, blockHash types.Hash)

type Client struct {
	*gsrpc.SubstrateAPI

	eventsListeners map[*EventsListener]struct{}
	mu              sync.Mutex
	isListening     uint32
	cancelListening func()
	errsListening   chan error

	DdcClusters  pallets.DdcClustersApi
	DdcCustomers pallets.DdcCustomersApi
	DdcNodes     pallets.DdcNodesApi
	DdcPayouts   pallets.DdcPayoutsApi
}

func NewClient(url string) (*Client, error) {
	substrateApi, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}
	meta, err := substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	return &Client{
		SubstrateAPI:    substrateApi,
		eventsListeners: make(map[*EventsListener]struct{}),
		DdcClusters:     pallets.NewDdcClustersApi(substrateApi),
		DdcCustomers:    pallets.NewDdcCustomersApi(substrateApi, meta),
		DdcNodes:        pallets.NewDdcNodesApi(substrateApi, meta),
		DdcPayouts:      pallets.NewDdcPayoutsApi(substrateApi, meta),
	}, nil
}

func (c *Client) StartEventsListening(
	afterBlock func(blockNumber types.BlockNumber, blockHash types.Hash),
) (context.CancelFunc, <-chan error, error) {
	if !atomic.CompareAndSwapUint32(&c.isListening, 0, 1) {
		return c.cancelListening, c.errsListening, nil
	}

	meta, err := c.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, nil, err
	}
	key, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return nil, nil, err
	}
	sub, err := c.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
	if err != nil {
		return nil, nil, err
	}

	done := make(chan struct{})
	c.errsListening = make(chan error)

	go func() {
		for {
			select {
			case <-done:
				return
			case set := <-sub.Chan():
				c.onChanges(
					meta,
					key,
					set.Changes,
					set.Block,
					func(events *pallets.Events, blockNumber types.BlockNumber, blockHash types.Hash) {
						c.mu.Lock()
						for callback := range c.eventsListeners {
							(*callback)(events, blockNumber, blockHash)
						}
						c.mu.Unlock()
					},
				)

				header, err := c.RPC.Chain.GetHeader(set.Block)
				if err != nil {
					c.errsListening <- fmt.Errorf("get header: %w", err)
					return
				}

				if afterBlock != nil {
					afterBlock(header.Number, set.Block)
				}
			}
		}
	}()

	once := sync.Once{}
	c.cancelListening = func() {
		once.Do(func() {
			done <- struct{}{}
			sub.Unsubscribe()
			c.isListening = 0
		})
	}

	return c.cancelListening, c.errsListening, nil
}

// RegisterEventsListener subscribes given callback to blockchain events.
func (c *Client) RegisterEventsListener(callback EventsListener) context.CancelFunc {
	c.mu.Lock()
	c.eventsListeners[&callback] = struct{}{}
	c.mu.Unlock()

	once := sync.Once{}
	return func() {
		once.Do(func() {
			c.mu.Lock()
			delete(c.eventsListeners, &callback)
			c.mu.Unlock()
		})
	}
}

func (c *Client) onChanges(
	meta *types.Metadata,
	key types.StorageKey,
	changes []types.KeyValueOption,
	block types.Hash,
	callback EventsListener,
) {
	header, err := c.RPC.Chain.GetHeader(block)
	if err != nil {
		c.errsListening <- fmt.Errorf("get header: %w", err)
		return
	}

	for _, change := range changes {
		if !codec.Eq(change.StorageKey, key) || !change.HasStorageData {
			continue
		}

		events := &pallets.Events{}
		err = types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, events)
		if err != nil {
			c.errsListening <- fmt.Errorf("events decoder: %w", err)
			continue
		}

		callback(events, header.Number, block)
	}
}

type blockEvents struct {
	Events *pallets.Events
	Hash   types.Hash
	Number types.BlockNumber
}

type pendingEvents struct {
	list []*blockEvents
	mu   sync.Mutex
	done bool
}

func (pe *pendingEvents) TryPush(events *pallets.Events, hash types.Hash, number types.BlockNumber) bool {
	pe.mu.Lock()
	if !pe.done {
		pe.list = append(pe.list, &blockEvents{
			Events: events,
			Hash:   hash,
			Number: number,
		})
		pe.mu.Unlock()
		return true
	}
	pe.mu.Unlock()
	return false
}

func (pe *pendingEvents) Do(callback EventsListener) {
	for {
		pe.mu.Lock()

		if len(pe.list) == 0 {
			pe.done = true
			pe.mu.Unlock()
			break
		}

		callback(pe.list[0].Events, pe.list[0].Number, pe.list[0].Hash)

		pe.list = pe.list[1:]
		pe.mu.Unlock()
	}
}
