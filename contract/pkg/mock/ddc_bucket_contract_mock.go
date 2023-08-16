package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
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
		Key string
		Url string
		Tag string
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

	CDNNode struct {
		Key    string        `json:"key"`
		Params CDNNodeParams `json:"params"`
	}

	CDNNodeParams struct {
		Url       string `json:"url"`
		Size      int    `json:"size"`
		Location  string `json:"location"`
		PublicKey string `json:"publicKey"`
	}

	CDNCluster struct {
		Id    uint32
		Nodes []string
	}

	ddcBucketContractMock struct {
		accountId      string
		apiUrl         string
		lastAccessTime time.Time
		nodes          []Node
		clusters       []Cluster
		cdnNodes       []CDNNode
		cdnClusters    []CDNCluster
	}
)

func mapNodesVNodes(nodes []NodeVNodes) []bucket.NodeVNodesInfo {
	var nodesVNodes []bucket.NodeVNodesInfo
	for _, node := range nodes {
		nodeVNodes := bucket.NodeVNodesInfo{
			NodeKey: node.NodeKey,
			VNodes:  node.VNodes,
		}
		nodesVNodes = append(nodesVNodes, nodeVNodes)
	}
	return nodesVNodes
}

func CreateDdcBucketContractMock(apiUrl string, accountId string, nodes []Node, clusters []Cluster, cdnNodes []CDNNode, cdnClusters []CDNCluster) bucket.DdcBucketContract {
	log.Info("DDC Bucket contract configured [MOCK]")
	return &ddcBucketContractMock{
		accountId:      accountId,
		apiUrl:         apiUrl,
		nodes:          nodes,
		clusters:       clusters,
		cdnClusters:    cdnClusters,
		cdnNodes:       cdnNodes,
		lastAccessTime: time.Now(),
	}
}

func (d *ddcBucketContractMock) BucketGet(bucketId uint32) (*bucket.BucketStatus, error) {
	if bucketId == 0 || len(d.clusters)*2 < int(bucketId) {
		return nil, errors.New("unknown bucket")
	}

	clusterId := bucketId
	if int(bucketId) > len(d.clusters) {
		clusterId -= uint32(len(d.clusters))
	}

	return CreateBucket(bucketId, clusterId, "", writerIds), nil
}

func (d *ddcBucketContractMock) ClusterGet(clusterId uint32) (*bucket.ClusterStatus, error) {
	for _, cluster := range d.clusters {
		if cluster.Id == clusterId {
			return &bucket.ClusterStatus{
				ClusterId: clusterId,
				Cluster: bucket.Cluster{
					ManagerId:        types.AccountID{},
					Params:           cluster.Params,
					ResourcePerVNode: 32,
					ResourceUsed:     0,
					Revenues:         types.NewU128(*big.NewInt(1)),
					TotalRent:        types.NewU128(*big.NewInt(1)),
				},
				NodesVNodes: mapNodesVNodes(cluster.NodesVNodes),
			}, nil
		}
	}

	available := []uint32{}
	for _, cluster := range d.clusters {
		available = append(available, cluster.Id)
	}

	return nil, fmt.Errorf("unknown cluster with id %v | available clusters are: %v", clusterId, available)
}

func (d *ddcBucketContractMock) NodeGet(nodeKey string) (*bucket.NodeStatus, error) {
	for _, node := range d.nodes {
		if node.Key == nodeKey {
			return &bucket.NodeStatus{
				Key: nodeKey,
				Node: bucket.Node{
					ProviderId:    types.AccountID{},
					RentPerMonth:  types.NewU128(*big.NewInt(1)),
					NodeState:     bucket.NodeTags[node.Tag],
					FreeResources: 100,
				},
				Params: `{"url":"` + node.Url + `"}`,
			}, nil
		}
	}

	available := []string{}
	for _, node := range d.nodes {
		available = append(available, node.Key)
	}

	return nil, fmt.Errorf("unknown node with key %v | available nodes are: %v", nodeKey, available)
}

