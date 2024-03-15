package blockchain

import (
	"fmt"
	"math"
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

	eventsListeners map[int]EventsListener
	mu              sync.Mutex
	isListening     uint32
	stopListening   func()
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
		eventsListeners: make(map[int]EventsListener),
		DdcClusters:     pallets.NewDdcClustersApi(substrateApi),
		DdcCustomers:    pallets.NewDdcCustomersApi(substrateApi, meta),
		DdcNodes:        pallets.NewDdcNodesApi(substrateApi, meta),
		DdcPayouts:      pallets.NewDdcPayoutsApi(substrateApi, meta),
	}, nil
}

func (c *Client) StartEventsListening() (func(), <-chan error, error) {
	if !atomic.CompareAndSwapUint32(&c.isListening, 0, 1) {
		return c.stopListening, c.errsListening, nil
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
				c.processSystemEventsStorageChanges(
					set.Changes,
					meta,
					key,
					set.Block,
				)
			}
		}
	}()

	once := sync.Once{}
	c.stopListening = func() {
		once.Do(func() {
			done <- struct{}{}
			sub.Unsubscribe()
			c.isListening = 0
		})
	}

	return c.stopListening, c.errsListening, nil
}

// RegisterEventsListener subscribes given callback to blockchain events. There is a begin parameter which
// can be used to get events from blocks older than the latest block. If begin is greater than the latest
// block number, the listener will start from the latest block. Subscription on new events starts
// immediately and does not wait until the older blocks events are processed. Rare cases of events
// duplication are possible.
func (c *Client) RegisterEventsListener(begin types.BlockNumber, callback EventsListener) (func(), error) {
	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := c.eventsListeners[i]; !ok {
			idx = i
			break
		}
		if i == math.MaxInt {
			return nil, fmt.Errorf("too many events listeners")
		}
	}

	c.mu.Lock()
	c.eventsListeners[idx] = callback
	c.mu.Unlock()

	stopped := false

	go func() {
		latestHeader, err := c.RPC.Chain.GetHeaderLatest()
		if err != nil {
			c.errsListening <- fmt.Errorf("get latest header: %w", err)
			return
		}

		if begin >= latestHeader.Number {
			return
		}

		meta, err := c.RPC.State.GetMetadataLatest() // TODO: update each runtime upgrade
		if err != nil {
			c.errsListening <- fmt.Errorf("get metadata: %w", err)
			return
		}

		key, err := types.CreateStorageKey(meta, "System", "Events")
		if err != nil {
			c.errsListening <- fmt.Errorf("create storage key: %w", err)
			return
		}

		beginHash, err := c.RPC.Chain.GetBlockHash(uint64(begin))
		if err != nil {
			c.errsListening <- fmt.Errorf("get block hash: %w", err)
			return
		}

		latestHash, err := c.RPC.Chain.GetBlockHashLatest()
		if err != nil {
			c.errsListening <- fmt.Errorf("get latest block hash: %w", err)
			return
		}

		changesSets, err := c.RPC.State.QueryStorage([]types.StorageKey{key}, beginHash, latestHash)
		if err != nil {
			c.errsListening <- fmt.Errorf("storage changes query: %w", err)
			return
		}

		for _, set := range changesSets {
			header, err := c.RPC.Chain.GetHeader(set.Block)
			if err != nil {
				c.errsListening <- fmt.Errorf("get header: %w", err)
				return
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

				if stopped {
					return
				}

				callback(events, header.Number, set.Block)
			}
		}
	}()

	once := sync.Once{}
	stop := func() {
		once.Do(func() {
			c.mu.Lock()
			stopped = true
			delete(c.eventsListeners, idx)
			c.mu.Unlock()
		})
	}

	return stop, nil
}

func (c *Client) processSystemEventsStorageChanges(
	changes []types.KeyValueOption,
	meta *types.Metadata,
	storageKey types.StorageKey,
	blockHash types.Hash,
) {
	header, err := c.RPC.Chain.GetHeader(blockHash)
	if err != nil {
		c.errsListening <- fmt.Errorf("get header: %w", err)
		return
	}

	for _, change := range changes {
		if !codec.Eq(change.StorageKey, storageKey) || !change.HasStorageData {
			continue
		}

		events := &pallets.Events{}
		err = types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, events)
		if err != nil {
			c.errsListening <- fmt.Errorf("events decoder: %w", err)
			continue
		}

		c.mu.Lock()
		for _, callback := range c.eventsListeners {
			go callback(events, header.Number, blockHash)
		}
		c.mu.Unlock()
	}
}
