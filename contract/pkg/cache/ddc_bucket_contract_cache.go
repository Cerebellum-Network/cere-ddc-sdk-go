package cache

import (
	"encoding/hex"
	"strconv"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/golang/groupcache/singleflight"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

const (
	defaultExpiration = 8 * time.Hour
	cleanupInterval   = 1 * time.Hour
)

type (
	DdcBucketContractCache interface {
		HookContractEvents() error
		Clear()
		ClearNodes()
		ClearBuckets()
		ClearAccounts()
		ClearNodeById(id bucket.NodeKey)
		ClearBucketById(id bucket.BucketId)
		ClearAccountById(id bucket.AccountId)
		bucket.DdcBucketContract
	}

	ddcBucketContractCached struct {
		ddcBucketContract   bucket.DdcBucketContract
		bucketCache         *cache.Cache
		bucketSingleFlight  singleflight.Group
		nodeCache           *cache.Cache
		nodeSingleFlight    singleflight.Group
		accountCache        *cache.Cache
		accountSingleFlight singleflight.Group
	}

	BucketCacheParameters struct {
		BucketCacheExpiration time.Duration
		BucketCacheCleanUp    time.Duration

		NodeCacheExpiration time.Duration
		NodeCacheCleanUp    time.Duration

		AccountCacheExpiration time.Duration
		AccountCacheCleanUp    time.Duration
	}
)

func CreateDdcBucketContractCache(ddcBucketContract bucket.DdcBucketContract, parameters BucketCacheParameters) DdcBucketContractCache {
	bucketCache := cache.New(
		cacheDurationOrDefault(parameters.BucketCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.BucketCacheCleanUp, cleanupInterval))
	nodeCache := cache.New(
		cacheDurationOrDefault(parameters.NodeCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.NodeCacheCleanUp, cleanupInterval))
	accountCache := cache.New(
		cacheDurationOrDefault(parameters.AccountCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.AccountCacheCleanUp, cleanupInterval))

	return &ddcBucketContractCached{
		ddcBucketContract: ddcBucketContract,
		bucketCache:       bucketCache,
		nodeCache:         nodeCache,
		accountCache:      accountCache,
	}
}

func (d *ddcBucketContractCached) HookContractEvents() error {
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketAllocatedEventId, func(raw interface{}) {
		args := raw.(*bucket.BucketAllocatedEvent)
		d.ClearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketAllocatedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketSettlePaymentEventId, func(raw interface{}) {
		args := raw.(*bucket.BucketSettlePaymentEvent)
		d.ClearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketSettlePaymentEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketAvailabilityUpdatedId, func(raw interface{}) {
		args := raw.(*bucket.BucketAvailabilityUpdatedEvent)
		d.ClearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketAvailabilityUpdatedId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.DepositEventId, func(raw interface{}) {
		args := raw.(*bucket.DepositEvent)
		d.ClearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.DepositEventId)
	}

	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketCreatedEventId, func(raw interface{}) {
		args := raw.(*bucket.BucketCreatedEvent)
		d.ClearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketCreatedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketParamsSetEventId, func(raw interface{}) {
		args := raw.(*bucket.BucketParamsSetEvent)
		d.ClearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketParamsSetEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterNodeAddedEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterNodeAddedEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterNodeAddedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterNodeRemovedEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterNodeRemovedEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterNodeRemovedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterCdnNodeAddedEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterCdnNodeAddedEvent)
		d.ClearNodeByKey(args.CdnNodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterCdnNodeAddedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterCdnNodeRemovedEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterCdnNodeRemovedEvent)
		d.ClearNodeByKey(args.CdnNodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterCdnNodeRemovedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterNodeStatusSetEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterNodeStatusSetEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterNodeStatusSetEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterCdnNodeStatusSetEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterCdnNodeStatusSetEvent)
		d.ClearNodeByKey(args.CdnNodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterCdnNodeStatusSetEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterNodeReplacedEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterNodeReplacedEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterNodeReplacedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterNodeResetEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterNodeResetEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterNodeResetEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.CdnNodeCreatedEventId, func(raw interface{}) {
		args := raw.(*bucket.CdnNodeCreatedEvent)
		d.ClearNodeByKey(args.CdnNodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.CdnNodeCreatedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.CdnNodeRemovedEventId, func(raw interface{}) {
		args := raw.(*bucket.CdnNodeRemovedEvent)
		d.ClearNodeByKey(args.CdnNodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.CdnNodeRemovedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.CdnNodeParamsSetEventId, func(raw interface{}) {
		args := raw.(*bucket.CdnNodeParamsSetEvent)
		d.ClearNodeByKey(args.CdnNodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.CdnNodeParamsSetEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.NodeRemovedEventId, func(raw interface{}) {
		args := raw.(*bucket.NodeRemovedEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.NodeRemovedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.NodeParamsSetEventId, func(raw interface{}) {
		args := raw.(*bucket.NodeParamsSetEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.NodeParamsSetEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.NodeCreatedEventId, func(raw interface{}) {
		args := raw.(*bucket.NodeCreatedEvent)
		d.ClearNodeByKey(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.NodeCreatedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.GrantPermissionEventId, func(raw interface{}) {
		args := raw.(*bucket.GrantPermissionEvent)
		d.ClearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.GrantPermissionEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.RevokePermissionEventId, func(raw interface{}) {
		args := raw.(*bucket.RevokePermissionEvent)
		d.ClearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.RevokePermissionEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.NodeOwnershipTransferredEventId, func(raw interface{}) {
		args := raw.(*bucket.NodeOwnershipTransferredEvent)
		d.ClearNodeById(args.NodeKey)
		d.ClearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.NodeOwnershipTransferredEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.CdnNodeOwnershipTransferredEventId, func(raw interface{}) {
		args := raw.(*bucket.CdnNodeOwnershipTransferredEvent)
		d.ClearNodeById(args.CdnNodeKey)
		d.ClearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.CdnNodeOwnershipTransferredEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterReserveResourceEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterReserveResourceEvent)
		d.ClearNodeById(args.NodeKey)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterReserveResourceEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterDistributeRevenuesEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterDistributeRevenuesEvent)
		d.ClearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterDistributeRevenuesEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.ClusterDistributeCdnRevenuesEventId, func(raw interface{}) {
		args := raw.(*bucket.ClusterDistributeCdnRevenuesEvent)
		d.ClearAccountById(args.ProviderId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.ClusterDistributeCdnRevenuesEventId)
	}

	return nil
}

func (d *ddcBucketContractCached) ClusterGet(clusterId bucket.ClusterId) (*bucket.ClusterInfo, error) {
	return d.ddcBucketContract.ClusterGet(clusterId)
}

func (d *ddcBucketContractCached) NodeGet(nodeKey bucket.NodeKey) (*bucket.NodeInfo, error) {

	result, err := d.nodeSingleFlight.Do(nodeKey.ToHexString(), func() (interface{}, error) {
		if cached, ok := d.nodeCache.Get(nodeKey.ToHexString()); ok {
			return cached, nil
		}

		value, err := d.ddcBucketContract.NodeGet(nodeKey)
		if err != nil {
			return nil, err
		}

		d.nodeCache.SetDefault(nodeKey.ToHexString(), value)
		return value, nil
	})

	resp, _ := result.(*bucket.NodeInfo)
	return resp, err
}

func (d *ddcBucketContractCached) CdnNodeGet(nodeKey bucket.CdnNodeKey) (*bucket.CdnNodeInfo, error) {
	return d.ddcBucketContract.CdnNodeGet(nodeKey)
}

func (d *ddcBucketContractCached) BucketGet(bucketId bucket.BucketId) (*bucket.BucketInfo, error) {
	key := toString(bucketId)
	result, err := d.bucketSingleFlight.Do(key, func() (interface{}, error) {
		if cached, ok := d.bucketCache.Get(key); ok {
			return cached, nil
		}

		value, err := d.ddcBucketContract.BucketGet(bucketId)
		if err != nil {
			return nil, err
		}

		d.bucketCache.SetDefault(key, value)
		return value, nil
	})

	resp, _ := result.(*bucket.BucketInfo)
	return resp, err
}

func (d *ddcBucketContractCached) AccountGet(account types.AccountID) (*bucket.Account, error) {
	key := hex.EncodeToString(account[:])
	result, err := d.accountSingleFlight.Do(key, func() (interface{}, error) {
		if cached, ok := d.accountCache.Get(key); ok {
			return cached, nil
		}

		value, err := d.ddcBucketContract.AccountGet(account)
		if err != nil {
			return &bucket.Account{}, err
		}

		d.accountCache.SetDefault(key, value)
		return value, nil
	})

	resp, _ := result.(*bucket.Account)
	return resp, err
}

func (d *ddcBucketContractCached) Clear() {
	d.ClearBuckets()
	d.ClearNodes()
	d.ClearAccounts()
}

func (d *ddcBucketContractCached) GetContractAddress() string {
	return d.ddcBucketContract.GetContractAddress()
}

func (d *ddcBucketContractCached) GetLastAccessTime() time.Time {
	return d.ddcBucketContract.GetLastAccessTime()
}

func (d *ddcBucketContractCached) AddContractEventHandler(event string, handler func(interface{})) error {
	return d.ddcBucketContract.AddContractEventHandler(event, handler)
}

func (d *ddcBucketContractCached) GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry {
	return d.ddcBucketContract.GetEventDispatcher()
}

func (d *ddcBucketContractCached) ClearNodes() {
	d.nodeCache.Flush()
}

func (d *ddcBucketContractCached) ClearBuckets() {
	d.bucketCache.Flush()
}

func (d *ddcBucketContractCached) ClearAccounts() {
	d.accountCache.Flush()
}

func (d *ddcBucketContractCached) ClearNodeById(key bucket.NodeKey) { //nolint:golint,unused
	d.nodeCache.Delete(key.ToHexString())
}

func (d *ddcBucketContractCached) ClearNodeByKey(nodeKey bucket.NodeKey) {
	d.nodeCache.Delete(nodeKey.ToHexString())
}

func (d *ddcBucketContractCached) ClearBucketById(id bucket.BucketId) {
	d.bucketCache.Delete(toString(id))
}

func (d *ddcBucketContractCached) ClearAccountById(id bucket.AccountId) {
	d.accountCache.Delete(hex.EncodeToString(id[:]))
}

func cacheDurationOrDefault(duration time.Duration, defaultDuration time.Duration) time.Duration {
	if duration > 0 {
		return duration
	}

	return defaultDuration
}

func isNodeKeyPresent(nodeKeys []bucket.NodeKey, nodeKey bucket.NodeKey) bool {
	for _, key := range nodeKeys {
		if key == nodeKey {
			return true
		}
	}
	return false
}

func toString(value bucket.BucketId) string {
	return strconv.FormatUint(uint64(value), 10)
}

func validateCDNNodeParams(params bucket.CDNNodeParams) error {
	if params.Url == "" {
		return errors.New("Empty CDN node URL.")
	}
	if params.Size <= 0 {
		return errors.New("Invalid CDN node size.")
	}
	if params.Location == "" {
		return errors.New("Empty CDN node location.")
	}

	return nil
}

func (d *ddcBucketContractCached) ClusterCreate(cluster *bucket.NewCluster) (clusterId bucket.ClusterId, err error) {
	clusterId, err = d.ddcBucketContract.ClusterCreate(cluster)

	if err != nil {
		return 0, err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return clusterId, nil
}

func (d *ddcBucketContractCached) ClusterAddNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, vNodes [][]bucket.Token) error {
	if len(vNodes) == 0 {
		return errors.New("No vNodes provided.")
	}

	clusterStatus, responseError := d.ClusterGet(clusterId)
	if responseError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if isNodeKeyPresent(clusterStatus.Cluster.NodesKeys, nodeKey) {
		return bucket.ErrNodeAlreadyExists
	}

	err := d.ddcBucketContract.ClusterAddNode(clusterId, nodeKey, vNodes)
	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) ClusterRemoveNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey) error {
	err := d.ddcBucketContract.ClusterRemoveNode(clusterId, nodeKey)
	if err != nil {
		return err
	}

	// If the node removal from the contract was successful, clear the cached node status.e
	d.ClearNodeByKey(nodeKey)

	return nil
}

func (d *ddcBucketContractCached) ClusterResetNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, vNodes [][]bucket.Token) error {
	clusterStatus, err := d.ClusterGet(clusterId)
	if err != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if !isNodeKeyPresent(clusterStatus.Cluster.NodesKeys, nodeKey) {
		return bucket.ErrNodeDoesNotExist
	}

	responseError := d.ddcBucketContract.ClusterResetNode(clusterId, nodeKey, vNodes)

	if responseError != nil {
		return responseError
	}

	d.ClearNodeByKey(nodeKey)

	return nil
}

func (d *ddcBucketContractCached) ClusterReplaceNode(clusterId bucket.ClusterId, vNodes [][]bucket.Token, newNodeKey bucket.NodeKey) error {
	if len(vNodes) == 0 {
		return errors.New("No vNodes provided.")
	}

	clusterStatus, clusterError := d.ClusterGet(clusterId)
	if clusterError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if isNodeKeyPresent(clusterStatus.Cluster.NodesKeys, newNodeKey) {
		return bucket.ErrNodeAlreadyExists
	}

	err := d.ddcBucketContract.ClusterReplaceNode(clusterId, vNodes, newNodeKey)
	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) ClusterAddCdnNode(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey) error {
	clusterStatus, responseError := d.ClusterGet(clusterId)
	if responseError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if isNodeKeyPresent(clusterStatus.Cluster.CdnNodesKeys, cdnNodeKey) {
		return bucket.ErrCdnNodeAlreadyExists
	}

	err := d.ddcBucketContract.ClusterAddCdnNode(clusterId, cdnNodeKey)
	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) ClusterRemoveCdnNode(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey) error {

	clusterStatus, clusterError := d.ClusterGet(clusterId)
	if clusterError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if !isNodeKeyPresent(clusterStatus.Cluster.CdnNodesKeys, cdnNodeKey) {
		return bucket.ErrCdnNodeDoesNotExist
	}

	err := d.ddcBucketContract.ClusterRemoveCdnNode(clusterId, cdnNodeKey)
	if err != nil {
		return err
	}

	d.ClearNodeByKey(cdnNodeKey)

	return nil
}

func (d *ddcBucketContractCached) ClusterSetParams(clusterId bucket.ClusterId, params bucket.Params) error {
	if params == "" {
		return errors.New("Empty cluster parameters.")
	}

	_, clusterError := d.ClusterGet(clusterId)
	if clusterError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	err := d.ddcBucketContract.ClusterSetParams(clusterId, params)
	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) ClusterRemove(clusterId bucket.ClusterId) error {
	_, clusterError := d.ClusterGet(clusterId)
	if clusterError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	err := d.ddcBucketContract.ClusterRemove(clusterId)
	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) ClusterSetNodeStatus(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, statusInCluster string) error {
	if statusInCluster == "" {
		return errors.New("Empty status in cluster.")
	}

	clusterStatus, clusterError := d.ClusterGet(clusterId)
	if clusterError != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if !isNodeKeyPresent(clusterStatus.Cluster.NodesKeys, nodeKey) {
		return bucket.ErrNodeDoesNotExist
	}

	err := d.ddcBucketContract.ClusterSetNodeStatus(clusterId, nodeKey, statusInCluster)
	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) ClusterSetCdnNodeStatus(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey, statusInCluster string) error {
	if statusInCluster == "" {
		return errors.New("Empty status in cluster.")
	}

	clusterStatus, err := d.ClusterGet(clusterId)
	if err != nil {
		return bucket.ErrClusterDoesNotExist
	}

	if !isNodeKeyPresent(clusterStatus.Cluster.CdnNodesKeys, cdnNodeKey) {
		return bucket.ErrCdnNodeIsNotAddedToCluster
	}

	err = d.ddcBucketContract.ClusterSetCdnNodeStatus(clusterId, cdnNodeKey, statusInCluster)
	if err != nil {
		return err
	}

	d.ClearNodeByKey(cdnNodeKey)

	return nil
}

func (d *ddcBucketContractCached) ClusterList(offset types.U32, limit types.U32, filterManagerId types.OptionAccountID) (*bucket.ClusterListInfo, error) {
	if limit == 0 {
		return nil, errors.New("Invalid limit. Limit must be greater than zero.")
	}

	clusters, err := d.ddcBucketContract.ClusterList(offset, limit, filterManagerId)

	if err != nil {
		return nil, err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return clusters, nil
}

func (d *ddcBucketContractCached) NodeCreate(nodeKey bucket.NodeKey, params bucket.Params, capacity bucket.Resource, rent bucket.Rent) (key bucket.NodeKey, err error) {
	key, err = d.ddcBucketContract.NodeCreate(nodeKey, params, capacity, rent)

	d.ClearNodes()

	return key, err
}

func (d *ddcBucketContractCached) NodeRemove(nodeKey bucket.NodeKey) error {
	err := d.ddcBucketContract.NodeRemove(nodeKey)

	if err != nil {
		return err
	}

	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) NodeSetParams(nodeKey bucket.NodeKey, params bucket.Params) error {
	err := d.ddcBucketContract.NodeSetParams(nodeKey, params)

	if err != nil {
		return err
	}

	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) NodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*bucket.NodeListInfo, error) {
	if limit == 0 {
		return nil, errors.New("Invalid limit. Limit must be greater than zero.")
	}

	nodes, err := d.ddcBucketContract.NodeList(offset, limit, filterProviderId)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (d *ddcBucketContractCached) CdnNodeCreate(nodeKey bucket.CdnNodeKey, params bucket.CDNNodeParams) error {
	err := d.ddcBucketContract.CdnNodeCreate(nodeKey, params)

	if err != nil {
		return err
	}

	d.ClearNodes()

	return err
}

func (d *ddcBucketContractCached) CdnNodeRemove(nodeKey bucket.CdnNodeKey) error {
	err := d.ddcBucketContract.CdnNodeRemove(nodeKey)
	if err != nil {
		return err
	}

	// Clear the corresponding cache since the CDN node data has been modified.
	d.ClearBuckets()
	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) CdnNodeSetParams(nodeKey bucket.CdnNodeKey, params bucket.CDNNodeParams) error {
	if err := validateCDNNodeParams(params); err != nil {
		return errors.Wrap(err, "Invalid CDN node params.")
	}

	err := d.ddcBucketContract.CdnNodeSetParams(nodeKey, params)
	if err != nil {
		return err
	}

	d.ClearNodes()

	return nil
}

func (d *ddcBucketContractCached) CdnNodeList(offset types.U32, limit types.U32, filterManagerId types.OptionAccountID) (*bucket.CdnNodeListInfo, error) {
	if limit == 0 {
		return nil, errors.New("Invalid limit. Limit must be greater than zero.")
	}

	nodes, err := d.ddcBucketContract.CdnNodeList(offset, limit, filterManagerId)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (d *ddcBucketContractCached) HasPermission(account types.AccountID, permission string) (bool, error) {
	if permission == "" {
		return false, errors.New("Empty permission string.")
	}

	return d.ddcBucketContract.HasPermission(account, permission)
}

func (d *ddcBucketContractCached) GrantTrustedManagerPermission(managerId bucket.AccountId) error {
	err := d.ddcBucketContract.GrantTrustedManagerPermission(managerId)

	d.ClearBuckets()
	d.ClearNodes()

	return err
}

func (d *ddcBucketContractCached) RevokeTrustedManagerPermission(managerId bucket.AccountId) error {
	err := d.ddcBucketContract.RevokeTrustedManagerPermission(managerId)

	d.ClearBuckets()
	d.ClearNodes()

	return err
}

func (d *ddcBucketContractCached) AdminGrantPermission(grantee bucket.AccountId, permission string) error {
	if permission == "" {
		return errors.New("Empty permission string.")
	}
	err := d.ddcBucketContract.AdminGrantPermission(grantee, permission)

	d.ClearBuckets()
	d.ClearNodes()

	return err
}

func (d *ddcBucketContractCached) AdminRevokePermission(grantee bucket.AccountId, permission string) error {
	if permission == "" {
		return errors.New("Empty permission string.")
	}
	err := d.ddcBucketContract.AdminRevokePermission(grantee, permission)

	d.ClearBuckets()
	d.ClearNodes()

	return err
}

func (d *ddcBucketContractCached) AdminTransferNodeOwnership(nodeKey bucket.NodeKey, newOwner bucket.AccountId) error {
	err := d.ddcBucketContract.AdminTransferNodeOwnership(nodeKey, newOwner)

	d.ClearBuckets()
	d.ClearNodes()

	return err
}

func (d *ddcBucketContractCached) AdminTransferCdnNodeOwnership(cdnNodeKey bucket.CdnNodeKey, newOwner bucket.AccountId) error {
	err := d.ddcBucketContract.AdminTransferCdnNodeOwnership(cdnNodeKey, newOwner)

	d.ClearBuckets()
	d.ClearNodes()

	return err
}

// TODO implement caching for underlying methods
func (d *ddcBucketContractCached) AccountDeposit() error {
	return d.ddcBucketContract.AccountDeposit()
}

func (d *ddcBucketContractCached) AccountBond(bondAmount bucket.Balance) error {
	return d.ddcBucketContract.AccountBond(bondAmount)
}

func (d *ddcBucketContractCached) AccountUnbond(bondAmount bucket.Balance) error {
	return d.ddcBucketContract.AccountUnbond(bondAmount)
}

func (d *ddcBucketContractCached) AccountGetUsdPerCere() (balance bucket.Balance, err error) {
	return d.ddcBucketContract.AccountGetUsdPerCere()
}

func (d *ddcBucketContractCached) AccountSetUsdPerCere(usdPerCere bucket.Balance) error {
	return d.ddcBucketContract.AccountSetUsdPerCere(usdPerCere)
}

func (d *ddcBucketContractCached) AccountWithdrawUnbonded() error {
	return d.ddcBucketContract.AccountWithdrawUnbonded()
}

func (d *ddcBucketContractCached) GetAccounts() ([]types.AccountID, error) {
	accounts, err := d.ddcBucketContract.GetAccounts()
	return accounts, err
}

func (d *ddcBucketContractCached) BucketCreate(bucketParams bucket.BucketParams, clusterId bucket.ClusterId, ownerId types.OptionAccountID) (bucketId bucket.BucketId, err error) {
	return d.ddcBucketContract.BucketCreate(bucketParams, clusterId, ownerId)
}

func (d *ddcBucketContractCached) BucketChangeOwner(bucketId bucket.BucketId, ownerId bucket.AccountId) error {
	return d.ddcBucketContract.BucketChangeOwner(bucketId, ownerId)
}

func (d *ddcBucketContractCached) BucketAllocIntoCluster(bucketId bucket.BucketId, resource bucket.Resource) error {
	return d.ddcBucketContract.BucketAllocIntoCluster(bucketId, resource)
}

func (d *ddcBucketContractCached) BucketSettlePayment(bucketId bucket.BucketId) error {
	return d.ddcBucketContract.BucketSettlePayment(bucketId)
}

func (d *ddcBucketContractCached) BucketChangeParams(bucketId bucket.BucketId, bucketParams bucket.BucketParams) error {
	return d.ddcBucketContract.BucketChangeParams(bucketId, bucketParams)
}

func (d *ddcBucketContractCached) BucketList(offset types.U32, limit types.U32, filterOwnerId types.OptionAccountID) (*bucket.BucketListInfo, error) {
	return d.ddcBucketContract.BucketList(offset, limit, filterOwnerId)
}

func (d *ddcBucketContractCached) BucketListForAccount(ownerId bucket.AccountId) (*[]bucket.Bucket, error) {
	return d.ddcBucketContract.BucketListForAccount(ownerId)
}

func (d *ddcBucketContractCached) BucketSetAvailability(bucketId bucket.BucketId, publicAvailability bool) error {
	return d.ddcBucketContract.BucketSetAvailability(bucketId, publicAvailability)
}

func (d *ddcBucketContractCached) BucketSetResourceCap(bucketId bucket.BucketId, newResourceCap bucket.Resource) error {
	return d.ddcBucketContract.BucketSetResourceCap(bucketId, newResourceCap)
}

func (d *ddcBucketContractCached) GetBucketWriters(bucketId bucket.BucketId) ([]types.AccountID, error) {
	return d.ddcBucketContract.GetBucketWriters(bucketId)
}

func (d *ddcBucketContractCached) GetBucketReaders(bucketId bucket.BucketId) ([]types.AccountID, error) {
	return d.ddcBucketContract.GetBucketReaders(bucketId)
}

func (d *ddcBucketContractCached) BucketSetWriterPerm(bucketId bucket.BucketId, writer bucket.AccountId) error {
	return d.ddcBucketContract.BucketSetWriterPerm(bucketId, writer)
}

func (d *ddcBucketContractCached) BucketRevokeWriterPerm(bucketId bucket.BucketId, writer bucket.AccountId) error {
	return d.ddcBucketContract.BucketRevokeWriterPerm(bucketId, writer)
}

func (d *ddcBucketContractCached) BucketSetReaderPerm(bucketId bucket.BucketId, reader bucket.AccountId) error {
	return d.ddcBucketContract.BucketSetReaderPerm(bucketId, reader)
}

func (d *ddcBucketContractCached) BucketRevokeReaderPerm(bucketId bucket.BucketId, reader bucket.AccountId) error {
	return d.ddcBucketContract.BucketRevokeReaderPerm(bucketId, reader)
}