func (d *ddcBucketContractMock) CDNClusterGet(clusterId uint32) (*bucket.CDNClusterStatus, error) {
	for _, cluster := range d.cdnClusters {
		if cluster.Id == clusterId {
			return &bucket.CDNClusterStatus{
				ClusterId: clusterId,
				CDNCluster: bucket.CDNCluster{
					ManagerId:    types.AccountID{},
					CDNNodes:     cluster.Nodes,
					ResourceUsed: 0,
					Revenues:     types.NewU128(*big.NewInt(1)),
					UsdPerGb:     types.NewU128(*big.NewInt(1)),
				},
			}, nil
		}
	}

	return nil, errors.New("unknown cluster")
}

func (d *ddcBucketContractMock) CDNNodeGet(nodeKey string) (*bucket.CDNNodeStatus, error) {
	for _, node := range d.cdnNodes {
		if node.Key == nodeKey {
			params, err := json.Marshal(node.Params)
			if err != nil {
				return nil, err
			}
			return &bucket.CDNNodeStatus{
				Key: nodeKey,
				Node: bucket.CDNNode{
					ProviderId:           types.AccountID{},
					UndistributedPayment: types.NewU128(*big.NewInt(1)),
				},
				Params: string(params),
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

	return nil, errors.New("account doesn't exist")
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

func CreateBucket(bucketId uint32, clusterId uint32, bucketParams string, writerIds []types.AccountID) *bucket.BucketStatus {
	return &bucket.BucketStatus{
		BucketId: bucketId,
		Bucket: bucket.Bucket{
			OwnerId:            writerIds[0],
			ClusterId:          clusterId,
			ResourceReserved:   32,
			PublicAvailability: clusterId != bucketId,
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

func (d *ddcBucketContractMock) ClusterCreate(cluster *bucket.NewCluster) (clusterId uint32, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterAddNode(clusterId uint32, nodeKey string, vNodes [][]bucket.Token) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterRemoveNode(clusterId uint32, nodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterResetNode(clusterId uint32, nodeKey string, vNodes [][]bucket.Token) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterReplaceNode(clusterId uint32, vNodes [][]bucket.Token, newNodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterAddCdnNode(clusterId uint32, cdnNodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterRemoveCdnNode(clusterId uint32, cdnNodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterSetParams(clusterId uint32, params bucket.Params) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterRemove(clusterId uint32) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterSetNodeStatus(clusterId uint32, nodeKey string, statusInCluster string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterSetCdnNodeStatus(clusterId uint32, cdnNodeKey string, statusInCluster string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) ClusterList(offset uint32, limit uint32, filterManagerId string) []*bucket.ClusterStatus {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeCreate(nodeKey string, params bucket.Params, capacity bucket.Resource) (key string, err error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeRemove(nodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeSetParams(nodeKey string, params bucket.Params) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) NodeList(offset uint32, limit uint32, filterManagerId string) ([]*bucket.NodeStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CDNNodeCreate(nodeKey string, params bucket.CDNNodeParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CDNNodeRemove(nodeKey string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CDNNodeSetParams(nodeKey string, params bucket.CDNNodeParams) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) CDNNodeList(offset uint32, limit uint32, filterManagerId string) ([]*bucket.CDNNodeStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) HasPermission(account types.AccountID, permission string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) GrantTrustedManagerPermission(managerId types.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) RevokeTrustedManagerPermission(managerId types.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminGrantPermission(grantee types.AccountID, permission string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminRevokePermission(grantee types.AccountID, permission string) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminTransferNodeOwnership(nodeKey string, newOwner types.AccountID) error {
	//TODO implement me
	panic("implement me")
}

func (d *ddcBucketContractMock) AdminTransferCdnNodeOwnership(cdnNodeKey string, newOwner types.AccountID) error {
	//TODO implement me
	panic("implement me")
}
