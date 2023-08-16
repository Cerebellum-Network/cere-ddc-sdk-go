package bucket

import (
	_ "embed"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	log "github.com/sirupsen/logrus"
)

const (
	nodeCreateMethod                     = "e8aa4ade"
	nodeRemoveMethod                     = "e068fb34"
	nodeSetParamsMethod                  = "df8b696e"
	nodeGetMethod                        = "847f3997"
	nodeListMethod                       = "423286d6"
	cdnNodeCreateMethod                  = "e8aa4ade"
	cdnNodeRemoveMethod                  = "e068fb34"
	cdnNodeSetParamsMethod               = "df8b696e"
	cdnNodeGetMethod                     = "f9a5a813"
	cdnNodeListMethod                    = "f8589aae"
	clusterCreateMethod                  = "4c0f21f6"
	clusterAddNodeMethod                 = "f7496bdc"
	clusterRemoveNodeMethod              = "793e0778"
	clusterResetNodeMethod               = "a78b2e19"
	clusterReplaceNodeMethod             = "48194ab1"
	clusterAddCdnNodeMethod              = "0b4199f3"
	clusterRemoveCdnNodeMethod           = "ff8531d8"
	clusterSetParamsMethod               = "7dac5f9a"
	clusterRemoveMethod                  = "2248742a"
	clusterSetNodeStatusMethod           = "8078df7f"
	clusterSetCdnNodeStatusMethod        = "577027ba"
	clusterGetMethod                     = "e75411f5"
	clusterListMethod                    = "d9db9d44"
	hasPermissionMethod                  = "e0942492"
	grantTrustedManagerPermissionMethod  = "ea0cbdcd"
	revokeTrustedManagerPermissionMethod = "83532355"
	adminGrantPermissionMethod           = "be41ea55"
	adminRevokePermissionMethod          = "6b150666"
	adminTransferNodeOwnershipMethod     = "783b382d"
	adminTransferCdnNodeOwnershipMethod  = "cd9821be"
	bucketGetMethod                      = "3802cb77"
	accountGetMethod                     = "1d4220fa"

	BucketCreatedEventId                = "004464634275636b65743a3a4275636b65744372656174656400000000000000"
	BucketAllocatedEventId              = "004464634275636b65743a3a4275636b6574416c6c6f63617465640000000000"
	BucketSettlePaymentEventId          = "004464634275636b65743a3a4275636b6574536574746c655061796d656e7400"
	BucketAvailabilityUpdatedId         = "8d8714b3df602b0ce92b8a3de12daedf222ff9198078f834d57176ca2a06359c"
	BucketParamsSetEventId              = "004464634275636b65743a3a4275636b6574506172616d735365740000000000"
	ClusterCreatedEventId               = "004464634275636b65743a3a436c757374657243726561746564000000000000"
	ClusterNodeAddedEventId             = "004464634275636b65743a3a436c75737465724e6f6465416464656400000000"
	ClusterNodeRemovedEventId           = "004464634275636b65743a3a436c75737465724e6f646552656d6f7665640000"
	ClusterCdnNodeAddedEventId          = "004464634275636b65743a3a436c757374657243646e4e6f6465416464656400"
	ClusterCdnNodeRemovedEventId        = "e8920de02c833de0d4c7a1cc213877437ddcc0e1f03f65dd88c7a79c91cde9d9"
	ClusterParamsSetEventId             = "004464634275636b65743a3a436c7573746572506172616d7353657400000000"
	ClusterRemovedEventId               = "004464634275636b65743a3a436c757374657252656d6f766564000000000000"
	ClusterNodeStatusSetEventId         = "004464634275636b65743a3a436c75737465724e6f6465537461747573536574"
	ClusterCdnNodeStatusSetEventId      = "b3c6265529c37cd82b1e4875fa439024770d825e335f643195801131645f3d26"
	ClusterNodeReplacedEventId          = "004464634275636b65743a3a436c75737465724e6f64655265706c6163656400"
	ClusterNodeResetEventId             = "004464634275636b65743a3a436c75737465724e6f6465526573657400000000"
	ClusterReserveResourceEventId       = "84d6d26a3275dced8e359779bf21488762a1d88029f52522d8fc27607759399e"
	ClusterDistributeRevenuesEventId    = "65441936759a16fb28d0e059b878f2e48283ca2eac57c396a8035cce9e2acdd3"
	ClusterDistributeCdnRevenuesEventId = "ec0e9cad0816c5f7e9c252a83e85ca177e162dcb4a284bf80b342942f3e9faa5"
	CdnNodeCreatedEventId               = "004464634275636b65743a3a43646e4e6f646543726561746564000000000000"
	CdnNodeRemovedEventId               = "004464634275636b65743a3a43646e4e6f646552656d6f766564000000000000"
	CdnNodeParamsSetEventId             = "004464634275636b65743a3a43646e4e6f6465506172616d7353657400000000"
	DepositEventId                      = "004464634275636b65743a3a4465706f73697400000000000000000000000000"
	NodeRemovedEventId                  = "004464634275636b65743a3a4e6f646552656d6f766564000000000000000000"
	NodeParamsSetEventId                = "004464634275636b65743a3a4e6f6465506172616d7353657400000000000000"
	NodeCreatedEventId                  = "004464634275636b65743a3a4e6f646543726561746564000000000000000000"
	GrantPermissionEventId              = "004464634275636b65743a3a5065726d697373696f6e4772616e746564000000"
	RevokePermissionEventId             = "004464634275636b65743a3a5065726d697373696f6e5265766f6b6564000000"
	NodeOwnershipTransferredEventId     = "f8da30f579016091acfaa384eee0ddbfcb94d408abc09fde35338ea47c83a0a2"
	CdnNodeOwnershipTransferredEventId  = "ad2b04ceaba2414e23695e96e4bd645d7616ba94cc155679497ef730c086b224"
)

