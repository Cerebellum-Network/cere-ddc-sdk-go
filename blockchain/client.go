package blockchain

import (
	"fmt"
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pallets"
)

type Client struct {
	*gsrpc.SubstrateAPI
	subs map[string]chan *pallets.Events

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

	subs := make(map[string]chan *pallets.Events)
	subs["DdcClusters"] = make(chan *pallets.Events)
	subs["DdcCustomers"] = make(chan *pallets.Events)
	subs["DdcPayouts"] = make(chan *pallets.Events)

	return &Client{
		SubstrateAPI: substrateApi,
		subs:         subs,
		DdcClusters: pallets.NewDdcClustersApi(
			substrateApi,
			subs["DdcClusters"],
		),
		DdcCustomers: pallets.NewDdcCustomersApi(substrateApi, meta, subs["DdcCustomers"]),
		DdcNodes:     pallets.NewDdcNodesApi(substrateApi, meta),
		DdcPayouts:   pallets.NewDdcPayoutsApi(substrateApi, meta, subs["DdcPayouts"]),
	}, nil
}

func (c *Client) StartEventsListening() (func(), <-chan error, error) {
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
	errCh := make(chan error)

	go func() {
		for {
			select {
			case <-done:
				return
			case set := <-sub.Chan():
				for _, change := range set.Changes {
					if !codec.Eq(change.StorageKey, key) || !change.HasStorageData {
						continue
					}

					events := &pallets.Events{}
					err = types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, events)
					if err != nil {
						errCh <- fmt.Errorf("events listener: %w", err)
					}

					for _, module := range c.subs {
						module <- events
					}
				}
			}
		}
	}()

	once := sync.Once{}
	stop := func() {
		once.Do(func() {
			done <- struct{}{}
			sub.Unsubscribe()
		})
	}

	return stop, errCh, nil
}
