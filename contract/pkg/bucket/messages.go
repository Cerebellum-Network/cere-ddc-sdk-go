package bucket

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type (
	Balance      = types.U128
	Cash         = Balance
	Resource     = uint32
	NodeId       = uint32
	Token        = uint64
	ClusterId    = uint32
	AccountId    = types.AccountID
	ProviderId   = AccountId
	BucketId     = uint32
	Params       = string
	BucketParams = Params
	NodeState    = byte
)

const (
	ACTIVE NodeState = iota
	ADDING
	DELETING
	OFFLINE
)

var NodeTags = map[string]byte{
	"ACTIVE":   ACTIVE,
	"ADDING":   ADDING,
	"DELETING": DELETING,
	"OFFLINE":  OFFLINE,
}

type Cluster struct {
	ManagerId        AccountId
	ResourcePerVNode Resource
	ResourceUsed     Resource
	Revenues         Cash
	Nodes            []NodeId
	VNodes           [][]Token
	TotalRent        Balance
}

type ClusterStatus struct {
	ClusterId ClusterId
	Cluster   Cluster
	Params    Params
}

type CDNCluster struct {
	ManagerId    AccountId
	CDNNodes     []NodeId
	ResourceUsed Resource
	Revenues     Cash
	UsdPerGb     Balance
}

type CDNClusterStatus struct {
	ClusterId  ClusterId
	CDNCluster CDNCluster
}

type Node struct {
	ProviderId    ProviderId
	RentPerMonth  Balance
	FreeResources Resource
	NodeState     NodeState
}

type NodeStatus struct {
	NodeId NodeId
	Node   Node
	Params string
}

type CDNNode struct {
	ProviderId           ProviderId
	UndistributedPayment Balance
}

type CDNNodeStatus struct {
	NodeId NodeId
	Node   CDNNode
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
	ReaderIds          []AccountId
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

type BucketCreated struct {
	BucketId  BucketId
	AccountId AccountId
}

type BucketAllocated struct {
	BucketId  BucketId
	ClusterId ClusterId
	Resource  Resource
}

type BucketSettlePayment struct {
	BucketId  BucketId
	ClusterId ClusterId
}

type ClusterCreated struct {
	ClusterId     ClusterId
	AccountId     AccountId
	ClusterParams Params
}

type ClusterNodeReplaced struct {
	ClusterId ClusterId
	NodeId    NodeId
}

type ClusterReserveResource struct {
	ClusterId ClusterId
	NodeId    NodeId
}

type ClusterDistributeRevenues struct {
	ClusterId ClusterId
	AccountId AccountId
}

type CdnClusterCreated struct {
	ClusterId ClusterId
	AccountId AccountId
}

type CdnClusterDistributeRevenues struct {
	ClusterId  ClusterId
	ProviderId AccountId
}

type CdnNodeCreated struct {
	NodeId    NodeId
	AccountId AccountId
	Payment   Balance
}

type NodeCreated struct {
	NodeId       NodeId
	ProviderId   AccountId
	RentPerMonth Balance
	NodeParams   Params
}

type Deposit struct {
	AccountId AccountId
	Value     Balance
}

type GrantPermission struct {
	AccountId  AccountId
	Permission byte
}

type RevokePermission struct {
	AccountId  AccountId
	Permission byte
}

func (a *Account) HasBalance() bool {
	return a.Bonded.Cmp(big.NewInt(0)) > 0
}

type ClusterParams struct {
	ReplicationFactor int `json:"replicationFactor"`
}

func (c *ClusterStatus) ReplicationFactor() uint {
	params := &ClusterParams{}
	err := json.Unmarshal([]byte(c.Params), params)
	if err != nil || params.ReplicationFactor <= 0 {
		return 0
	}

	return uint(params.ReplicationFactor)
}

func (b *BucketStatus) RentExpired() bool {
	return b.RentCoveredUntilMs < uint64(time.Now().UnixMilli())
}

func (b *BucketStatus) HasWriteAccess(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.isOwner(address) || b.isWriter(address)
}

func (b *BucketStatus) HasReadAccess(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.isOwner(address) || b.isWriter(address) || b.isReader(address)
}

func (b *BucketStatus) IsOwner(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.isOwner(address)
}

func (b *BucketStatus) isOwner(address types.Address) bool {
	return b.Bucket.OwnerId == address.AsAccountID
}

func (b *BucketStatus) isWriter(address types.Address) bool {
	for _, writerId := range b.WriterIds {
		if writerId == address.AsAccountID {
			return true
		}
	}

	return false
}

func (b *BucketStatus) isReader(address types.Address) bool {
	for _, readerId := range b.ReaderIds {
		if readerId == address.AsAccountID {
			return true
		}
	}

	return false
}
