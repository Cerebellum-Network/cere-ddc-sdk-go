package pkg

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pkg/pallets"
)

type BlockchainAPI struct {
	*gsrpc.SubstrateAPI

	DDCClusters  *pallets.DDCClustersAPI
	DDCCustomers *pallets.DDCCustomersAPI
}

func NewBlockchainAPI(substrateAPI *gsrpc.SubstrateAPI) (*BlockchainAPI, error) {
	meta, err := substrateAPI.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	return &BlockchainAPI{
		SubstrateAPI: substrateAPI,
		DDCClusters:  pallets.NewDDCClustersAPI(substrateAPI),
		DDCCustomers: pallets.NewDDCCustomersAPI(substrateAPI, meta),
	}, nil
}
