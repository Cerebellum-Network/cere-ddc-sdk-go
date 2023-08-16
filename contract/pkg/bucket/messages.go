package bucket

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type (
	Balance       = types.U128
	Cash          = Balance
	Resource      = uint32
	NodeId        = uint32
	Token         = uint64
	ClusterId     = uint32
	AccountId     = types.AccountID
	ProviderId    = AccountId
	BucketId      = uint32
	Params        = string
	BucketParams  = Params
	CdnNodeParams = Params
	NodeState     = byte
	NodeKey       = uint64
	VNodes        = [][]Token
)

const (
	UNKNOWN NodeState = iota
	ADDING
	ACTIVE
	DELETING
	OFFLINE
)

var NodeTags = map[string]byte{
	"UNKNOWN":  UNKNOWN,
	"ADDING":   ADDING,
	"ACTIVE":   ACTIVE,
	"DELETING": DELETING,
	"OFFLINE":  OFFLINE,
}

type Cluster struct {
	ManagerId        AccountId
	NodesKeys        []string
	ResourcePerVNode Resource
	ResourceUsed     Resource
	Revenues         Cash
	Nodes            []NodeKey
	VNodes           [][]Token
	TotalRent        Balance
	CdnNodesKeys     []string
	CdnUsdPerGb      Balance
	CdnRevenues      Cash
}

type NewCluster struct {
	Params           Params
	ResourcePerVNode Resource
}

type ClusterStatus struct {
	ClusterId ClusterId
	Cluster   Cluster
	Params    Params
}

type CDNCluster struct {
	ManagerId    AccountId
	CDNNodes     []NodeKey
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
	Key    string
	Node   Node
	Params string
}

type CDNNode struct {
	ProviderId           ProviderId
	UndistributedPayment Balance
}

type CDNNodeStatus struct {
	Key    string
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

type BucketCreatedEvent struct {
	BucketId  BucketId
	AccountId AccountId
}

type BucketAllocatedEvent struct {
	BucketId  BucketId
	ClusterId ClusterId
	Resource  Resource
}

type BucketSettlePaymentEvent struct {
	BucketId  BucketId
	ClusterId ClusterId
}

type BucketAvailabilityUpdatedEvent struct {
	BucketId           BucketId
	PublicAvailability bool
}

type BucketParamsSetEvent struct {
	BucketId     BucketId
	BucketParams BucketParams
}

type ClusterCreatedEvent struct {
	ClusterId     ClusterId
	AccountId     AccountId
	ClusterParams Params
}

type ClusterParamsSetEvent struct {
	ClusterId     ClusterId
	ClusterParams ClusterParams
}

type ClusterRemovedEvent struct {
	ClusterId ClusterId
}

type ClusterNodeStatusSetEvent struct {
	ClusterId  ClusterId
	NodeKey    NodeKey
	NodeStatus NodeStatus
}

type ClusterNodeAddedEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
	VNodes    VNodes
}

type ClusterNodeRemovedEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
}

type ClusterCdnNodeAddedEvent struct {
	ClusterId  ClusterId
	CdnNodeKey AccountId
}

type ClusterCdnNodeRemovedEvent struct {
	ClusterId  ClusterId
	CdnNodeKey AccountId
}

type CdnNodeRemovedEvent struct {
	CdnNodeKey AccountId
}

type NodeRemovedEvent struct {
	NodeKey NodeKey
}

type ClusterNodeResetEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
	VNodes    VNodes
}

type ClusterCdnNodeStatusSetEvent struct {
	CdnNodeKey AccountId
	ClusterId  ClusterId
	NodeStatus NodeState
}

type ClusterNodeReplacedEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
	VNodes    VNodes
}

type ClusterReserveResourceEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
}

type ClusterDistributeRevenuesEvent struct {
	ClusterId ClusterId
	AccountId AccountId
}

// type CdnClusterCreatedEvent struct {
// 	ClusterId ClusterId
// 	AccountId AccountId
// }

// type CdnClusterDistributeRevenuesEvent struct {
// 	ClusterId  ClusterId
// 	ProviderId AccountId
// }

type CdnNodeCreatedEvent struct {
	NodeKey   NodeKey
	AccountId AccountId
	Payment   Balance
}

type NodeCreatedEvent struct {
	NodeKey      NodeKey
	ProviderId   AccountId
	RentPerMonth Balance
	NodeParams   Params
}

type DepositEvent struct {
	AccountId AccountId
	Value     Balance
}

type GrantPermissionEvent struct {
	AccountId  AccountId
	Permission byte
}

type RevokePermissionEvent struct {
	AccountId  AccountId
	Permission byte
}

type CdnNodeOwnershipTransferredEvent struct {
	AccountId  AccountId
	CdnNodeKey AccountId
}

type NodeOwnershipTransferredEvent struct {
	AccountId AccountId
	NodeKey   NodeKey
}

type PermissionRevokedEvent struct {
	AccountId  AccountId
	Permission byte
}

type PermissionGrantedEvent struct {
	AccountId  AccountId
	Permission byte
}

type CdnNodeParamsSetEvent struct {
	CdnNodeKey    AccountId
	CdnNodeParams CdnNodeParams
}

type NodeParamsSetEvent struct {
	NodeKey    NodeKey
	NodeParams Params
}

type ClusterDistributeCdnRevenuesEvent struct {
	ClusterId  ClusterId
	ProviderId AccountId
}

func (a *Account) HasBalance() bool {
	return a.Bonded.Cmp(big.NewInt(0)) > 0
}

type ClusterParams struct {
	ReplicationFactor FlexInt `json:"replicationFactor"`
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
