package cache

import (
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *mockedDdcBucketContract) NodeGet(nodeKey string) (*bucket.NodeStatus, error) {
	args := m.Called(nodeKey)
	return args.Get(0).(*bucket.NodeStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) CDNClusterGet(clusterId uint32) (*bucket.CDNClusterStatus, error) {
	args := m.Called(clusterId)
	return args.Get(0).(*bucket.CDNClusterStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) CDNNodeGet(nodeKey string) (*bucket.CDNNodeStatus, error) {
	args := m.Called(nodeKey)
	return args.Get(0).(*bucket.CDNNodeStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) BucketGet(bucketId uint32) (*bucket.BucketStatus, error) {
	args := m.Called(bucketId)
	return args.Get(0).(*bucket.BucketStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) AccountGet(account types.AccountID) (*bucket.Account, error) {
	args := m.Called(account)
	return args.Get(0).(*bucket.Account), args.Error(1)
}

func (m *mockedDdcBucketContract) CDNNodeList(offset uint32, limit uint32, filterManagerId string) ([]*bucket.CDNNodeStatus, error) {
	args := m.Called(offset, limit, filterManagerId)
	return args.Get(0).([]*bucket.CDNNodeStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) NodeList(offset uint32, limit uint32, filterManagerId string) ([]*bucket.NodeStatus, error) {
	args := m.Called(offset, limit, filterManagerId)
	return args.Get(0).([]*bucket.NodeStatus), args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterCreate(cluster *bucket.NewCluster) (clusterId uint32, err error) {
	return 0, nil
}

func (d *mockedDdcBucketContract) AddContractEventHandler(event string, handler func(interface{})) error {
	return nil
}

func (d *mockedDdcBucketContract) GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry {
	return nil
}

func (m *mockedDdcBucketContract) AdminGrantPermission(grantee types.AccountID, permission string) error {
	return nil
}

func (m *mockedDdcBucketContract) AdminRevokePermission(grantee types.AccountID, permission string) error {
	return nil
}

func (m *mockedDdcBucketContract) AdminTransferCdnNodeOwnership(cdnNodeKey string, newOwner types.AccountID) error {
	return nil
}

func (m *mockedDdcBucketContract) AdminTransferNodeOwnership(nodeKey string, newOwner types.AccountID) error {
	return nil
}

func (m *mockedDdcBucketContract) CDNNodeCreate(nodeKey string, params bucket.CDNNodeParams) error {
	return nil
}

func (m *mockedDdcBucketContract) NodeCreate(nodeKey string, params bucket.Params, capacity bucket.Resource) (key string, err error) {
	args := m.Called(nodeKey, params, capacity)
	return args.Get(0).(string), args.Error(1)
}

func (m *mockedDdcBucketContract) CDNNodeRemove(nodeKey string) error {
	args := m.Called(nodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) NodeRemove(nodeKey string) error {
	args := m.Called(nodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) CDNNodeSetParams(nodeKey string, params bucket.CDNNodeParams) error {
	args := m.Called(nodeKey, params)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterAddCdnNode(clusterId uint32, cdnNodeKey string) error {
	args := m.Called(clusterId, cdnNodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterAddNode(clusterId uint32, nodeKey string, vNodes [][]bucket.Token) error {
	args := m.Called(clusterId, nodeKey, vNodes)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterList(offset uint32, limit uint32, filterManagerId string) []*bucket.ClusterStatus {
	args := m.Called(offset, limit, filterManagerId)
	return args.Get(0).([]*bucket.ClusterStatus)
}

func (m *mockedDdcBucketContract) ClusterRemoveNode(clusterId uint32, nodeKey string) error {
	args := m.Called(clusterId, nodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterRemove(clusterId uint32) error {
	args := m.Called(clusterId)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterRemoveCdnNode(clusterId uint32, cdnNodeKey string) error {
	args := m.Called(clusterId, cdnNodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterReplaceNode(clusterId uint32, vNodes [][]bucket.Token, newNodeKey string) error {
	args := m.Called(clusterId, vNodes, newNodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterResetNode(clusterId uint32, nodeKey string, vNodes [][]bucket.Token) error {
	args := m.Called(clusterId, nodeKey, vNodes)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterSetCdnNodeStatus(clusterId uint32, cdnNodeKey string, statusInCluster string) error {
	args := m.Called(clusterId, cdnNodeKey, statusInCluster)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterSetNodeStatus(clusterId uint32, nodeKey string, statusInCluster string) error {
	args := m.Called(clusterId, nodeKey, statusInCluster)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterSetParams(clusterId uint32, params bucket.Params) error {
	args := m.Called(clusterId, params)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) GrantTrustedManagerPermission(managerId types.AccountID) error {
	args := m.Called(managerId)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) HasPermission(account types.AccountID, permission string) (bool, error) {
	args := m.Called(account, permission)
	return true, args.Error(1)
}

func (m *mockedDdcBucketContract) NodeSetParams(nodeKey string, params bucket.Params) error {
	args := m.Called(nodeKey, params)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) RevokeTrustedManagerPermission(managerId types.AccountID) error {
	args := m.Called(managerId)
	return args.Error(1)
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
