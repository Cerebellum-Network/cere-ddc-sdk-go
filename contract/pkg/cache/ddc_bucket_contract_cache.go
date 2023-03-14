package cache

import (
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/golang/groupcache/singleflight"
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
		Flush()
		EvictCluster(clusterId uint32)
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
		clusterCache        *cache.Cache
		clusterSingleFlight singleflight.Group
	}

	BucketCacheParameters struct {
		BucketCacheExpiration time.Duration
		BucketCacheCleanUp    time.Duration

		NodeCacheExpiration time.Duration
		NodeCacheCleanUp    time.Duration

		AccountCacheExpiration time.Duration
		AccountCacheCleanUp    time.Duration

		ClusterCacheExpiration time.Duration
		ClusterCacheCleanUp    time.Duration
	}
)

func CreateDdcBucketContractCache(ddcBucketContract bucket.DdcBucketContract, parameters BucketCacheParameters) DdcBucketContractCache {
	bucketCache := cache.New(cacheDurationOrDefault(parameters.BucketCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.BucketCacheCleanUp, cleanupInterval))
	nodeCache := cache.New(cacheDurationOrDefault(parameters.NodeCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.NodeCacheCleanUp, cleanupInterval))
	accountCache := cache.New(cacheDurationOrDefault(parameters.AccountCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.AccountCacheCleanUp, cleanupInterval))
	clusterCache := cache.New(cacheDurationOrDefault(parameters.ClusterCacheExpiration, defaultExpiration), cacheDurationOrDefault(parameters.ClusterCacheCleanUp, cleanupInterval))

	return &ddcBucketContractCached{
		ddcBucketContract: ddcBucketContract,
		bucketCache:       bucketCache,
		nodeCache:         nodeCache,
		accountCache:      accountCache,
		clusterCache:      clusterCache,
	}
}

func (d *ddcBucketContractCached) ClusterGet(clusterId uint32) (*bucket.ClusterStatus, error) {
	key := toString(clusterId)
	result, err := d.clusterSingleFlight.Do(key, func() (interface{}, error) {
		if cached, ok := d.clusterCache.Get(key); ok {
			return cached, nil
		}

		value, err := d.ddcBucketContract.ClusterGet(clusterId)
		if err != nil {
			return nil, err
		}

		d.clusterCache.SetDefault(key, value)
		return value, nil
	})

	resp, _ := result.(*bucket.ClusterStatus)
	return resp, err
}

func (d *ddcBucketContractCached) EvictCluster(clusterId uint32) {
	d.clusterCache.Delete(toString(clusterId))
}

func (d *ddcBucketContractCached) NodeGet(nodeId uint32) (*bucket.NodeStatus, error) {
	key := toString(nodeId)
	result, err := d.nodeSingleFlight.Do(key, func() (interface{}, error) {
		if cached, ok := d.nodeCache.Get(key); ok {
			return cached, nil
		}

		value, err := d.ddcBucketContract.NodeGet(nodeId)
		if err != nil {
			return nil, err
		}

		d.nodeCache.SetDefault(key, value)
		return value, nil
	})

	resp, _ := result.(*bucket.NodeStatus)
	return resp, err
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

func (d *ddcBucketContractCached) Flush() {
	d.bucketCache.Flush()
	d.nodeCache.Flush()
	d.accountCache.Flush()
	d.clusterCache.Flush()
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
