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
		d.clearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketAllocatedEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketSettlePaymentEventId, func(raw interface{}) {
		args := raw.(*bucket.BucketSettlePaymentEvent)
		d.clearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketSettlePaymentEventId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.BucketAvailabilityUpdatedId, func(raw interface{}) {
		args := raw.(*bucket.BucketAvailabilityUpdatedEvent)
		d.clearBucketById(args.BucketId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.BucketAvailabilityUpdatedId)
	}
	if err := d.ddcBucketContract.AddContractEventHandler(bucket.DepositEventId, func(raw interface{}) {
		args := raw.(*bucket.DepositEvent)
		d.clearAccountById(args.AccountId)
	}); err != nil {
		return errors.Wrap(err, "Unable to hook event "+bucket.DepositEventId)
	}
	return nil
}

func (d *ddcBucketContractCached) ClusterGet(clusterId uint32) (*bucket.ClusterStatus, error) {
	return d.ddcBucketContract.ClusterGet(clusterId)
}

func (d *ddcBucketContractCached) NodeGet(nodeKey string) (*bucket.NodeStatus, error) {

	result, err := d.nodeSingleFlight.Do(nodeKey, func() (interface{}, error) {
		if cached, ok := d.nodeCache.Get(nodeKey); ok {
			return cached, nil
		}

		value, err := d.ddcBucketContract.NodeGet(nodeKey)
		if err != nil {
			return nil, err
		}

		d.nodeCache.SetDefault(nodeKey, value)
		return value, nil
	})

	resp, _ := result.(*bucket.NodeStatus)
	return resp, err
}

func (d *ddcBucketContractCached) CDNNodeGet(nodeKey string) (*bucket.CDNNodeStatus, error) {
	return d.ddcBucketContract.CDNNodeGet(nodeKey)
}

func (d *ddcBucketContractCached) BucketGet(bucketId uint32) (*bucket.BucketStatus, error) {
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

	resp, _ := result.(*bucket.BucketStatus)
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

func (d *ddcBucketContractCached) clearNodeById(id bucket.NodeId) { //nolint:golint,unused
	d.nodeCache.Delete(toString(id))
}

func (d *ddcBucketContractCached) clearBucketById(id bucket.BucketId) {
	d.bucketCache.Delete(toString(id))
}

func (d *ddcBucketContractCached) clearAccountById(id bucket.AccountId) {
	d.accountCache.Delete(hex.EncodeToString(id[:]))
}

func cacheDurationOrDefault(duration time.Duration, defaultDuration time.Duration) time.Duration {
	if duration > 0 {
		return duration
	}

	return defaultDuration
}

func toString(value uint32) string {
	return strconv.FormatUint(uint64(value), 10)
}

func (d *ddcBucketContractCached) ClusterCreate(cluster *bucket.NewCluster) (clusterId uint32, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterAddNode(clusterId uint32, nodeKey string, vNodes [][]bucket.Token) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterRemoveNode(clusterId uint32, nodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterResetNode(clusterId uint32, nodeKey string, vNodes [][]bucket.Token) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterReplaceNode(clusterId uint32, vNodes [][]bucket.Token, newNodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterAddCdnNode(clusterId uint32, cdnNodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterRemoveCdnNode(clusterId uint32, cdnNodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterSetParams(clusterId uint32, params bucket.Params) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterRemove(clusterId uint32) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterSetNodeStatus(clusterId uint32, nodeKey string, statusInCluster string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterSetCdnNodeStatus(clusterId uint32, cdnNodeKey string, statusInCluster string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) ClusterList(offset uint32, limit uint32, filterManagerId string) []*bucket.ClusterStatus {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) NodeCreate(nodeKey string, params bucket.Params, capacity bucket.Resource) (key string, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) NodeRemove(nodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) NodeSetParams(nodeKey string, params bucket.Params) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) NodeList(offset uint32, limit uint32, filterManagerId string) ([]*bucket.NodeStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) CDNNodeCreate(nodeKey string, params bucket.CDNNodeParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) CDNNodeRemove(nodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) CDNNodeSetParams(nodeKey string, params bucket.CDNNodeParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) CDNNodeList(offset uint32, limit uint32, filterManagerId string) ([]*bucket.CDNNodeStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) HasPermission(account types.AccountID, permission string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) GrantTrustedManagerPermission(managerId types.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) RevokeTrustedManagerPermission(managerId types.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) AdminGrantPermission(grantee types.AccountID, permission string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) AdminRevokePermission(grantee types.AccountID, permission string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) AdminTransferNodeOwnership(nodeKey string, newOwner types.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractCached) AdminTransferCdnNodeOwnership(cdnNodeKey string, newOwner types.AccountID) error {
	//TODO implement me
	panic("implement me")
}
