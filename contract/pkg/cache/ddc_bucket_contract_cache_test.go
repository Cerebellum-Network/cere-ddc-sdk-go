package cache

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockedDdcBucketContract struct {
	mock.Mock
}

func (m *mockedDdcBucketContract) GetContractAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockedDdcBucketContract) GetLastAccessTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *mockedDdcBucketContract) ClusterGet(clusterId uint32) (*bucket.ClusterStatus, error) {
	args := m.Called(clusterId)
	return args.Get(0).(*bucket.ClusterStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) NodeGet(nodeId uint32) (*bucket.NodeStatus, error) {
	args := m.Called(nodeId)
	return args.Get(0).(*bucket.NodeStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) BucketGet(bucketId uint32) (*bucket.BucketStatus, error) {
	args := m.Called(bucketId)
	return args.Get(0).(*bucket.BucketStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) AccountGet(account types.AccountID) (*bucket.Account, error) {
	args := m.Called(account)
	return args.Get(0).(*bucket.Account), args.Error(1)
}

func TestBucketGet(t *testing.T) {
	//given
	ddcBucketContract := &mockedDdcBucketContract{}
	testSubject := &ddcBucketContractCached{bucketCache: cache.New(defaultExpiration, cleanupInterval), ddcBucketContract: ddcBucketContract}
	result := &bucket.BucketStatus{}
	ddcBucketContract.On("BucketGet", uint32(1)).Return(result, nil).Once()

	//when
	bucket, err := testSubject.BucketGet(uint32(1))

	//then
	assert.NoError(t, err)
	assert.Equal(t, result, bucket)
	ddcBucketContract.AssertExpectations(t)
}

func TestBucketGetCached(t *testing.T) {
	//given
	ddcBucketContract := &mockedDdcBucketContract{}
	testSubject := &ddcBucketContractCached{bucketCache: cache.New(defaultExpiration, cleanupInterval), ddcBucketContract: ddcBucketContract}
	result := &bucket.BucketStatus{BucketId: uint32(1)}
	ddcBucketContract.On("BucketGet", uint32(1)).Return(result, nil).Once()
	_, _ = testSubject.BucketGet(uint32(1))

	//when
	bucket, err := testSubject.BucketGet(uint32(1))

	//then
	assert.NoError(t, err)
	assert.Equal(t, result, bucket)
	ddcBucketContract.AssertExpectations(t)
	ddcBucketContract.AssertNumberOfCalls(t, "BucketGet", 1)
}
