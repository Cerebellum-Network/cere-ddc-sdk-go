package blockchain

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/retriever"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/state"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pallets"
)

type Client struct {
	*gsrpc.SubstrateAPI
	eventRetriever retriever.EventRetriever

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
	eventRetriever, _ := retriever.NewDefaultEventRetriever(
		state.NewEventProvider(substrateApi.RPC.State),
		substrateApi.RPC.State,
	)

	return &Client{
		SubstrateAPI:   substrateApi,
		eventRetriever: eventRetriever,
		DdcClusters:    pallets.NewDdcClustersApi(substrateApi),
		DdcCustomers:   pallets.NewDdcCustomersApi(substrateApi, meta),
		DdcNodes:       pallets.NewDdcNodesApi(substrateApi, meta),
		DdcPayouts:     pallets.NewDdcPayoutsApi(substrateApi, meta),
	}, nil
}
