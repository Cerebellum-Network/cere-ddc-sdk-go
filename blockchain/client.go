package blockchain

import (
	"context"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/exec"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/retriever"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/state"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"golang.org/x/sync/errgroup"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pallets"
)

// Stop events listening when no new events received for this time.
const EventsListeningTimeout = 60 * time.Second

type EventsListener func(events []*parser.Event, blockNumber types.BlockNumber, blockHash types.Hash) error

type Client struct {
	*gsrpc.SubstrateAPI

	mu              sync.Mutex
	eventsListeners map[*EventsListener]struct{}

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

// ListenEvents listens for blockchain events and sequentially calls registered events listeners to
// process incoming events. It starts from the block begin and calls callback after when all events
// listeners already called on a block events.
//
// ListenEvents always returns a non-nil error from a registered events listener or a callback
// after.
func (c *Client) ListenEvents(
	ctx context.Context,
	begin types.BlockNumber,
	after func(blockNumber types.BlockNumber, blockHash types.Hash) error,
) error {
	sub, err := c.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		return err
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
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	liveHeadersC := sub.Chan()
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()

	// Query historical headers.
	histHeadersC := make(chan types.Header)
	g.Go(func() error {
		defer close(histHeadersC)

		firstLiveHeader, ok := <-liveHeadersC // first live header will be the last historical
		if !ok {
			return ctx.Err()
		}

		for block := begin; block < firstLiveHeader.Number; block++ {
			blockHash, err := c.RPC.Chain.GetBlockHash(uint64(block))
			if err != nil {
				return err
			}

			header, err := c.RPC.Chain.GetHeader(blockHash)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case histHeadersC <- *header:
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case histHeadersC <- firstLiveHeader:
		}

		return nil
	})

	// Sequence historical and live headers.
	headersC := make(chan types.Header, 2)
	g.Go(func() error {
		defer close(headersC)

		for header := range histHeadersC {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case headersC <- header:
			}
		}

		for header := range liveHeadersC {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case headersC <- header:
			}
		}

		return nil
	})

	// Retrieve events skipping blocks before 'begin'.
	eventsC := make(chan blockEvents, 2)
	g.Go(func() error {
		defer close(eventsC)

		for header := range headersC {
			if header.Number < begin {
				continue
			}

			hash, err := c.RPC.Chain.GetBlockHash(uint64(header.Number))
			if err != nil {
				return err
			}

			events, err := retriever.GetEvents(hash)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case eventsC <- blockEvents{
				Events: events,
				Hash:   hash,
				Number: header.Number,
			}:
			}
		}

		return nil
	})

	// Invoke listeners.
	g.Go(func() error {
		for {
			select {
			case blockEvents := <-eventsC:
				for callback := range c.eventsListeners {
					err := (*callback)(blockEvents.Events, blockEvents.Number, blockEvents.Hash)
					if err != nil {
						return err
					}
				}

				if after != nil {
					err := after(blockEvents.Number, blockEvents.Hash)
					if err != nil {
						return err
					}
				}
			// Watchdog for the websocket. It silently hangs sometimes with no error nor new events. In
			// all Cere blockchain runtimes we have `pallet-timestamp` which makes at least one event
			// (System.ExtrinsicSuccess for the timestamp.set extrinsic) per block.
			case <-time.After(EventsListeningTimeout):
				return context.DeadlineExceeded
			}
		}
	})

	return g.Wait()
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
