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

// StartEventsListening subscribes for blockchain events and passes events starting from the
// 'begin' block to registered events listeners. Listeners registered after this call will only
// receive live events meaning all listeners which need historical events from 'begin' block
// should be registered at the moment of calling this function. The 'afterBlock' callback is
// invoked after all registered events listeners are already invoked.
func (c *Client) StartEventsListening(
	begin types.BlockNumber,
	afterBlock func(blockNumber types.BlockNumber, blockHash types.Hash),
) (context.CancelFunc, <-chan error, error) {
	if !atomic.CompareAndSwapUint32(&c.isListening, 0, 1) {
		return c.cancelListening, c.errsListening, nil
	}

	c.errsListening = make(chan error)

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

	liveChangesC := sub.Chan()
	histChangesC := make(chan types.StorageChangeSet)
	var wg sync.WaitGroup

	// Query historical changes.
	var cancelled atomic.Value
	cancelled.Store(false)
	wg.Add(1)
	go func(begin types.BlockNumber, liveChanges <-chan types.StorageChangeSet, histChangesC chan types.StorageChangeSet) {
		defer wg.Done()
		defer close(histChangesC)

		set := <-liveChanges // first live changes set block is the last historical block

		header, err := c.RPC.Chain.GetHeader(set.Block)
		if err != nil {
			c.errsListening <- fmt.Errorf("get header: %w", err)
			return
		}

		for currentBlock := begin; currentBlock < header.Number; currentBlock++ {
			blockHash, err := c.RPC.Chain.GetBlockHash(uint64(currentBlock))
			if err != nil {
				c.errsListening <- fmt.Errorf("get block hash: %w", err)
				return
			}

			blockChangesSets, err := c.RPC.State.QueryStorageAt([]types.StorageKey{key}, blockHash)
			if err != nil {
				c.errsListening <- fmt.Errorf("query storage: %w", err)
				return
			}

			for _, set := range blockChangesSets {
				histChangesC <- set
			}

			// Graceful stop must finish the block before exiting.
			if cancelled.Load().(bool) {
				return
			}
		}

		histChangesC <- set
	}(begin, liveChangesC, histChangesC)

	// Sequence historical and live changes.
	changesC := make(chan types.StorageChangeSet)
	wg.Add(1)
	go func(histChangesC, liveChangesC <-chan types.StorageChangeSet, changesC chan types.StorageChangeSet) {
		defer wg.Done()
		defer close(changesC)

		for set := range histChangesC {
			changesC <- set
		}

		for set := range liveChangesC {
			changesC <- set
		}
	}(histChangesC, liveChangesC, changesC)

	// Decode events from changes skipping blocks before 'begin'.
	eventsC := make(chan blockEvents)
	wg.Add(1)
	go func(changesC <-chan types.StorageChangeSet, eventsC chan blockEvents) {
		defer wg.Done()
		defer close(eventsC)

		for set := range changesC {
			header, err := c.RPC.Chain.GetHeader(set.Block)
			if err != nil {
				c.errsListening <- fmt.Errorf("get header: %w", err)
				return
			}

			if header.Number < begin {
				continue
			}

			for _, change := range set.Changes {
				if !codec.Eq(change.StorageKey, key) || !change.HasStorageData {
					continue
				}

				events := &pallets.Events{}
				err = types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, events)
				if err != nil {
					c.errsListening <- fmt.Errorf("events decoder: %w", err)
					continue
				}

				eventsC <- blockEvents{
					Events: events,
					Number: header.Number,
					Hash:   set.Block,
				}
			}
		}
	}(changesC, eventsC)

	// Invoke listeners.
	go func(eventsC <-chan blockEvents) {
		for blockEvents := range eventsC {
			for callback := range c.eventsListeners {
				(*callback)(blockEvents.Events, blockEvents.Number, blockEvents.Hash)
			}

			if afterBlock != nil {
				afterBlock(blockEvents.Number, blockEvents.Hash)
			}
		}
	}(eventsC)

	once := sync.Once{}
	c.cancelListening = func() {
		once.Do(func() {
			sub.Unsubscribe()
			cancelled.Store(true)
			wg.Wait()
			close(c.errsListening)
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

type blockEvents struct {
	Events *pallets.Events
	Hash   types.Hash
	Number types.BlockNumber
}
