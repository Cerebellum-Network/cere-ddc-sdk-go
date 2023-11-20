package pkg

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pkg/pallets"
)

type BlockchainApi struct {
	*gsrpc.SubstrateAPI

	DdcClusters  *pallets.DdcClustersApi
	DdcCustomers *pallets.DdcCustomersApi
}

func NewBlockchainApi(substrateApi *gsrpc.SubstrateAPI) (*BlockchainApi, error) {
	meta, err := substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	return &BlockchainApi{
		SubstrateAPI: substrateApi,
		DdcClusters:  pallets.NewDdcClustersApi(substrateApi),
		DdcCustomers: pallets.NewDdcCustomersApi(substrateApi, meta),
	}, nil
}
