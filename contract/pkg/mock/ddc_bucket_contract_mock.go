package mock

import (
	"errors"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	patractTypes "github.com/patractlabs/go-patract/types"
	log "github.com/sirupsen/logrus"
	"math/big"
	"time"
)

var writerIds = getAccountIDs([]string{"5FJDBC3jJbWX48PyfpRCo7pKsFwSy4Mzj5t39PfXixD5jMgy"})

var buckets = []*pkg.BucketStatus{
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
		VNodes []uint32
	}

	ddcBucketContractMock struct {
		accountId      string
		apiUrl         string
		lastAccessTime time.Time
		nodes          []Node
		clusters       []Cluster
	}
)

func CreateDdcBucketContractMock(apiUrl string, accountId string, nodes []Node, clusters []Cluster) pkg.DdcBucketContract {
	log.Info("DDC Bucket contract configured [MOCK]")
	return &ddcBucketContractMock{
		accountId:      accountId,
		apiUrl:         apiUrl,
		nodes:          nodes,
		clusters:       clusters,
		lastAccessTime: time.Now(),
	}
}

func (d *ddcBucketContractMock) BucketGet(bucketId uint32) (*pkg.BucketStatus, error) {
	for _, bucket := range buckets {
		if bucket.BucketId == bucketId {
			return bucket, nil
		}
	}

	return nil, errors.New("unknown bucket")
}

func (d *ddcBucketContractMock) ClusterGet(clusterId uint32) (*pkg.ClusterStatus, error) {
	for _, cluster := range d.clusters {
		if cluster.Id == clusterId {
			return &pkg.ClusterStatus{
				ClusterId: clusterId,
				Cluster: pkg.Cluster{
					ManagerId:        types.AccountID{},
					VNodes:           cluster.VNodes,
					ResourcePerVNode: 32,
					ResourceUsed:     0,
					Revenues:         types.NewU128(*big.NewInt(1)),
					TotalRent:        types.NewU128(*big.NewInt(1)),
				},
				Params: "",
			}, nil
		}
	}

	return nil, errors.New("unknown cluster")
}

func (d *ddcBucketContractMock) NodeGet(nodeId uint32) (*pkg.NodeStatus, error) {
	for _, node := range d.nodes {
		if node.Id == nodeId {
			return &pkg.NodeStatus{
				NodeId: nodeId,
				Node: pkg.Node{
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

func (d *ddcBucketContractMock) GetApiUrl() string {
	return d.apiUrl
}

func (d *ddcBucketContractMock) GetAccountId() string {
	return d.accountId
}

func (d *ddcBucketContractMock) GetLastAccessTime() time.Time {
	return d.lastAccessTime
}

func CreateBucket(bucketId uint32, bucketParams string, writerIds []types.AccountID) *pkg.BucketStatus {
	return &pkg.BucketStatus{
		BucketId: bucketId,
		Bucket: pkg.Bucket{
			OwnerId:          writerIds[0],
			ClusterId:        0,
			ResourceReserved: 0,
		},
		Params:             bucketParams,
		WriterIds:          writerIds,
		RentCoveredUntilMs: uint64(time.Now().UnixMilli() + time.Hour.Milliseconds()),
	}
}

func getAccountIDs(ss58Addresses []string) []patractTypes.AccountID {
	accountIDs := make([]patractTypes.AccountID, len(ss58Addresses))
	for i, address := range ss58Addresses {
		if accountID, err := patractTypes.DecodeAccountIDFromSS58(address); err != nil {
			log.Fatal("Failed decode private key ed25519")
		} else {
			accountIDs[i] = accountID
		}
	}

	return accountIDs
}
