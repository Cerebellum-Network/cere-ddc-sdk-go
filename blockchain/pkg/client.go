package pkg

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pkg/pallets"
)

type BlockchainAPI struct {
	*gsrpc.SubstrateAPI

	DDCClusters *pallets.DDCClustersAPI
}

func NewBlockchainAPI(substrateAPI *gsrpc.SubstrateAPI) *BlockchainAPI {
	return &BlockchainAPI{
		SubstrateAPI: substrateAPI,
		DDCClusters:  pallets.NewDDCClustersAPI(substrateAPI),
	}
}
