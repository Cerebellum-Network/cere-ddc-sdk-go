package mock

import (
	"errors"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	log "github.com/sirupsen/logrus"
	"math"
	"math/big"
	"time"
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
var buckets = []*bucket.BucketStatus{
	CreateBucket(1, `{"replication":1}`, writerIds),
	CreateBucket(2, `{"replication":2}`, writerIds),
	CreateBucket(3, `{"replication":3}`, writerIds),
}

type (
	Node struct {
		Id  uint32
		Url string
	}

	Cluster struct {
		Id     uint32
		VNodes [][]uint64
		Nodes  []uint32
		Params string
	}

	ddcBucketContractMock struct {
		accountId      string
		apiUrl         string
		lastAccessTime time.Time
		nodes          []Node
		clusters       []Cluster
	}
)

func CreateDdcBucketContractMock(apiUrl string, accountId string, nodes []Node, clusters []Cluster) bucket.DdcBucketContract {
	log.Info("DDC Bucket contract configured [MOCK]")
	return &ddcBucketContractMock{
		accountId:      accountId,
		apiUrl:         apiUrl,
		nodes:          nodes,
		clusters:       clusters,
		lastAccessTime: time.Now(),
	}
}

func (d *ddcBucketContractMock) BucketGet(bucketId uint32) (*bucket.BucketStatus, error) {
	for _, bucket := range buckets {
		if bucket.BucketId == bucketId {
			return bucket, nil
		}
	}

	return nil, errors.New("unknown bucket")
}

func (d *ddcBucketContractMock) ClusterGet(clusterId uint32) (*bucket.ClusterStatus, error) {
	for _, cluster := range d.clusters {
		if cluster.Id == clusterId {
			return &bucket.ClusterStatus{
				ClusterId: clusterId,
				Cluster: bucket.Cluster{
					ManagerId:        types.AccountID{},
					Nodes:            cluster.Nodes,
					VNodes:           cluster.VNodes,
					ResourcePerVNode: 32,
					ResourceUsed:     1,
					Revenues:         types.NewU128(*big.NewInt(1)),
					TotalRent:        types.NewU128(*big.NewInt(1)),
				},
				Params: cluster.Params,
			}, nil
		}
	}

	return nil, errors.New("unknown cluster")
}

func (d *ddcBucketContractMock) NodeGet(nodeId uint32) (*bucket.NodeStatus, error) {
	for _, node := range d.nodes {
		if node.Id == nodeId {
			return &bucket.NodeStatus{
				NodeId: nodeId,
				Node: bucket.Node{
					ProviderId:    types.AccountID{},
					RentPerMonth:  types.NewU128(*big.NewInt(1)),
					FreeResources: 100,
				},
				Params: `{"url":"` + node.Url + `"}`,
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

func CreateBucket(bucketId uint32, bucketParams string, writerIds []types.AccountID) *bucket.BucketStatus {
	return &bucket.BucketStatus{
		BucketId: bucketId,
		Bucket: bucket.Bucket{
			OwnerId:            writerIds[0],
			ClusterId:          0,
			ResourceReserved:   32,
			PublicAvailability: bucketId%2 == 0,
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