type (
	DdcBucketContract interface {
		GetContractAddress() string
		GetLastAccessTime() time.Time

		BucketGet(bucketId uint32) (*BucketStatus, error)
		ClusterGet(clusterId uint32) (*ClusterStatus, error)
		ClusterCreate(cluster *NewCluster) (clusterId uint32, err error)
		ClusterAddNode(clusterId uint32, nodeKey string, vNodes [][]Token) error
		ClusterRemoveNode(clusterId uint32, nodeKey string) error
		ClusterResetNode(clusterId uint32, nodeKey string, vNodes [][]Token) error
		ClusterReplaceNode(clusterId uint32, vNodes [][]Token, newNodeKey string) error
		ClusterAddCdnNode(clusterId uint32, cdnNodeKey string) error
		ClusterRemoveCdnNode(clusterId uint32, cdnNodeKey string) error
		ClusterSetParams(clusterId uint32, params Params) error
		ClusterRemove(clusterId uint32) error
		ClusterSetNodeStatus(clusterId uint32, nodeKey string, statusInCluster string) error
		ClusterSetCdnNodeStatus(clusterId uint32, cdnNodeKey string, statusInCluster string) error
		ClusterList(offset uint32, limit uint32, filterManagerId string) []*ClusterStatus
		NodeGet(nodeKey string) (*NodeStatus, error)
		NodeCreate(nodeKey string, params Params, capacity Resource) (key string, err error)
		NodeRemove(nodeKey string) error
		NodeSetParams(nodeKey string, params Params) error
		NodeList(offset uint32, limit uint32, filterManagerId string) ([]*NodeStatus, error)
		CDNNodeGet(nodeKey string) (*CDNNodeStatus, error)
		CDNNodeCreate(nodeKey string, params CDNNodeParams) error
		CDNNodeRemove(nodeKey string) error
		CDNNodeSetParams(nodeKey string, params CDNNodeParams) error
		CDNNodeList(offset uint32, limit uint32, filterManagerId string) ([]*CDNNodeStatus, error)
		AccountGet(account types.AccountID) (*Account, error)
		HasPermission(account types.AccountID, permission string) (bool, error)
		GrantTrustedManagerPermission(managerId types.AccountID) error
		RevokeTrustedManagerPermission(managerId types.AccountID) error
		AdminGrantPermission(grantee types.AccountID, permission string) error
		AdminRevokePermission(grantee types.AccountID, permission string) error
		AdminTransferNodeOwnership(nodeKey string, newOwner types.AccountID) error
		AdminTransferCdnNodeOwnership(cdnNodeKey string, newOwner types.AccountID) error
		AddContractEventHandler(event string, handler func(interface{})) error
		GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry
	}

	ddcBucketContract struct {
		contract                               pkg.BlockchainClient
		lastAccessTime                         time.Time
		contractAddressSS58                    string
		keyringPair                            signature.KeyringPair
		nodeCreateMethodId                     []byte
		nodeRemoveMethodId                     []byte
		nodeSetParamsMethodId                  []byte
		nodeGetMethodId                        []byte
		nodeListMethodId                       []byte
		cdnNodeCreateMethodId                  []byte
		cdnNodeRemoveMethodId                  []byte
		cdnNodeSetParamsMethodId               []byte
		cdnNodeGetMethodId                     []byte
		cdnNodeListMethodId                    []byte
		clusterCreateMethodId                  []byte
		clusterAddNodeMethodId                 []byte
		clusterRemoveNodeMethodId              []byte
		clusterResetNodeMethodId               []byte
		clusterReplaceNodeMethodId             []byte
		clusterAddCdnNodeMethodId              []byte
		clusterRemoveCdnNodeMethodId           []byte
		clusterSetParamsMethodId               []byte
		clusterRemoveMethodId                  []byte
		clusterSetNodeStatusMethodId           []byte
		clusterSetCdnNodeStatusMethodId        []byte
		clusterGetMethodId                     []byte
		clusterListMethodId                    []byte
		hasPermissionMethodId                  []byte
		grantTrustedManagerPermissionMethodId  []byte
		revokeTrustedManagerPermissionMethodId []byte
		adminGrantPermissionMethodId           []byte
		adminRevokePermissionMethodId          []byte
		adminTransferNodeOwnershipMethodId     []byte
		adminTransferCdnNodeOwnershipMethodId  []byte
		bucketGetMethodId                      []byte
		accountGetMethodId                     []byte
		eventDispatcher                        map[types.Hash]pkg.ContractEventDispatchEntry
	}
)

