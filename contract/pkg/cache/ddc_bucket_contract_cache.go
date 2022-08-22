package cache

import (
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
	}
)

func CreateDdcBucketContractCache(ddcBucketContract bucket.DdcBucketContract) DdcBucketContractCache {
	bucketCache := cache.New(defaultExpiration, cleanupInterval)
	nodeCache := cache.New(defaultExpiration, cleanupInterval)

	return &ddcBucketContractCached{ddcBucketContract: ddcBucketContract, bucketCache: bucketCache, nodeCache: nodeCache}
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

func toString(value uint32) string {
	return strconv.FormatUint(uint64(value), 10)
}
