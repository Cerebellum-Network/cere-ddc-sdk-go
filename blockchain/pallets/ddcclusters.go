package pallets

import (
	"fmt"
	"math"
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/hash"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/centrifuge/go-substrate-rpc-client/v4/xxhash"
)

type Cluster struct {
	ClusterId ClusterId
	ManagerId types.AccountID
	ReserveId types.AccountID
	Props     ClusterProps
}

type ClustersNodes map[ClusterId][]NodePubKey

type ClusterProps struct {
	NodeProviderAuthContract types.AccountID
}

// Events
type (
	EventDdcClustersClusterCreated struct {
		Phase     types.Phase
		ClusterId ClusterId
		Topics    []types.Hash
	}

	EventDdcClustersClusterNodeAdded struct {
		Phase      types.Phase
		ClusterId  ClusterId
		NodePubKey NodePubKey
		Topics     []types.Hash
	}

	EventDdcClustersClusterNodeRemoved struct {
		Phase      types.Phase
		ClusterId  ClusterId
		NodePubKey NodePubKey
		Topics     []types.Hash
	}

	EventDdcClustersClusterParamsSet struct {
		Phase     types.Phase
		ClusterId ClusterId
		Topics    []types.Hash
	}

	EventDdcClustersClusterGovParamsSet struct {
		Phase     types.Phase
		ClusterId ClusterId
		Topics    []types.Hash
	}
)

type DdcClustersApi interface {
	GetClustersNodes(clusterId ClusterId) ([]NodePubKey, error)
	SubscribeNewClusterCreated() (*NewEventSubscription[EventDdcClustersClusterCreated], error)
	SubscribeNewClusterNodeAdded() (*NewEventSubscription[EventDdcClustersClusterNodeAdded], error)
}

type ddcClustersEventsSubs struct {
	clusterCreated      map[int]subscriber[EventDdcClustersClusterCreated]
	clusterNodeAdded    map[int]subscriber[EventDdcClustersClusterNodeAdded]
	clusterNodeRemoved  map[int]subscriber[EventDdcClustersClusterNodeRemoved]
	clusterParamsSet    map[int]subscriber[EventDdcClustersClusterParamsSet]
	clusterGovParamsSet map[int]subscriber[EventDdcClustersClusterGovParamsSet]
}

type ddcClustersApi struct {
	substrateApi *gsrpc.SubstrateAPI

	clustersNodesKey []byte

	subs *ddcClustersEventsSubs
	mu   sync.Mutex
}

func NewDdcClustersApi(
	substrateApi *gsrpc.SubstrateAPI,
	events <-chan *Events,
) DdcClustersApi {
	clustersNodesKey := append(
		xxhash.New128([]byte("DdcClusters")).Sum(nil),
		xxhash.New128([]byte("ClustersNodes")).Sum(nil)...,
	)

	subs := &ddcClustersEventsSubs{
		clusterCreated:      make(map[int]subscriber[EventDdcClustersClusterCreated]),
		clusterNodeAdded:    make(map[int]subscriber[EventDdcClustersClusterNodeAdded]),
		clusterNodeRemoved:  make(map[int]subscriber[EventDdcClustersClusterNodeRemoved]),
		clusterParamsSet:    make(map[int]subscriber[EventDdcClustersClusterParamsSet]),
		clusterGovParamsSet: make(map[int]subscriber[EventDdcClustersClusterGovParamsSet]),
	}

	api := &ddcClustersApi{
		substrateApi:     substrateApi,
		clustersNodesKey: clustersNodesKey,
		subs:             subs,
		mu:               sync.Mutex{},
	}

	go func() {
		for blockEvents := range events {
			for _, e := range blockEvents.DdcClusters_ClusterCreated {
				api.mu.Lock()
				for i, sub := range api.subs.clusterCreated {
					select {
					case <-sub.done:
						delete(api.subs.clusterCreated, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcClusters_ClusterNodeAdded {
				api.mu.Lock()
				for i, sub := range api.subs.clusterNodeAdded {
					select {
					case <-sub.done:
						delete(api.subs.clusterNodeAdded, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}
		}
	}()

	return api
}

func (api *ddcClustersApi) GetClustersNodes(clusterId ClusterId) ([]NodePubKey, error) {
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

	nodesKeys := make([]NodePubKey, len(keys))
	for i, key := range keys {
		var nodePubKey NodePubKey

		// Decode SCALE-encoded NodePubKey from the secondary key:
		// 	- 16 bytes - Blake2_128 hash,
		// 	- 1 byte - enum variant,
		// 	- 32 - node public key length (as long StoragePubKey is AccountId32 type).
		if err := codec.Decode(key[len(moduleMethodPrefix1Key)+16:len(moduleMethodPrefix1Key)+16+1+32], &nodePubKey); err != nil {
			return nil, err
		}

		nodesKeys[i] = nodePubKey
	}

	return nodesKeys, nil
}

func (api *ddcClustersApi) SubscribeNewClusterCreated() (*NewEventSubscription[EventDdcClustersClusterCreated], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.clusterCreated == nil {
		api.subs.clusterCreated = make(map[int]subscriber[EventDdcClustersClusterCreated])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.clusterCreated[i]; !ok {
			idx = i
			break
		}
		if i == math.MaxInt {
			return nil, fmt.Errorf("can't create %d+1 subscriber", len(api.subs.clusterCreated))
		}
	}

	sub := subscriber[EventDdcClustersClusterCreated]{
		ch:   make(chan EventDdcClustersClusterCreated),
		done: make(chan struct{}),
	}

	api.subs.clusterCreated[idx] = sub

	return &NewEventSubscription[EventDdcClustersClusterCreated]{
		ch:   sub.ch,
		done: sub.done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.clusterCreated, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcClustersApi) SubscribeNewClusterNodeAdded() (*NewEventSubscription[EventDdcClustersClusterNodeAdded], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.clusterNodeAdded == nil {
		api.subs.clusterNodeAdded = make(map[int]subscriber[EventDdcClustersClusterNodeAdded])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.clusterNodeAdded[i]; !ok {
			idx = i
			break
		}
		if i == math.MaxInt {
			return nil, fmt.Errorf("can't create %d+1 subscriber", len(api.subs.clusterNodeAdded))
		}
	}

	sub := subscriber[EventDdcClustersClusterNodeAdded]{
		ch:   make(chan EventDdcClustersClusterNodeAdded),
		done: make(chan struct{}),
	}

	api.subs.clusterNodeAdded[idx] = sub

	return &NewEventSubscription[EventDdcClustersClusterNodeAdded]{
		ch:   sub.ch,
		done: sub.done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.clusterNodeAdded, idx)
			api.mu.Unlock()
		},
	}, nil
}
