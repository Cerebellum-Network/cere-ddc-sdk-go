package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	log "github.com/sirupsen/logrus"
)

var accounts = []string{
	// ed 25519
	// privateKey "0x38a538d3d890bfe8f76dc9bf578e215af16fd3d684666f72db0bc0a22bc1d05b"
	"5FJDBC3jJbWX48PyfpRCo7pKsFwSy4Mzj5t39PfXixD5jMgy",
	// sr 25519
	// privateKey "0x2cf8a6819aa7f2a2e7a62ce8cf0dca2aca48d87b2001652de779f43fecbc5a03"
	"5G1Jb8qPFxPrNb7C9L4d3QWsjiKpfpwTBX1L6M1Wqb5t3oUk",
	// ed 25519
	// privateKey "0x93e0153dc0f0bbee868dc00d8d05ddae260e01d418746443fa190c8eacd9544c"
	"5DoxVJMBeYHfukDQx5G4w9yoTc72cEhVpJD9v1KiTkkr4iJX",
}

var writerIds = getAccountIDs(accounts)

type (
	Node struct {
		Key             string
		Url             string
		StatusInCluster string
	}

	Cluster struct {
		Id          uint32
		NodesVNodes []NodeVNodes
		Params      string
	}

	NodeVNodes struct {
		NodeKey string
		VNodes  []uint64
	}

	CdnNode struct {
		Key    string        `json:"key"`
		Params CDNNodeParams `json:"params"`
	}

	CDNNodeParams struct {
		Url      string `json:"url"`
		Size     int    `json:"size"`
		Location string `json:"location"`
	}

	ddcBucketContractMock struct {
		accountId      string
		apiUrl         string
		lastAccessTime time.Time
		nodes          []Node
		clusters       []Cluster
		cdnNodes       []CdnNode
	}
)

func MapTestNodesVNodes(nodes []NodeVNodes) []bucket.NodeVNodesInfo {
	var nodesVNodes []bucket.NodeVNodesInfo
	for _, node := range nodes {
		nodeKey, err := types.NewAccountIDFromHexString(node.NodeKey)
		if err != nil {
			log.Printf("ERROR DECODING THE ACCOUNT ID err: %v", err)
		}
		nodeVNodes := bucket.NodeVNodesInfo{
			NodeKey: *nodeKey,
			VNodes:  MapTokens(node.VNodes),
		}
		nodesVNodes = append(nodesVNodes, nodeVNodes)
	}
	return nodesVNodes
}

func MapTokens(tokens []uint64) []bucket.Token {
	var result []bucket.Token
	for _, token := range tokens {
		result = append(result, types.NewU64(token))
	}
	return result
}

func CreateDdcBucketContractMock(apiUrl string, accountId string, nodes []Node, clusters []Cluster, cdnNodes []CdnNode) bucket.DdcBucketContract {
	log.Info("DDC Bucket contract configured [MOCK]")
	return &ddcBucketContractMock{
		accountId:      accountId,
		apiUrl:         apiUrl,
		nodes:          nodes,
		clusters:       clusters,
		cdnNodes:       cdnNodes,
		lastAccessTime: time.Now(),
	}
}

func (d *ddcBucketContractMock) BucketGet(bucketId bucket.BucketId) (*bucket.BucketInfo, error) {
	if bucketId == 0 || len(d.clusters)*2 < int(bucketId) {
		return nil, errors.New("unknown bucket")
	}

	clusterId := uint32(bucketId)
	if int(bucketId) > len(d.clusters) {
		clusterId -= uint32(len(d.clusters))
	}

	return CreateBucket(bucketId, clusterId, "", writerIds), nil
}

func (d *ddcBucketContractMock) ClusterGet(clusterId bucket.ClusterId) (*bucket.ClusterInfo, error) {
	for _, cluster := range d.clusters {
		if cluster.Id == uint32(clusterId) {
			return &bucket.ClusterInfo{
				ClusterId: clusterId,
				Cluster: bucket.Cluster{
					ManagerId:        types.AccountID{},
					Params:           cluster.Params,
					ResourcePerVNode: 32,
					ResourceUsed:     0,
					Revenues:         types.NewU128(*big.NewInt(1)),
					TotalRent:        types.NewU128(*big.NewInt(1)),
				},
				NodesVNodes: MapTestNodesVNodes(cluster.NodesVNodes),
			}, nil
		}
	}

	available := []uint32{}
	for _, cluster := range d.clusters {
		available = append(available, cluster.Id)
	}

	return nil, fmt.Errorf("unknown cluster with id %v | available clusters are: %v", clusterId, available)
}

