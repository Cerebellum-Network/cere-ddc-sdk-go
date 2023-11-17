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
	ClusterID ddcprimitives.ClusterID
	ManagerID types.AccountID
	ReserveID types.AccountID
	Props     ClusterProps
}

type ClustersNodes map[ddcprimitives.ClusterID][]ddcprimitives.NodePubKey

type ClusterProps struct {
	NodeProviderAuthContract types.AccountID
}

type DDCClustersAPI struct {
	substrateAPI *gsrpc.SubstrateAPI

	clustersNodesKey []byte
}

func NewDDCClustersAPI(substrateAPI *gsrpc.SubstrateAPI) *DDCClustersAPI {
	clustersNodesKey := append(
		xxhash.New128([]byte("DdcClusters")).Sum(nil),
		xxhash.New128([]byte("ClustersNodes")).Sum(nil)...,
	)

	return &DDCClustersAPI{
		substrateAPI:     substrateAPI,
		clustersNodesKey: clustersNodesKey,
	}
}

func (api *DDCClustersAPI) GetClustersNodes(clusterID ddcprimitives.ClusterID) ([]ddcprimitives.NodePubKey, error) {
	clusterIDbytes, err := codec.Encode(clusterID)
	if err != nil {
		return nil, err
	}
	hasher, err := hash.NewBlake2b128Concat(nil)
	if err != nil {
		return nil, err
	}
	if _, err := hasher.Write(clusterIDbytes); err != nil {
		return nil, err
	}

	moduleMethodPrefix1Key := append(
		api.clustersNodesKey,
		hasher.Sum(nil)...,
	)

	queryKey := types.NewStorageKey(moduleMethodPrefix1Key)
	keys, err := api.substrateAPI.RPC.State.GetKeysLatest(queryKey)
	if err != nil {
		return nil, err
	}

	nodesKeys := make([]ddcprimitives.NodePubKey, len(keys))
	for i, key := range keys {
		var nodePubKey ddcprimitives.NodePubKey

		// Decode SCALE-encoded NodePubKey from the secondary key:
		// 	- 16 bytes - Blake2_128 hash,
		// 	- 1 byte - enum variant,
		// 	- 32 - node public key length (as long as CDNPubKey and StoragePubKey are of the same AccountID32 type).
		if err := codec.Decode(key[len(moduleMethodPrefix1Key)+16:len(moduleMethodPrefix1Key)+16+1+32], &nodePubKey); err != nil {
			return nil, err
		}

		nodesKeys[i] = nodePubKey
	}

	return nodesKeys, nil
}