var eventDispatchTable = map[string]reflect.Type{
	BucketCreatedEventId:                reflect.TypeOf(BucketCreatedEvent{}),
	BucketAllocatedEventId:              reflect.TypeOf(BucketAllocatedEvent{}),
	BucketSettlePaymentEventId:          reflect.TypeOf(BucketSettlePaymentEvent{}),
	BucketAvailabilityUpdatedId:         reflect.TypeOf(BucketAvailabilityUpdatedEvent{}),
	ClusterCreatedEventId:               reflect.TypeOf(ClusterCreatedEvent{}),
	ClusterNodeReplacedEventId:          reflect.TypeOf(ClusterNodeReplacedEvent{}),
	ClusterReserveResourceEventId:       reflect.TypeOf(ClusterReserveResourceEvent{}),
	ClusterDistributeRevenuesEventId:    reflect.TypeOf(ClusterDistributeRevenuesEvent{}),
	CdnNodeCreatedEventId:               reflect.TypeOf(CdnNodeCreatedEvent{}),
	NodeCreatedEventId:                  reflect.TypeOf(NodeCreatedEvent{}),
	DepositEventId:                      reflect.TypeOf(DepositEvent{}),
	GrantPermissionEventId:              reflect.TypeOf(GrantPermissionEvent{}),
	RevokePermissionEventId:             reflect.TypeOf(RevokePermissionEvent{}),
	BucketParamsSetEventId:              reflect.TypeOf(BucketParamsSetEvent{}),
	ClusterNodeAddedEventId:             reflect.TypeOf(ClusterNodeAddedEvent{}),
	ClusterNodeRemovedEventId:           reflect.TypeOf(ClusterNodeRemovedEvent{}),
	ClusterCdnNodeAddedEventId:          reflect.TypeOf(ClusterCdnNodeAddedEvent{}),
	ClusterCdnNodeRemovedEventId:        reflect.TypeOf(ClusterCdnNodeRemovedEvent{}),
	ClusterParamsSetEventId:             reflect.TypeOf(ClusterParamsSetEvent{}),
	ClusterRemovedEventId:               reflect.TypeOf(ClusterRemovedEvent{}),
	ClusterNodeStatusSetEventId:         reflect.TypeOf(ClusterNodeStatusSetEvent{}),
	ClusterCdnNodeStatusSetEventId:      reflect.TypeOf(ClusterCdnNodeStatusSetEvent{}),
	ClusterNodeResetEventId:             reflect.TypeOf(ClusterNodeResetEvent{}),
	ClusterDistributeCdnRevenuesEventId: reflect.TypeOf(ClusterDistributeCdnRevenuesEvent{}),
	CdnNodeRemovedEventId:               reflect.TypeOf(CdnNodeRemovedEvent{}),
	CdnNodeParamsSetEventId:             reflect.TypeOf(CdnNodeParamsSetEvent{}),
	NodeRemovedEventId:                  reflect.TypeOf(NodeRemovedEvent{}),
	NodeParamsSetEventId:                reflect.TypeOf(NodeParamsSetEvent{}),
	NodeOwnershipTransferredEventId:     reflect.TypeOf(NodeOwnershipTransferredEvent{}),
	CdnNodeOwnershipTransferredEventId:  reflect.TypeOf(CdnNodeOwnershipTransferredEvent{}),
}

