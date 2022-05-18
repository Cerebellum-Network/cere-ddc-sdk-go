package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"time"
)

type Cluster struct {
	ManagerId        types.AccountID
	VNodes           []uint32
	ResourcePerVNode uint32
	ResourceUsed     uint32
	Revenues         types.U128
	TotalRent        types.U128
}

type ClusterStatus struct {
	ClusterId uint32
	Cluster   Cluster
	Params    string
}

type Node struct {
	ProviderId    types.AccountID
	RentPerMonth  types.U128
	FreeResources uint32
}

type NodeStatus struct {
	NodeId uint32
	Node   Node
	Params string
}

type Bucket struct {
	OwnerId          types.AccountID
	ClusterId        uint32
	ResourceReserved uint32
}

type Schedule struct {
	Rate   types.U128
	Offset types.U128
}

type BucketStatus struct {
	BucketId           uint32
	Bucket             Bucket
	Params             string
	WriterIds          []types.AccountID
	RentCoveredUntilMs uint64
}

func (s *BucketStatus) RentExpired() bool {
	return s.RentCoveredUntilMs < uint64(time.Now().UnixMilli())
}