func (d *ddcBucketContractMock) NodeGet(nodeKey bucket.NodeKey) (*bucket.NodeInfo, error) {
	for _, node := range d.nodes {
		if strings.TrimPrefix(node.Key, "0x") == strings.TrimPrefix(nodeKey.ToHexString(), "0x") {
			return &bucket.NodeInfo{
				Key: nodeKey,
				Node: bucket.Node{
					ProviderId:      types.AccountID{},
					RentPerMonth:    types.NewU128(*big.NewInt(1)),
					Params:          `{"url":"` + node.Url + `"}`,
					StatusInCluster: types.NewOptionU8(types.NewU8((bucket.NodeStatusesInClusterMap[node.StatusInCluster]))), //types.NewOptionBytes([]byte{bucket.NodeStatusesInClusterMap[node.StatusInCluster]}),
					FreeResources:   100,
				},
			}, nil
		}
	}

	available := []string{}
	for _, node := range d.nodes {
		available = append(available, node.Key)
	}

	return nil, fmt.Errorf("unknown node with key %v | available nodes are: %v", nodeKey, available)
}

func (d *ddcBucketContractMock) CdnNodeGet(nodeKey bucket.CdnNodeKey) (*bucket.CdnNodeInfo, error) {
	for _, node := range d.cdnNodes {
		if strings.TrimPrefix(node.Key, "0x") == strings.TrimPrefix(nodeKey.ToHexString(), "0x") {
			params, err := json.Marshal(node.Params)
			if err != nil {
				return nil, err
			}
			return &bucket.CdnNodeInfo{
				Key: nodeKey,
				Node: bucket.CdnNode{
					ProviderId:           types.AccountID{},
					UndistributedPayment: types.NewU128(*big.NewInt(1)),
					Params:               string(params),
				},
			}, nil
		}
	}

	return nil, errors.New("unknown node")
}

func (d *ddcBucketContractMock) AccountGet(account types.AccountID) (*bucket.Account, error) {
	for _, acc := range writerIds {
		if acc == account {
			return &bucket.Account{
				Bonded:            types.NewU128(*big.NewInt(100000)),
				UnbondedTimestamp: uint64(time.Now().UnixMilli()),
			}, nil
		}
	}

	return nil, fmt.Errorf("account doesn't exist %x | available nodes are: %v", account, writerIds)
}

func (d *ddcBucketContractMock) GetApiUrl() string {
	return d.apiUrl
}

func (d *ddcBucketContractMock) GetAccountId() string {
	return d.accountId
}

func (d *ddcBucketContractMock) GetLastAccessTime() time.Time {
	return d.lastAccessTime
}

func (d *ddcBucketContractMock) GetContractAddress() string {
	return "mock_ddc_bucket"
}

func (d *ddcBucketContractMock) AddContractEventHandler(event string, handler func(interface{})) error {
	return nil
}

func CreateBucket(bucketId bucket.BucketId, clusterId uint32, bucketParams string, writerIds []types.AccountID) *bucket.BucketInfo {
	return &bucket.BucketInfo{
		BucketId: bucketId,
		Bucket: bucket.Bucket{
			OwnerId:            writerIds[0],
			ClusterId:          types.NewU32(clusterId),
			ResourceReserved:   32,
			PublicAvailability: types.NewU32(clusterId) != bucketId,
			GasConsumptionCap:  math.MaxUint32,
		},
		Params:             bucketParams,
		WriterIds:          writerIds,
		RentCoveredUntilMs: uint64(time.Now().UnixMilli() + time.Hour.Milliseconds()),
	}
}

func getAccountIDs(ss58Addresses []string) []types.AccountID {
	accountIDs := make([]types.AccountID, len(ss58Addresses))
	for i, address := range ss58Addresses {
		if accountID, err := pkg.DecodeAccountIDFromSS58(address); err != nil {
			log.Fatal("Failed decode private key ed25519")
		} else {
			accountIDs[i] = accountID
		}
	}

	return accountIDs
}

