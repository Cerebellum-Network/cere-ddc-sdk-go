package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/hash"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/centrifuge/go-substrate-rpc-client/v4/xxhash"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pkg/ddcprimitives"
)

type Cluster struct {
	ClusterId ddcprimitives.ClusterId
	ManagerId types.AccountID
	ReserveId types.AccountID
	Props     ClusterProps
}

type ClustersNodes map[ddcprimitives.ClusterId][]ddcprimitives.NodePubKey

type ClusterProps struct {
	NodeProviderAuthContract types.AccountID
}

type DdcClustersApi struct {
	substrateApi *gsrpc.SubstrateAPI

	clustersNodesKey []byte
}

func NewDdcClustersApi(substrateApi *gsrpc.SubstrateAPI) *DdcClustersApi {
	clustersNodesKey := append(
		xxhash.New128([]byte("DdcClusters")).Sum(nil),
		xxhash.New128([]byte("ClustersNodes")).Sum(nil)...,
	)

	return &DdcClustersApi{
		substrateApi:     substrateApi,
		clustersNodesKey: clustersNodesKey,
	}
}

func (api *DdcClustersApi) GetClustersNodes(clusterId ddcprimitives.ClusterId) ([]ddcprimitives.NodePubKey, error) {
	clusterIdBytes, err := codec.Encode(clusterId)
	if err != nil {
		return nil, err
	}
	hasher, err := hash.NewBlake2b128Concat(nil)
	if err != nil {
		return nil, err
	}
	if _, err := hasher.Write(clusterIdBytes); err != nil {
		return nil, err
	}

	moduleMethodPrefix1Key := append(
		api.clustersNodesKey,
		hasher.Sum(nil)...,
	)

	queryKey := types.NewStorageKey(moduleMethodPrefix1Key)
	keys, err := api.substrateApi.RPC.State.GetKeysLatest(queryKey)
	if err != nil {
		return nil, err
	}

	nodesKeys := make([]ddcprimitives.NodePubKey, len(keys))
	for i, key := range keys {
		var nodePubKey ddcprimitives.NodePubKey

		// Decode SCALE-encoded NodePubKey from the secondary key:
		// 	- 16 bytes - Blake2_128 hash,
		// 	- 1 byte - enum variant,
		// 	- 32 - node public key length (as long as CdnPubKey and StoragePubKey are of the same AccountId32 type).
		if err := codec.Decode(key[len(moduleMethodPrefix1Key)+16:len(moduleMethodPrefix1Key)+16+1+32], &nodePubKey); err != nil {
			return nil, err
		}

		nodesKeys[i] = nodePubKey
	}

	return nodesKeys, nil
}