func CreateDdcBucketContract(client pkg.BlockchainClient, contractAddressSS58 string) DdcBucketContract {
	bucketGetMethodId, err := hex.DecodeString(bucketGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketGetMethod).Fatal("Can't decode method bucketGetMethod")
	}

	clusterGetMethodId, err := hex.DecodeString(clusterGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterGetMethod).Fatal("Can't decode method clusterGetMethod")
	}

	nodeGetMethodId, err := hex.DecodeString(nodeGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", nodeGetMethod).Fatal("Can't decode method nodeGetMethod")
	}

	cdnNodeGetMethodId, err := hex.DecodeString(cdnNodeGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnNodeGetMethod).Fatal("Can't decode method cdnNodeGetMethod")
	}

	accountGetMethodId, err := hex.DecodeString(accountGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountGetMethod).Fatal("Can't decode method accountGetMethod")
	}

	nodeCreateMethodId, err := hex.DecodeString(nodeCreateMethod)
	if err != nil {
		log.WithError(err).WithField("method", nodeCreateMethod).Fatal("Can't decode method nodeCreateMethod")
	}

	nodeRemoveMethodId, err := hex.DecodeString(nodeRemoveMethod)
	if err != nil {
		log.WithError(err).WithField("method", nodeRemoveMethod).Fatal("Can't decode method nodeRemoveMethod")
	}

	nodeSetParamsMethodId, err := hex.DecodeString(nodeSetParamsMethod)
	if err != nil {
		log.WithError(err).WithField("method", nodeSetParamsMethod).Fatal("Can't decode method nodeSetParamsMethod")
	}

	nodeListMethodId, err := hex.DecodeString(nodeListMethod)
	if err != nil {
		log.WithError(err).WithField("method", nodeListMethod).Fatal("Can't decode method nodeListMethod")
	}

	cdnNodeCreateMethodId, err := hex.DecodeString(cdnNodeCreateMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnNodeCreateMethod).Fatal("Can't decode method cdnNodeCreateMethod")
	}

	cdnNodeRemoveMethodId, err := hex.DecodeString(cdnNodeRemoveMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnNodeRemoveMethod).Fatal("Can't decode method cdnNodeRemoveMethod")
	}

	cdnNodeSetParamsMethodId, err := hex.DecodeString(cdnNodeSetParamsMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnNodeSetParamsMethod).Fatal("Can't decode method cdnNodeSetParamsMethod")
	}

	cdnNodeListMethodId, err := hex.DecodeString(cdnNodeListMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnNodeListMethod).Fatal("Can't decode method cdnNodeListMethod")
	}

	clusterCreateMethodId, err := hex.DecodeString(clusterCreateMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterCreateMethod).Fatal("Can't decode method clusterCreateMethod")
	}

	clusterAddNodeMethodId, err := hex.DecodeString(clusterAddNodeMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterAddNodeMethod).Fatal("Can't decode method clusterAddNodeMethod")
	}

	clusterRemoveNodeMethodId, err := hex.DecodeString(clusterRemoveNodeMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterRemoveNodeMethod).Fatal("Can't decode method clusterRemoveNodeMethod")
	}

	clusterResetNodeMethodId, err := hex.DecodeString(clusterResetNodeMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterResetNodeMethod).Fatal("Can't decode method clusterResetNodeMethod")
	}

	clusterReplaceNodeMethodId, err := hex.DecodeString(clusterReplaceNodeMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterReplaceNodeMethod).Fatal("Can't decode method clusterReplaceNodeMethod")
	}

	clusterAddCdnNodeMethodId, err := hex.DecodeString(clusterAddCdnNodeMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterAddCdnNodeMethod).Fatal("Can't decode method clusterAddCdnNodeMethod")
	}

	clusterRemoveCdnNodeMethodId, err := hex.DecodeString(clusterRemoveCdnNodeMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterRemoveCdnNodeMethod).Fatal("Can't decode method clusterRemoveCdnNodeMethod")
	}

	clusterSetParamsMethodId, err := hex.DecodeString(clusterSetParamsMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterSetParamsMethod).Fatal("Can't decode method clusterSetParamsMethod")
	}

	clusterRemoveMethodId, err := hex.DecodeString(clusterRemoveMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterRemoveMethod).Fatal("Can't decode method clusterRemoveMethod")
	}

	clusterSetNodeStatusMethodId, err := hex.DecodeString(clusterSetNodeStatusMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterSetNodeStatusMethod).Fatal("Can't decode method clusterSetNodeStatusMethod")
	}

	clusterSetCdnNodeStatusMethodId, err := hex.DecodeString(clusterSetCdnNodeStatusMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterSetCdnNodeStatusMethod).Fatal("Can't decode method clusterSetCdnNodeStatusMethod")
	}

	clusterListMethodId, err := hex.DecodeString(clusterListMethod)
	if err != nil {
		log.WithError(err).WithField("method", clusterListMethod).Fatal("Can't decode method clusterListMethod")
	}

	hasPermissionMethodId, err := hex.DecodeString(hasPermissionMethod)
	if err != nil {
		log.WithError(err).WithField("method", hasPermissionMethod).Fatal("Can't decode method hasPermissionMethod")
	}

	grantTrustedManagerPermissionMethodId, err := hex.DecodeString(grantTrustedManagerPermissionMethod)
	if err != nil {
		log.WithError(err).WithField("method", grantTrustedManagerPermissionMethod).Fatal("Can't decode method grantTrustedManagerPermissionMethod")
	}

	revokeTrustedManagerPermissionMethodId, err := hex.DecodeString(revokeTrustedManagerPermissionMethod)
	if err != nil {
		log.WithError(err).WithField("method", revokeTrustedManagerPermissionMethod).Fatal("Can't decode method revokeTrustedManagerPermissionMethod")
	}

	adminGrantPermissionMethodId, err := hex.DecodeString(adminGrantPermissionMethod)
	if err != nil {
		log.WithError(err).WithField("method", adminGrantPermissionMethod).Fatal("Can't decode method adminGrantPermissionMethod")
	}

	adminRevokePermissionMethodId, err := hex.DecodeString(adminRevokePermissionMethod)
	if err != nil {
		log.WithError(err).WithField("method", adminRevokePermissionMethod).Fatal("Can't decode method adminRevokePermissionMethod")
	}

	adminTransferNodeOwnershipMethodId, err := hex.DecodeString(adminTransferNodeOwnershipMethod)
	if err != nil {
		log.WithError(err).WithField("method", adminTransferNodeOwnershipMethod).Fatal("Can't decode method adminTransferNodeOwnershipMethod")
	}

	adminTransferCdnNodeOwnershipMethodId, err := hex.DecodeString(adminTransferCdnNodeOwnershipMethod)
	if err != nil {
		log.WithError(err).WithField("method", adminTransferCdnNodeOwnershipMethod).Fatal("Can't decode method adminTransferCdnNodeOwnershipMethodId")
	}

	eventDispatcher := make(map[types.Hash]pkg.ContractEventDispatchEntry)
	for k, v := range eventDispatchTable {
		if key, err := types.NewHashFromHexString(k); err != nil {
			log.WithError(err).WithField("hash", k).Fatalf("Bad event hash for event %s", v.Name())
		} else {
			eventDispatcher[key] = pkg.ContractEventDispatchEntry{ArgumentType: v}
		}
	}

	return &ddcBucketContract{
		contract:                               client,
		contractAddressSS58:                    contractAddressSS58,
		keyringPair:                            signature.KeyringPair{Address: contractAddressSS58},
		bucketGetMethodId:                      bucketGetMethodId,
		clusterGetMethodId:                     clusterGetMethodId,
		nodeGetMethodId:                        nodeGetMethodId,
		cdnNodeGetMethodId:                     cdnNodeGetMethodId,
		accountGetMethodId:                     accountGetMethodId,
		nodeCreateMethodId:                     nodeCreateMethodId,
		nodeRemoveMethodId:                     nodeRemoveMethodId,
		nodeSetParamsMethodId:                  nodeSetParamsMethodId,
		nodeListMethodId:                       nodeListMethodId,
		cdnNodeCreateMethodId:                  cdnNodeCreateMethodId,
		cdnNodeRemoveMethodId:                  cdnNodeRemoveMethodId,
		cdnNodeSetParamsMethodId:               cdnNodeSetParamsMethodId,
		cdnNodeListMethodId:                    cdnNodeListMethodId,
		clusterCreateMethodId:                  clusterCreateMethodId,
		clusterAddNodeMethodId:                 clusterAddNodeMethodId,
		clusterRemoveNodeMethodId:              clusterRemoveNodeMethodId,
		clusterResetNodeMethodId:               clusterResetNodeMethodId,
		clusterReplaceNodeMethodId:             clusterReplaceNodeMethodId,
		clusterAddCdnNodeMethodId:              clusterAddCdnNodeMethodId,
		clusterRemoveCdnNodeMethodId:           clusterRemoveCdnNodeMethodId,
		clusterSetParamsMethodId:               clusterSetParamsMethodId,
		clusterRemoveMethodId:                  clusterRemoveMethodId,
		clusterSetNodeStatusMethodId:           clusterSetNodeStatusMethodId,
		clusterSetCdnNodeStatusMethodId:        clusterSetCdnNodeStatusMethodId,
		clusterListMethodId:                    clusterListMethodId,
		hasPermissionMethodId:                  hasPermissionMethodId,
		grantTrustedManagerPermissionMethodId:  grantTrustedManagerPermissionMethodId,
		revokeTrustedManagerPermissionMethodId: revokeTrustedManagerPermissionMethodId,
		adminGrantPermissionMethodId:           adminGrantPermissionMethodId,
		adminRevokePermissionMethodId:          adminRevokePermissionMethodId,
		adminTransferNodeOwnershipMethodId:     adminTransferNodeOwnershipMethodId,
		adminTransferCdnNodeOwnershipMethodId:  adminTransferCdnNodeOwnershipMethodId,
		eventDispatcher:                        eventDispatcher,
	}
}

