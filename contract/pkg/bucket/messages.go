package bucket

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"math/big"
	"time"
)

type (
	Balance      = types.U128
	Cash         = Balance
	Resource     = uint32
	NodeId       = uint32
	ClusterId    = uint32
	AccountId    = types.AccountID
	ProviderId   = AccountId
	BucketId     = uint32
	Params       = string
	BucketParams = Params
)

type Cluster struct {
	ManagerId        AccountId
	VNodes           []NodeId
	ResourcePerVNode Resource
	ResourceUsed     Resource
	Revenues         Cash
	TotalRent        Balance
}

type ClusterStatus struct {
	ClusterId ClusterId
	Cluster   Cluster
	Params    Params
}

type Node struct {
	ProviderId    ProviderId
	RentPerMonth  Balance
	FreeResources Resource
}

type NodeStatus struct {
	NodeId NodeId
	Node   Node
	Params string
}

type Bucket struct {
	OwnerId            AccountId
	ClusterId          ClusterId
	ResourceReserved   Resource
	PublicAvailability bool
	GasConsumptionCap  Resource
}

type Schedule struct {
	Rate   Balance
	Offset Balance
}

type BucketStatus struct {
	BucketId           BucketId
	Bucket             Bucket
	Params             BucketParams
	WriterIds          []AccountId
	RentCoveredUntilMs uint64
}

type Account struct {
	Deposit           Cash
	Bonded            Cash
	Negative          Cash
	UnboundedAmount   Cash
	UnbondedTimestamp uint64
	PayableSchedule   Schedule
}

func (a *Account) HasBalance() bool {
	return a.Bonded.Cmp(big.NewInt(0)) > 0
}

func (b *BucketStatus) RentExpired() bool {
	return b.RentCoveredUntilMs < uint64(time.Now().UnixMilli())
}

func (b *BucketStatus) HasWriteAccess(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	for _, writerId := range b.WriterIds {
		if writerId == address.AsAccountID {
			return true
		}
	}

	return false
}

func (b *BucketStatus) IsOwner(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.Bucket.OwnerId == address.AsAccountID
}
