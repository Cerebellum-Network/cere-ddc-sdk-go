package cache

import (
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/patrickmn/go-cache"
	"strconv"
	"time"
)

const (
	defaultExpiration = 8 * time.Hour
	cleanupInterval   = 1 * time.Hour
)

type (
	DdcBucketContractCache interface {
		Clear()
		bucket.DdcBucketContract
	}

	ddcBucketContractCached struct {
		ddcBucketContract bucket.DdcBucketContract
		bucketCache       *cache.Cache
		nodeCache         *cache.Cache
		accountGetCache   *cache.Cache
	}

	BucketCacheParameters struct {
		BucketCacheExpiration time.Duration
		BucketCacheCleanUp    time.Duration

		NodeCacheExpiration time.Duration
		NodeCacheCleanUp    time.Duration

		AccountGetCacheExpiration time.Duration
		AccountGetCacheCleanUp    time.Duration
	}
)

func CreateDdcBucketContractCache(ddcBucketContract bucket.DdcBucketContract, parameters BucketCacheParameters) DdcBucketContractCache {
	bucketCache := cache.New(cacheDurationOrDefault(parameters.BucketCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.BucketCacheCleanUp, cleanupInterval))
	nodeCache := cache.New(cacheDurationOrDefault(parameters.NodeCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.NodeCacheCleanUp, cleanupInterval))
	accountGetCache := cache.New(cacheDurationOrDefault(parameters.AccountGetCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.AccountGetCacheCleanUp, cleanupInterval))

	return &ddcBucketContractCached{
		ddcBucketContract: ddcBucketContract,
		bucketCache:       bucketCache,
		nodeCache:         nodeCache,
		accountGetCache:   accountGetCache,
	}
}

func (d *ddcBucketContractCached) ClusterGet(clusterId uint32) (*bucket.ClusterStatus, error) {
	return d.ddcBucketContract.ClusterGet(clusterId)
}

func (d *ddcBucketContractCached) NodeGet(nodeId uint32) (*bucket.NodeStatus, error) {
	key := toString(nodeId)
	cached, ok := d.nodeCache.Get(key)

	if !ok {
		value, err := d.ddcBucketContract.NodeGet(nodeId)
		if err != nil {
			return nil, err
		}

		d.nodeCache.SetDefault(key, value)
		return value, nil
	}

	return cached.(*bucket.NodeStatus), nil
}

func (d *ddcBucketContractCached) BucketGet(bucketId uint32) (*bucket.BucketStatus, error) {
	key := toString(bucketId)
	cached, ok := d.bucketCache.Get(key)

	if !ok {
		value, err := d.ddcBucketContract.BucketGet(bucketId)
		if err != nil {
			return nil, err
		}

		d.bucketCache.SetDefault(key, value)
		return value, nil
	}

	return cached.(*bucket.BucketStatus), nil
}

func (d *ddcBucketContractCached) AccountGet(account types.AccountID) (*bucket.Account, error) {
	key := hex.EncodeToString(account[:])
	cached, ok := d.accountGetCache.Get(key)

	if !ok {
		value, err := d.ddcBucketContract.AccountGet(account)
		if err != nil {
			return &bucket.Account{}, err
		}

		d.accountGetCache.SetDefault(key, value)
		return value, nil
	}

	return cached.(*bucket.Account), nil
}

func (d *ddcBucketContractCached) Clear() {
	d.bucketCache.Flush()
	d.nodeCache.Flush()
}

func (d *ddcBucketContractCached) GetContractAddress() string {
	return d.ddcBucketContract.GetContractAddress()
}

func (d *ddcBucketContractCached) GetLastAccessTime() time.Time {
	return d.ddcBucketContract.GetLastAccessTime()
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
