package blockchain

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pallets"
)

type Client interface {
	DdcClusters() *pallets.DdcClustersApi
	DdcCustomers() *pallets.DdcCustomersApi
	DdcNodes() *pallets.DdcNodesApi
}

type client struct {
	*gsrpc.SubstrateAPI

	ddcClusters  *pallets.DdcClustersApi
	ddcCustomers *pallets.DdcCustomersApi
	ddcNodes     *pallets.DdcNodesApi
}

func NewClient(url string) (Client, error) {
	substrateApi, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}
	meta, err := substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	return &client{
		SubstrateAPI: substrateApi,
		ddcClusters:  pallets.NewDdcClustersApi(substrateApi),
		ddcCustomers: pallets.NewDdcCustomersApi(substrateApi, meta),
		ddcNodes:     pallets.NewDdcNodesApi(substrateApi, meta),
	}, nil
}

func (c *client) DdcClusters() *pallets.DdcClustersApi {
	return c.ddcClusters
}

func (c *client) DdcCustomers() *pallets.DdcCustomersApi {
	return c.ddcCustomers
}

func (c *client) DdcNodes() *pallets.DdcNodesApi {
	return c.ddcNodes
}