func (d *ddcBucketContractMock) GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry {
	return nil
}

func (d *ddcBucketContractMock) ClusterCreate(cluster *bucket.NewCluster) (clusterId bucket.ClusterId, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterAddNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, vNodes [][]bucket.Token) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterRemoveNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterResetNode(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, vNodes [][]bucket.Token) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterReplaceNode(clusterId bucket.ClusterId, vNodes [][]bucket.Token, newNodeKey bucket.NodeKey) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterAddCdnNode(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterRemoveCdnNode(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterSetParams(clusterId bucket.ClusterId, params bucket.Params) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterRemove(clusterId bucket.ClusterId) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterSetNodeStatus(clusterId bucket.ClusterId, nodeKey bucket.NodeKey, statusInCluster string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterSetCdnNodeStatus(clusterId bucket.ClusterId, cdnNodeKey bucket.CdnNodeKey, statusInCluster string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterList(offset uint32, limit uint32, filterManagerId types.OptionAccountID) (*bucket.ClusterListInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeCreate(nodeKey bucket.NodeKey, params bucket.Params, capacity bucket.Resource) (key bucket.NodeKey, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeRemove(nodeKey bucket.NodeKey) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeSetParams(nodeKey bucket.NodeKey, params bucket.Params) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeList(offset uint32, limit uint32, filterProviderId types.OptionAccountID) (*bucket.NodeListInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CdnNodeCreate(nodeKey bucket.CdnNodeKey, params bucket.CDNNodeParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CdnNodeRemove(nodeKey bucket.CdnNodeKey) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CdnNodeSetParams(nodeKey bucket.CdnNodeKey, params bucket.CDNNodeParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CdnNodeList(offset uint32, limit uint32, filterProviderId types.OptionAccountID) (*bucket.CdnNodeListInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) HasPermission(account bucket.AccountId, permission string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) GrantTrustedManagerPermission(managerId bucket.AccountId) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) RevokeTrustedManagerPermission(managerId bucket.AccountId) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminGrantPermission(grantee bucket.AccountId, permission string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminRevokePermission(grantee bucket.AccountId, permission string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminTransferNodeOwnership(nodeKey bucket.NodeKey, newOwner bucket.AccountId) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminTransferCdnNodeOwnership(cdnNodeKey bucket.CdnNodeKey, newOwner bucket.AccountId) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AccountDeposit() error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AccountBond(balance bucket.Balance) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AccountUnbond(bondAmount bucket.Balance) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AccountGetUsdPerCere() (bucket.Balance, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AccountSetUsdPerCere(balance bucket.Balance) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AccountWithdrawUnbonded() error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) GetAccounts() ([]bucket.AccountId, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketCreate(bucketParams bucket.BucketParams, clusterId bucket.ClusterId, ownerId types.OptionAccountID) (bucketId bucket.BucketId, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketChangeOwner(bucketId bucket.BucketId, ownerId types.OptionAccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketAllocIntoCluster(bucketId bucket.BucketId, resource bucket.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketSettlePayment(bucketId bucket.BucketId) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketChangeParams(bucketId bucket.BucketId, bucketParams bucket.BucketParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketList(offset uint32, limit uint32, filterOnwerId types.OptionAccountID) (*bucket.BucketListInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketListForAccount(ownerId types.OptionAccountID) ([]*bucket.Bucket, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketSetAvailability(bucketId bucket.BucketId, publicAvailability bool) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketSetResourceCap(bucketId bucket.BucketId, newResourceCap bucket.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) GetBucketWriters(bucketId bucket.BucketId) ([]bucket.AccountId, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) GetBucketReaders(bucketId bucket.BucketId) ([]bucket.AccountId, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketSetWriterPerm(bucketId bucket.BucketId, writer types.OptionAccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketRevokeWriterPerm(bucketId bucket.BucketId, writer types.OptionAccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketSetReaderPerm(bucketId bucket.BucketId, reader types.OptionAccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) BucketRevokeReaderPerm(bucketId bucket.BucketId, reader types.OptionAccountID) error {
	//TODO implement me
	panic("implement me")
}
