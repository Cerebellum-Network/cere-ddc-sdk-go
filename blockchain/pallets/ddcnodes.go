package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

const (
	MaxStorageNodeParamsLen = 2048
	MaxHostLen              = 255
)

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
	Domain   []types.U8
	Ssl      types.Bool
	HttpPort types.U16
	GrpcPort types.U16
	P2pPort  types.U16
	Mode     StorageNodeMode
}

type DdcNodesApi interface {
	GetStorageNodes(pubkey StorageNodePubKey) (types.Option[StorageNode], error)
}

type ddcNodesApi struct {
	substrateApi *gsrpc.SubstrateAPI
}

func NewDdcNodesApi(substrateApi *gsrpc.SubstrateAPI) DdcNodesApi {
	return &ddcNodesApi{substrateApi}
}

func (api *ddcNodesApi) GetStorageNodes(pubkey StorageNodePubKey) (types.Option[StorageNode], error) {
	maybeNode := types.NewEmptyOption[StorageNode]()

	meta, err := api.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return maybeNode, err
	}

	bytes, err := codec.Encode(pubkey)
	if err != nil {
		return maybeNode, err
	}

	key, err := types.CreateStorageKey(meta, "DdcNodes", "StorageNodes", bytes)
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
