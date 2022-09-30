package bucket

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
	OwnerId            types.AccountID
	ClusterId          uint32
	ResourceReserved   uint32
	PublicAvailability bool
	GasConsumptionCap  uint32
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

type Account struct {
	//ToDo fill account struct
}

func (a *Account) HasBalance() bool {
	//ToDo add logic
	return true
}

func (b *BucketStatus) RentExpired() bool {
	return b.RentCoveredUntilMs < uint64(time.Now().UnixMilli())
}

func (b *BucketStatus) HasWriteAccess(publicKey []byte) bool {
	address := types.NewAddressFromAccountID(publicKey)

	for _, writerId := range b.WriterIds {
		if writerId == address.AsAccountID {
			return true
		}
	}

	return false
}

func (b *BucketStatus) IsOwner(publicKey []byte) bool {
	address := types.NewAddressFromAccountID(publicKey)

	return b.Bucket.OwnerId == address.AsAccountID
}