func (d *ddcBucketContract) BucketGet(bucketId uint32) (*BucketStatus, error) {
	res := &BucketStatus{}
	err := d.callToRead(res, d.bucketGetMethodId, types.U32(bucketId))

	return res, err
}

func (d *ddcBucketContract) ClusterGet(clusterId uint32) (*ClusterStatus, error) {
	res := &ClusterStatus{}
	err := d.callToRead(res, d.clusterGetMethodId, types.U32(clusterId))

	return res, err
}

func (d *ddcBucketContract) NodeGet(nodeKey string) (*NodeStatus, error) {
	res := &NodeStatus{}
	err := d.callToRead(res, d.nodeGetMethodId, nodeKey)

	return res, err
}

func (d *ddcBucketContract) CDNNodeGet(nodeKey string) (*CDNNodeStatus, error) {
	res := &CDNNodeStatus{}
	err := d.callToRead(res, d.cdnNodeGetMethodId, nodeKey)

	return res, err
}

func (d *ddcBucketContract) AccountGet(account types.AccountID) (*Account, error) {
	res := &Account{}
	if err := d.callToRead(res, d.accountGetMethodId, account); err != nil {
		return nil, err
	}

	return res, nil
}

func (d *ddcBucketContract) callToRead(result interface{}, method []byte, args ...interface{}) error {
	data, err := d.contract.CallToReadEncoded(d.contractAddressSS58, d.contractAddressSS58, method, args...)
	if err != nil {
		return err
	}

	d.lastAccessTime = time.Now()

	res := Result{data: result}
	if err = res.decodeDdcBucketContract(data); err != nil {
		return err
	}

	return res.err
}

