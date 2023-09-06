package bucket

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type (
	Balance             = types.U128
	Cash                = Balance
	Resource            = types.U32
	Token               = types.U64
	ClusterId           = types.U32
	AccountId           = types.AccountID
	ProviderId          = AccountId
	BucketId            = types.U32
	Params              = string
	BucketParams        = Params
	NodeParams          = Params
	CdnNodeParams       = Params
	NodeStatusInCluster = uint8
	NodeKey             = AccountId
	CdnNodeKey          = AccountId
	Rent                = Balance
)

const (
	ADDING = iota
	ACTIVE
	DELETING
	OFFLINE
)

const UNKNOWN_NODE_STATUS_IN_CLUSTER byte = 0xFF

var NodeStatusesInClusterMap = map[string]byte{
	"ADDING":   ADDING,
	"ACTIVE":   ACTIVE,
	"DELETING": DELETING,
	"OFFLINE":  OFFLINE,
}

type Cluster struct {
	ManagerId        AccountId
	Params           Params
	NodesKeys        []NodeKey
	ResourcePerVNode Resource
	ResourceUsed     Resource
	Revenues         Cash
	TotalRent        Balance
	CdnNodesKeys     []CdnNodeKey
	CdnRevenues      Cash
	CdnUsdPerGb      Balance
}

type NodeVNodesInfo struct {
	NodeKey NodeKey
	VNodes  []Token
}

type ClusterInfo struct {
	ClusterId   ClusterId
	Cluster     Cluster
	NodesVNodes []NodeVNodesInfo
}

type ClusterListInfo struct {
	Clusters []ClusterInfo
	Total    types.U32
}

type NewCluster struct {
	Params           Params
	ResourcePerVNode Resource
}

type Node struct {
	ProviderId      ProviderId
	RentPerMonth    Balance
	FreeResources   Resource
	Params          NodeParams
	ClusterId       types.OptionU32
	StatusInCluster types.OptionU8
}

type NodeInfo struct {
	Key    NodeKey
	Node   Node
	VNodes []Token
}

type NodeListInfo struct {
	Nodes []NodeInfo
	Total types.U32
}

type CdnNode struct {
	ProviderId           ProviderId
	UndistributedPayment Balance
	Params               CdnNodeParams
	ClusterId            types.OptionU32
	StatusInCluster      types.OptionU8
}

type CdnNodeInfo struct {
	Key  CdnNodeKey
	Node CdnNode
}

type CdnNodeListInfo struct {
	Nodes []CdnNodeInfo
	Total types.U32
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

type BucketInfo struct {
	BucketId           BucketId
	Bucket             Bucket
	Params             BucketParams
	WriterIds          []AccountId
	ReaderIds          []AccountId
	RentCoveredUntilMs types.U64
}

type BucketListInfo struct {
	Buckets []BucketListInfo
	Total   types.U32
}

type Account struct {
	Deposit           Cash
	Bonded            Cash
	Negative          Cash
	UnboundedAmount   Cash
	UnbondedTimestamp types.U64
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
	ClusterId           ClusterId
	NodeKey             NodeKey
	NodeStatusInCluster NodeStatusInCluster
}

type ClusterNodeAddedEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
	VNodes    []Token
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
	VNodes    []Token
}

type ClusterCdnNodeStatusSetEvent struct {
	CdnNodeKey          AccountId
	ClusterId           ClusterId
	NodeStatusInCluster NodeStatusInCluster
}

type ClusterNodeReplacedEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
	VNodes    []Token
}

type ClusterReserveResourceEvent struct {
	ClusterId ClusterId
	NodeKey   NodeKey
}

type ClusterDistributeRevenuesEvent struct {
	ClusterId ClusterId
	AccountId AccountId
}

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

func (c *ClusterInfo) ReplicationFactor() uint {
	params := &ClusterParams{}
	err := json.Unmarshal([]byte(c.Cluster.Params), params)
	if err != nil || params.ReplicationFactor <= 0 {
		return 0
	}

	return uint(params.ReplicationFactor)
}

func (b *BucketInfo) RentExpired() bool {
	return b.RentCoveredUntilMs < types.U64(time.Now().UnixMilli())
}

func (b *BucketInfo) HasWriteAccess(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.isOwner(address) || b.isWriter(address)
}

func (b *BucketInfo) HasReadAccess(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.isOwner(address) || b.isWriter(address) || b.isReader(address)
}

func (b *BucketInfo) IsOwner(publicKey []byte) bool {
	address, err := types.NewAddressFromAccountID(publicKey)
	if err != nil {
		return false
	}

	return b.isOwner(address)
}

func (b *BucketInfo) isOwner(address types.Address) bool {
	return b.Bucket.OwnerId == address.AsAccountID
}

func (b *BucketInfo) isWriter(address types.Address) bool {
	for _, writerId := range b.WriterIds {
		if writerId == address.AsAccountID {
			return true
		}
	}

	return false
}

func (b *BucketInfo) isReader(address types.Address) bool {
	for _, readerId := range b.ReaderIds {
		if readerId == address.AsAccountID {
			return true
		}
	}

	return false
}

func (n *Node) GetStatusInCluster() (NodeStatusInCluster, error) {
	hasValue, val := n.StatusInCluster.Unwrap()
	if !hasValue {
		return UNKNOWN_NODE_STATUS_IN_CLUSTER, fmt.Errorf("unknown storage node status in cluster")
	} else {
		return uint8(val), nil
	}
}

func (n *CdnNode) GetStatusInCluster() (NodeStatusInCluster, error) {
	hasValue, val := n.StatusInCluster.Unwrap()
	if !hasValue {
		return UNKNOWN_NODE_STATUS_IN_CLUSTER, fmt.Errorf("unknown cdn node status in cluster")
	} else {
		return uint8(val), nil
	}
}

func (n *NodeInfo) GetStatusInCluster() (NodeStatusInCluster, error) {
	return n.Node.GetStatusInCluster()
}

func (n *CdnNodeInfo) GetStatusInCluster() (NodeStatusInCluster, error) {
	return n.Node.GetStatusInCluster()
}
