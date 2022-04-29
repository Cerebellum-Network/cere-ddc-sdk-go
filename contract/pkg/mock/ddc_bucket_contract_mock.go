package mock

import (
	"errors"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	log "github.com/sirupsen/logrus"
	"math/big"
	"time"
)

var buckets = []*pkg.BucketStatus{
	CreateBucket(1, `{"replication":1}`),
	CreateBucket(2, `{"replication":2}`),
	CreateBucket(3, `{"replication":3}`),
}

type (
	Node = struct {
		Id  uint32
		Url string
	}

	Cluster = struct {
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

func CreateDdcBucketContractMock(accountId string, apiUrl string, nodes []Node, clusters []Cluster) pkg.DdcBucketContract {
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

func CreateBucket(bucketId uint32, bucketParams string) *pkg.BucketStatus {
	return &pkg.BucketStatus{
		BucketId: bucketId,
		Bucket: pkg.Bucket{
			OwnerId:          types.AccountID{},
			ClusterId:        0,
			Flow:             pkg.Flow{},
			ResourceReserved: 0,
		},
		Params:             bucketParams,
		WriterIds:          []types.AccountID{},
		RentCoveredUntilMs: uint64(time.Now().UnixMilli() + time.Hour.Milliseconds()),
	}
}