func (d *ddcBucketContract) AddContractEventHandler(event string, handler func(interface{})) error {
	key, err := types.NewHashFromHexString(event)
	if err != nil {
		return err
	}
	entry, found := d.eventDispatcher[key]
	if !found {
		return errors.New("Event not found")
	}
	if entry.Handler != nil {
		return errors.New("Contract event handler already set for " + event)
	}
	entry.Handler = handler
	d.eventDispatcher[key] = entry
	return nil
}

func (d *ddcBucketContract) GetContractAddress() string {
	return d.contractAddressSS58
}

func (d *ddcBucketContract) GetLastAccessTime() time.Time {
	return d.lastAccessTime
}

func (d *ddcBucketContract) GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry {
	return d.eventDispatcher
}

func (d *ddcBucketContract) ClusterCreate(cluster *NewCluster) (clusterId uint32, err error) {
	err = d.callToRead(clusterId, d.clusterCreateMethodId, cluster)
	return clusterId, err
}

func (d *ddcBucketContract) ClusterAddNode(clusterId uint32, nodeKey string, vNodes [][]Token) error {
	err := d.callToRead(clusterId, d.clusterAddNodeMethodId, clusterId, nodeKey, vNodes)
	return err
}

func (d *ddcBucketContract) ClusterRemoveNode(clusterId uint32, nodeKey string) error {
	err := d.callToRead(clusterId, d.clusterRemoveNodeMethodId, clusterId, nodeKey)
	return err
}

