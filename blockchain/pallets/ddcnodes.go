package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

const (
	MaxCdnNodeParamsLen     = 2048
	MaxStorageNodeParamsLen = 2048
	MaxHostLen              = 255
)

type CdnNode struct {
	PubKey     CdnNodePubKey
	ProviderId types.AccountID
	ClusterId  types.Option[ClusterId]
	Props      CdnNodeProps
}

// TODO: `Host` is not `[MaxHostLen]types.U8` because the original `BoundedVec<_, MaxHostLen>`
// decoding returns an unexpected error "error: expected more bytes, but could not decode any
// more".
type CdnNodeProps struct {
	Host     []types.U8
	HttpPort types.U16
	GrpcPort types.U16
	P2pPort  types.U16
}

type StorageNode struct {
	PubKey     StorageNodePubKey
	ProviderId types.AccountID
	ClusterId  types.Option[ClusterId]
	Props      StorageNodeProps
}

// TODO: `Host` is not `[MaxHostLen]types.U8` because the original `BoundedVec<_, MaxHostLen>`
// decoding returns an unexpected error "error: expected more bytes, but could not decode any
// more".
type StorageNodeProps struct {
	Host     []types.U8
	HttpPort types.U16
	GrpcPort types.U16
	P2pPort  types.U16
}

type DdcNodesApi struct {
	substrateApi *gsrpc.SubstrateAPI
	meta         *types.Metadata
}

func NewDdcNodesApi(substrateAPI *gsrpc.SubstrateAPI, meta *types.Metadata) *DdcNodesApi {
	return &DdcNodesApi{
		substrateAPI,
		meta,
	}
}

func (api *DdcNodesApi) GetStorageNodes(pubkey StorageNodePubKey) (types.Option[StorageNode], error) {
	maybeNode := types.NewEmptyOption[StorageNode]()

	bytes, err := codec.Encode(pubkey)
	if err != nil {
		return maybeNode, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcNodes", "StorageNodes", bytes)
	if err != nil {
		return maybeNode, err
	}

	var node StorageNode
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &node)
	if !ok || err != nil {
		return maybeNode, err
	}

	maybeNode.SetSome(node)

	return maybeNode, nil
}

func (api *DdcNodesApi) GetCdnNodes(pubkey CdnNodePubKey) (types.Option[CdnNode], error) {
	maybeNode := types.NewEmptyOption[CdnNode]()

	bytes, err := codec.Encode(pubkey)
	if err != nil {
		return maybeNode, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcNodes", "CdnNodes", bytes)
	if err != nil {
		return maybeNode, err
	}

	var node CdnNode
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &node)
	if !ok || err != nil {
		return maybeNode, err
	}

	maybeNode.SetSome(node)

	return maybeNode, nil
}
