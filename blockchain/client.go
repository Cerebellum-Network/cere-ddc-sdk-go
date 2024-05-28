package blockchain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/exec"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/retriever"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/state"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pallets"
)

var errCancelled = errors.New("cancelled")

type EventsListener func(events []*parser.Event, blockNumber types.BlockNumber, blockHash types.Hash)

type Client struct {
	*gsrpc.SubstrateAPI

	mu              sync.Mutex
	eventsListeners map[*EventsListener]struct{}

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
		DdcClusters:     pallets.NewDdcClustersApi(substrateApi, meta),
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
	after func(blockNumber types.BlockNumber, blockHash types.Hash),
) (context.CancelFunc, <-chan error, error) {
	if !atomic.CompareAndSwapUint32(&c.isListening, 0, 1) {
		return c.cancelListening, c.errsListening, nil
	}

	sub, err := c.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		return nil, nil, fmt.Errorf("subscribe new heads: %w", err)
	}

	retriever, err := retriever.NewEventRetriever(
		parser.NewEventParser(),
		state.NewEventProvider(c.RPC.State),
		c.RPC.State,
		registry.NewFactory(),
		exec.NewRetryableExecutor[*types.StorageDataRaw](exec.WithMaxRetryCount(0)),
		exec.NewRetryableExecutor[[]*parser.Event](exec.WithMaxRetryCount(0)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("event retriever: %w", err)
	}

	c.errsListening = make(chan error)

	liveHeadersC := sub.Chan()
	histHeadersC := make(chan types.Header)
	var wg sync.WaitGroup

	// Query historical headers.
	var cancelled atomic.Value
	cancelled.Store(false)
	wg.Add(1)
	go func(beginBlock types.BlockNumber, live <-chan types.Header, hist chan types.Header) {
		defer wg.Done()
		defer close(hist)

		firstLiveHeader := <-live // the first live header is the last historical header

		for block := beginBlock; block < firstLiveHeader.Number; {
			var header *types.Header
			err := retryUntilCancelled(func() error {
				blockHash, err := c.RPC.Chain.GetBlockHash(uint64(block))
				if err != nil {
					c.errsListening <- fmt.Errorf("get historical block hash: %w", err)
					return err
				}

				header, err = c.RPC.Chain.GetHeader(blockHash)
				if err != nil {
					c.errsListening <- fmt.Errorf("get historical header: %w", err)
					return err
				}

				return nil
			}, &cancelled)
			if err != nil {
				if err == errCancelled {
					return
				}
				continue
			}

			hist <- *header

			block++
		}

		hist <- firstLiveHeader
	}(begin, liveHeadersC, histHeadersC)

	// Sequence historical and live headers.
	headersC := make(chan types.Header)
	wg.Add(1)
	go func(hist, live <-chan types.Header, headersC chan types.Header) {
		defer wg.Done()
		defer close(headersC)

		for header := range hist {
			headersC <- header
		}

		for header := range live {
			headersC <- header
		}
	}(histHeadersC, liveHeadersC, headersC)

	// Retrieve events skipping blocks before 'begin'.
	eventsC := make(chan blockEvents)
	wg.Add(1)
	go func(headersC <-chan types.Header, eventsC chan blockEvents) {
		defer wg.Done()
		defer close(eventsC)

		for header := range headersC {
			if header.Number < begin {
				continue
			}

			var hash types.Hash
			var events []*parser.Event
			err := retryUntilCancelled(func() error {
				var err error
				hash, err = c.RPC.Chain.GetBlockHash(uint64(header.Number))
				if err != nil {
					c.errsListening <- fmt.Errorf("get block hash: %w", err)
					return err
				}

				events, err = retriever.GetEvents(hash)
				if err != nil {
					c.errsListening <- fmt.Errorf("events retriever: %w", err)
					return err
				}

				return nil
			}, &cancelled)
			if err != nil {
				continue
			}

			eventsC <- blockEvents{
				Events: events,
				Hash:   hash,
				Number: header.Number,
			}
		}
	}(headersC, eventsC)

	// Invoke listeners.
	go func(eventsC <-chan blockEvents) {
		for blockEvents := range eventsC {
			for callback := range c.eventsListeners {
				(*callback)(blockEvents.Events, blockEvents.Number, blockEvents.Hash)
			}

			if after != nil {
				after(blockEvents.Number, blockEvents.Hash)
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
	Events []*parser.Event
	Hash   types.Hash
	Number types.BlockNumber
}

func retryUntilCancelled(f func() error, cancelled *atomic.Value) error {
	expbackoff := backoff.NewExponentialBackOff()
	expbackoff.MaxElapsedTime = 0 // never stop
	expbackoff.InitialInterval = 10 * time.Second
	expbackoff.Multiplier = 2
	expbackoff.MaxInterval = 10 * time.Minute

	ff := func() error {
		if cancelled.Load().(bool) {
			return backoff.Permanent(errCancelled)
		}
		return f()
	}

	return backoff.Retry(ff, expbackoff)
}
