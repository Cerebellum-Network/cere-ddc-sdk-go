package cache

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
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

func (m *mockedDdcBucketContract) ClusterGet(clusterId bucket.ClusterId) (*bucket.ClusterInfo, error) {
	args := m.Called(clusterId)
	return args.Get(0).(*bucket.ClusterInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) NodeGet(nodeKey bucket.NodeKey) (*bucket.NodeInfo, error) {
	args := m.Called(nodeKey)
	return args.Get(0).(*bucket.NodeInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) CdnNodeGet(nodeKey bucket.CdnNodeKey) (*bucket.CdnNodeInfo, error) {
	args := m.Called(nodeKey)
	return args.Get(0).(*bucket.CdnNodeInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) BucketGet(bucketId bucket.BucketId) (*bucket.BucketInfo, error) {
	args := m.Called(bucketId)
	return args.Get(0).(*bucket.BucketInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) AccountGet(account bucket.AccountId) (*bucket.Account, error) {
	args := m.Called(account)
	return args.Get(0).(*bucket.Account), args.Error(1)
}

func (m *mockedDdcBucketContract) CdnNodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*bucket.CdnNodeListInfo, error) {
	args := m.Called(offset, limit, filterProviderId)
	return args.Get(0).(*bucket.CdnNodeListInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) NodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*bucket.NodeListInfo, error) {
	args := m.Called(offset, limit, filterProviderId)
	return args.Get(0).(*bucket.NodeListInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterCreate(keyPair signature.KeyringPair, ctx context.Context, cluster *bucket.NewCluster) (blockHash types.Hash, err error) {
	return types.Hash{}, nil
}

func (d *mockedDdcBucketContract) AddContractEventHandler(event string, handler func(interface{})) error {
	return nil
}

func (d *mockedDdcBucketContract) GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry {
	return nil
}

func (m *mockedDdcBucketContract) AdminGrantPermission(grantee bucket.AccountId, permission string) error {
	return nil
}

func (m *mockedDdcBucketContract) AdminRevokePermission(grantee bucket.AccountId, permission string) error {
	return nil
}

func (m *mockedDdcBucketContract) AdminTransferCdnNodeOwnership(cdnNodeKey bucket.CdnNodeKey, newOwner bucket.AccountId) error {
	return nil
}

func (m *mockedDdcBucketContract) AdminTransferNodeOwnership(nodeKey bucket.NodeKey, newOwner bucket.AccountId) error {
	return nil
}

func (m *mockedDdcBucketContract) CdnNodeCreate(nodeKey bucket.NodeKey, params bucket.CDNNodeParams) error {
	return nil
}

func (m *mockedDdcBucketContract) NodeCreate(nodeKey bucket.NodeKey, params bucket.Params, capacity bucket.Resource, rent bucket.Rent) (key bucket.NodeKey, err error) {
	args := m.Called(nodeKey, params, capacity, rent)
	accountId, err := types.NewAccountIDFromHexString(args.Get(0).(string))
	if err != nil {
		log.Fatalf("Failed to create AccountID from hex string: %v", err)
		return types.AccountID{}, err
	}
	key = *accountId
	return key, args.Error(1)
}

func (m *mockedDdcBucketContract) CdnNodeRemove(nodeKey bucket.CdnNodeKey) error {
	args := m.Called(nodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) NodeRemove(nodeKey bucket.NodeKey) error {
	args := m.Called(nodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) CdnNodeSetParams(nodeKey bucket.CdnNodeKey, params bucket.CDNNodeParams) error {
	args := m.Called(nodeKey, params)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterAddCdnNode(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey) error {
	args := m.Called(clusterId, cdnNodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterAddNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, vNodes [][]bucket.Token) error {
	args := m.Called(clusterId, nodeKey, vNodes)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterList(offset types.U32, limit types.U32, filterManagerId types.OptionAccountID) (*bucket.ClusterListInfo, error) {
	args := m.Called(offset, limit, filterManagerId)
	return args.Get(0).(*bucket.ClusterListInfo), args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterRemoveNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey) error {
	args := m.Called(clusterId, nodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterRemove(clusterId bucket.ClusterId) error {
	args := m.Called(clusterId)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterRemoveCdnNode(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey) error {
	args := m.Called(clusterId, cdnNodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterReplaceNode(clusterId bucket.ClusterId, vNodes [][]bucket.Token, newNodeKey bucket.NodeKey) error {
	args := m.Called(clusterId, vNodes, newNodeKey)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterResetNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, vNodes [][]bucket.Token) error {
	args := m.Called(clusterId, nodeKey, vNodes)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterSetCdnNodeStatus(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey, statusInCluster string) error {
	args := m.Called(clusterId, cdnNodeKey, statusInCluster)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterSetNodeStatus(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, statusInCluster string) error {
	args := m.Called(clusterId, nodeKey, statusInCluster)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) ClusterSetParams(clusterId bucket.ClusterId, params bucket.Params) error {
	args := m.Called(clusterId, params)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) GrantTrustedManagerPermission(managerId bucket.AccountId) error {
	args := m.Called(managerId)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) HasPermission(account bucket.AccountId, permission string) (bool, error) {
	args := m.Called(account, permission)
	return true, args.Error(1)
}

func (m *mockedDdcBucketContract) NodeSetParams(nodeKey bucket.NodeKey, params bucket.Params) error {
	args := m.Called(nodeKey, params)
	return args.Error(1)
}

func (m *mockedDdcBucketContract) RevokeTrustedManagerPermission(managerId bucket.AccountId) error {
	args := m.Called(managerId)
	return args.Error(1)
}

// TODO: implement yhe underlying methods
func (m *mockedDdcBucketContract) AccountDeposit() error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) AccountBond(bondAmount bucket.Balance) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) AccountUnbond(bondAmount bucket.Balance) error {
	panic("implement me")
}
func (m *mockedDdcBucketContract) AccountGetUsdPerCere() (balance bucket.Balance, err error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) AccountSetUsdPerCere(usdPerCere bucket.Balance) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) AccountWithdrawUnbonded() error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) GetAccounts() ([]types.AccountID, error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketCreate(bucketParams bucket.BucketParams, clusterId bucket.ClusterId, ownerId types.OptionAccountID) (bucketId bucket.BucketId, err error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketChangeOwner(bucketId bucket.BucketId, ownerId bucket.AccountId) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketAllocIntoCluster(bucketId bucket.BucketId, resource bucket.Resource) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketSetResourceCap(bucketId bucket.BucketId, newResourceCap bucket.Resource) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketSettlePayment(bucketId bucket.BucketId) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketChangeParams(bucketId bucket.BucketId, bucketParams bucket.BucketParams) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketList(offset types.U32, limit types.U32, ownerId types.OptionAccountID) (*bucket.BucketListInfo, error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketListForAccount(ownerId bucket.AccountId) ([]*bucket.Bucket, error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketSetAvailability(bucketId bucket.BucketId, publicAvailability bool) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) GetBucketWriters(bucketId bucket.BucketId) ([]types.AccountID, error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) GetBucketReaders(bucketId bucket.BucketId) ([]types.AccountID, error) {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketSetWriterPerm(bucketId bucket.BucketId, writer bucket.AccountId) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketRevokeWriterPerm(bucketId bucket.BucketId, writer bucket.AccountId) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketSetReaderPerm(bucketId bucket.BucketId, reader bucket.AccountId) error {
	panic("implement me")
}

func (m *mockedDdcBucketContract) BucketRevokeReaderPerm(bucketId bucket.BucketId, reader bucket.AccountId) error {
	panic("implement me")
}

func TestBucketGet(t *testing.T) {
	//given
	ddcBucketContract := &mockedDdcBucketContract{}
	testSubject := &ddcBucketContractCached{bucketCache: cache.New(defaultExpiration, cleanupInterval), ddcBucketContract: ddcBucketContract}
	result := &bucket.BucketInfo{}
	ddcBucketContract.On("BucketGet", types.NewU32(1)).Return(result, nil).Once()

	//when
	bucket, err := testSubject.BucketGet(types.NewU32(1))

	//then
	assert.NoError(t, err)
	assert.Equal(t, result, bucket)
	ddcBucketContract.AssertExpectations(t)
}

func TestBucketGetCached(t *testing.T) {
	//given
	ddcBucketContract := &mockedDdcBucketContract{}
	testSubject := &ddcBucketContractCached{bucketCache: cache.New(defaultExpiration, cleanupInterval), ddcBucketContract: ddcBucketContract}
	result := &bucket.BucketInfo{BucketId: types.NewU32(1)}
	ddcBucketContract.On("BucketGet", types.NewU32(1)).Return(result, nil).Once()
	_, _ = testSubject.BucketGet(types.NewU32(1))

	//when
	bucket, err := testSubject.BucketGet(types.NewU32(1))

	//then
	assert.NoError(t, err)
	assert.Equal(t, result, bucket)
	ddcBucketContract.AssertExpectations(t)
	ddcBucketContract.AssertNumberOfCalls(t, "BucketGet", 1)
}

// func TestCDNNodeList(t *testing.T) {
// 	//given
//     ddcBucketContract := &mockedDdcBucketContract{}
//     testSubject := &ddcBucketContractCached{bucketCache: cache.New(defaultExpiration, cleanupInterval), ddcBucketContract: ddcBucketContract}
//     result := []*bucket.CdnNodeInfo{}
//     ddcBucketContract.On("CdnNodeGet", 0, 1, "filterManagerId").Return(result, nil).Once()

//     //when
//     nodes, err := testSubject.CdnNodeList(0, 1, "filterManagerId")

//     //then
//     assert.NoError(t, err)
//     assert.Equal(t, result, nodes)
//     ddcBucketContract.AssertExpectations(t)
// }
//