func (d *ddcBucketContract) ClusterResetNode(clusterId uint32, nodeKey string, vNodes [][]Token) error {
	err := d.callToRead(clusterId, d.clusterResetNodeMethodId, clusterId, nodeKey, vNodes)
	return err
}

func (d *ddcBucketContract) ClusterReplaceNode(clusterId uint32, vNodes [][]Token, newNodeKey string) error {
	err := d.callToRead(clusterId, d.clusterReplaceNodeMethodId, clusterId, vNodes, newNodeKey)
	return err
}

func (d *ddcBucketContract) ClusterAddCdnNode(clusterId uint32, cdnNodeKey string) error {
	err := d.callToRead(clusterId, d.clusterAddCdnNodeMethodId, clusterId, cdnNodeKey)
	return err
}

func (d *ddcBucketContract) ClusterRemoveCdnNode(clusterId uint32, cdnNodeKey string) error {
	err := d.callToRead(clusterId, d.clusterRemoveCdnNodeMethodId, clusterId, cdnNodeKey)
	return err
}

func (d *ddcBucketContract) ClusterSetParams(clusterId uint32, params Params) error {
	err := d.callToRead(clusterId, d.clusterSetParamsMethodId, clusterId, params)
	return err
}

func (d *ddcBucketContract) ClusterRemove(clusterId uint32) error {
	err := d.callToRead(clusterId, d.clusterRemoveMethodId, clusterId)
	return err
}

func (d *ddcBucketContract) ClusterSetNodeStatus(clusterId uint32, nodeKey string, statusInCluster string) error {
	err := d.callToRead(clusterId, d.clusterSetNodeStatusMethodId, clusterId, nodeKey, statusInCluster)
	return err
}

func (d *ddcBucketContract) ClusterSetCdnNodeStatus(clusterId uint32, cdnNodeKey string, statusInCluster string) error {
	err := d.callToRead(clusterId, d.clusterSetCdnNodeStatusMethodId, clusterId, cdnNodeKey, statusInCluster)
	return err
}

func (d *ddcBucketContract) ClusterList(offset uint32, limit uint32, filterManagerId string) (clusters []*ClusterStatus) {
	_ = d.callToRead(clusters, d.clusterListMethodId, offset, limit, filterManagerId)
	return clusters
}

func (d *ddcBucketContract) NodeCreate(nodeKey string, params Params, capacity Resource) (key string, err error) {
	err = d.callToRead(key, d.nodeCreateMethodId, nodeKey, params, capacity)
	return key, err
}

func (d *ddcBucketContract) NodeRemove(nodeKey string) error {
	err := d.callToRead(nodeKey, d.nodeRemoveMethodId, nodeKey)
	return err
}

func (d *ddcBucketContract) NodeSetParams(nodeKey string, params Params) error {
	err := d.callToRead(nodeKey, d.nodeSetParamsMethodId, nodeKey, params)
	return err
}

func (d *ddcBucketContract) NodeList(offset uint32, limit uint32, filterManagerId string) (nodes []*NodeStatus, err error) {
	err = d.callToRead(nodes, d.nodeListMethodId, offset, limit, filterManagerId)
	return nodes, err
}

func (d *ddcBucketContract) CDNNodeCreate(nodeKey string, params CDNNodeParams) error {
	err := d.callToRead(nodeKey, d.cdnNodeCreateMethodId, nodeKey, params)
	return err
}

func (d *ddcBucketContract) CDNNodeRemove(nodeKey string) error {
	err := d.callToRead(nodeKey, d.cdnNodeRemoveMethodId, nodeKey)
	return err
}

func (d *ddcBucketContract) CDNNodeSetParams(nodeKey string, params CDNNodeParams) error {
	err := d.callToRead(nodeKey, d.cdnNodeSetParamsMethodId, nodeKey, params)
	return err
}

func (d *ddcBucketContract) CDNNodeList(offset uint32, limit uint32, filterManagerId string) (nodes []*CDNNodeStatus, err error) {
	err = d.callToRead(nodes, d.cdnNodeListMethodId, offset, limit, filterManagerId)
	return nodes, err
}

func (d *ddcBucketContract) HasPermission(account types.AccountID, permission string) (has bool, err error) {
	err = d.callToRead(has, d.hasPermissionMethodId, account, permission)
	return has, err
}

func (d *ddcBucketContract) GrantTrustedManagerPermission(managerId types.AccountID) error {
	err := d.callToRead(managerId, d.grantTrustedManagerPermissionMethodId, managerId)
	return err
}

func (d *ddcBucketContract) RevokeTrustedManagerPermission(managerId types.AccountID) error {
	err := d.callToRead(managerId, d.revokeTrustedManagerPermissionMethodId, managerId)
	return err
}

func (d *ddcBucketContract) AdminGrantPermission(grantee types.AccountID, permission string) error {
	err := d.callToRead(grantee, d.adminGrantPermissionMethodId, grantee, permission)
	return err
}

func (d *ddcBucketContract) AdminRevokePermission(grantee types.AccountID, permission string) error {
	err := d.callToRead(grantee, d.adminRevokePermissionMethodId, grantee, permission)
	return err
}

func (d *ddcBucketContract) AdminTransferNodeOwnership(nodeKey string, newOwner types.AccountID) error {
	err := d.callToRead(newOwner, d.adminTransferNodeOwnershipMethodId, nodeKey, newOwner)
	return err
}

func (d *ddcBucketContract) AdminTransferCdnNodeOwnership(cdnNodeKey string, newOwner types.AccountID) error {
	err := d.callToRead(newOwner, d.adminTransferCdnNodeOwnershipMethodId, cdnNodeKey, newOwner)
	return err
}
