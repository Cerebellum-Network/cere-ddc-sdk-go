package bucket

import (
	_ "embed"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
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
	accountDepositMethod                 = "c311af62"
	accountBondMethod                    = "e9fad0bf"
	accountUnbondMethod                  = "f7ea2c67"
	accountGetUsdPerCereMethod           = "e4a4652a"
	accountSetUsdPerCereMethod           = "48d45ee8"
	accountWithdrawUnbondedMethod        = "98173716"
	getAccountsMethod                    = "ef03ead7"
	bucketCreateMethod                   = "0aeb2379"
	bucketChangeOwnerMethod              = "c7d0c2cd"
	bucketAllocIntoClusterMethod         = "4c482d19"
	bucketSettlePaymentMethod            = "15974555"
	bucketChangeParamsMethod             = "9f2d075b"
	bucketListMethod                     = "417ab584"
	bucketListForAccountMethod           = "c434cf57"
	bucketSetAvailabilityMethod          = "053eb3ce"
	bucketSetResourceCapMethod           = "85010c6d"
	getBucketWritersMethod               = "499cd4b7"
	getBucketReadersMethod               = "b9a7cc1c"
	bucketSetWriterPermMethod            = "ea2e477a"
	bucketRevokeWriterPermMethod         = "2b3d8dd1"
	bucketSetReaderPermMethod            = "fc0e94ea"
	bucketRevokeReaderPermMethod         = "e9bfed5a"

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

		AccountDeposit() error
		AccountBond(bondAmount Balance) error
		AccountUnbond(bondAmount Cash) error
		AccountGetUsdPerCere() (balance Balance, err error)
		AccountSetUsdPerCere(usdPerCere Balance) error
		AccountWithdrawUnbonded() error
		GetAccounts() ([]AccountId, error)

		BucketGet(bucketId BucketId) (*BucketInfo, error)
		BucketCreate(bucketParams BucketParams, clusterId ClusterId, ownerId types.OptionAccountID) (bucketId BucketId, err error)
		BucketChangeOwner(bucketId BucketId, ownerId AccountId) error
		BucketAllocIntoCluster(bucketId BucketId, resource Resource) error
		BucketSettlePayment(bucketId BucketId) error
		BucketChangeParams(bucketId BucketId, bucketParams BucketParams) error
		BucketList(offset types.U32, limit types.U32, ownerId types.OptionAccountID) (*BucketListInfo, error)
		BucketListForAccount(ownerId AccountId) ([]*Bucket, error)
		BucketSetAvailability(bucketId BucketId, publicAvailability bool) error
		BucketSetResourceCap(bucketId BucketId, newResourceCap Resource) error
		GetBucketWriters(bucketId BucketId) ([]AccountId, error)
		GetBucketReaders(bucketId BucketId) ([]AccountId, error)
		BucketSetWriterPerm(bucketId BucketId, writer AccountId) error
		BucketRevokeWriterPerm(bucketId BucketId, writer AccountId) error
		BucketSetReaderPerm(bucketId BucketId, reader AccountId) error
		BucketRevokeReaderPerm(bucketId BucketId, reader AccountId) error

		ClusterGet(clusterId ClusterId) (*ClusterInfo, error)
		ClusterCreate(cluster *NewCluster) (clusterId ClusterId, err error)
		ClusterAddNode(clusterId ClusterId, nodeKey NodeKey, vNodes [][]Token) error
		ClusterRemoveNode(clusterId ClusterId, nodeKey NodeKey) error
		ClusterResetNode(clusterId ClusterId, nodeKey NodeKey, vNodes [][]Token) error
		ClusterReplaceNode(clusterId ClusterId, vNodes [][]Token, newNodeKey NodeKey) error
		ClusterAddCdnNode(clusterId ClusterId, nodeKey CdnNodeKey) error
		ClusterRemoveCdnNode(clusterId ClusterId, nodeKey CdnNodeKey) error
		ClusterSetParams(clusterId ClusterId, params Params) error
		ClusterRemove(clusterId ClusterId) error
		ClusterSetNodeStatus(clusterId ClusterId, nodeKey NodeKey, statusInCluster string) error
		ClusterSetCdnNodeStatus(clusterId ClusterId, nodeKey CdnNodeKey, statusInCluster string) error
		ClusterList(offset types.U32, limit types.U32, filterManagerId types.OptionAccountID) (*ClusterListInfo, error)

		NodeGet(nodeKey NodeKey) (*NodeInfo, error)
		NodeCreate(nodeKey NodeKey, params Params, capacity Resource, rent Rent) (key NodeKey, err error)
		NodeRemove(nodeKey NodeKey) error
		NodeSetParams(nodeKey NodeKey, params Params) error
		NodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*NodeListInfo, error)
		CdnNodeGet(nodeKey CdnNodeKey) (*CdnNodeInfo, error)
		CdnNodeCreate(nodeKey CdnNodeKey, params CDNNodeParams) error
		CdnNodeRemove(nodeKey CdnNodeKey) error
		CdnNodeSetParams(nodeKey CdnNodeKey, params CDNNodeParams) error
		CdnNodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*CdnNodeListInfo, error)
		AccountGet(account AccountId) (*Account, error)
		HasPermission(account AccountId, permission string) (bool, error)
		GrantTrustedManagerPermission(managerId AccountId) error
		RevokeTrustedManagerPermission(managerId AccountId) error
		AdminGrantPermission(grantee AccountId, permission string) error
		AdminRevokePermission(grantee AccountId, permission string) error
		AdminTransferNodeOwnership(nodeKey NodeKey, newOwner AccountId) error
		AdminTransferCdnNodeOwnership(nodeKey CdnNodeKey, newOwner AccountId) error
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
		accountGetMethodId                     []byte
		accountDepositMethodId                 []byte
		accountBondMethodId                    []byte
		accountUnbondMethodId                  []byte
		accountGetUsdPerCereMethodId           []byte
		accountSetUsdPerCereMethodId           []byte
		accountWithdrawUnbondedMethodId        []byte
		getAccountsMethodId                    []byte
		bucketGetMethodId                      []byte
		bucketCreateMethodId                   []byte
		bucketChangeOwnerMethodId              []byte
		bucketAllocIntoClusterMethodId         []byte
		bucketSettlePaymentMethodId            []byte
		bucketChangeParamsMethodId             []byte
		bucketListMethodId                     []byte
		bucketListForAccountMethodId           []byte
		bucketSetAvailabilityMethodId          []byte
		bucketSetResourceCapMethodId           []byte
		getBucketWritersMethodId               []byte
		getBucketReadersMethodId               []byte
		bucketSetWriterPermMethodId            []byte
		bucketRevokeWriterPermMethodId         []byte
		bucketSetReaderPermMethodId            []byte
		bucketRevokeReaderPermMethodId         []byte

		eventDispatcher map[types.Hash]pkg.ContractEventDispatchEntry
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

	accountDepositMethodId, err := hex.DecodeString(accountDepositMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountDepositMethod).Fatal("Can't decode method accountDepositMethodId")
	}

	accountBondMethodId, err := hex.DecodeString(accountBondMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountBondMethod).Fatal("Can't decode method accountBondMethodId")
	}

	accountUnbondMethodId, err := hex.DecodeString(accountUnbondMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountUnbondMethod).Fatal("Can't decode method accountUnbondMethodId")
	}

	accountGetUsdPerCereMethodId, err := hex.DecodeString(accountGetUsdPerCereMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountGetUsdPerCereMethod).Fatal("Can't decode method accountGetUsdPerCereMethodId")
	}

	accountSetUsdPerCereMethodId, err := hex.DecodeString(accountSetUsdPerCereMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountSetUsdPerCereMethod).Fatal("Can't decode method accountSetUsdPerCereMethodId")
	}

	accountWithdrawUnbondedMethodId, err := hex.DecodeString(accountWithdrawUnbondedMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountWithdrawUnbondedMethod).Fatal("Can't decode method accountWithdrawUnbondedMethodId")
	}

	getAccountsMethodId, err := hex.DecodeString(getAccountsMethod)
	if err != nil {
		log.WithError(err).WithField("method", getAccountsMethod).Fatal("Can't decode method getAccountsMethodId")
	}

	bucketCreateMethodId, err := hex.DecodeString(bucketCreateMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketCreateMethod).Fatal("Can't decode method bucketCreateMethodId")
	}

	bucketChangeOwnerMethodId, err := hex.DecodeString(bucketChangeOwnerMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketChangeOwnerMethod).Fatal("Can't decode method bucketChangeOwnerMethodId")
	}

	bucketAllocIntoClusterMethodId, err := hex.DecodeString(bucketAllocIntoClusterMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketAllocIntoClusterMethod).Fatal("Can't decode method bucketAllocIntoClusterMethodId")
	}

	bucketSettlePaymentMethodId, err := hex.DecodeString(bucketSettlePaymentMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketSettlePaymentMethod).Fatal("Can't decode method bucketSettlePaymentMethodId")
	}

	bucketChangeParamsMethodId, err := hex.DecodeString(bucketChangeParamsMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketChangeParamsMethod).Fatal("Can't decode method bucketChangeParamsMethodId")
	}

	bucketListMethodId, err := hex.DecodeString(bucketListMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketListMethod).Fatal("Can't decode method bucketListMethodId")
	}

	bucketListForAccountMethodId, err := hex.DecodeString(bucketListForAccountMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketListForAccountMethod).Fatal("Can't decode method bucketListForAccountMethodId")
	}

	bucketSetAvailabilityMethodId, err := hex.DecodeString(bucketSetAvailabilityMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketSetAvailabilityMethod).Fatal("Can't decode method bucketSetAvailabilityMethodId")
	}

	bucketSetResourceCapMethodId, err := hex.DecodeString(bucketSetResourceCapMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketSetResourceCapMethod).Fatal("Can't decode method bucketSetResourceCapMethodId")
	}

	getBucketWritersMethodId, err := hex.DecodeString(bucketSetResourceCapMethod)
	if err != nil {
		log.WithError(err).WithField("method", getBucketWritersMethodId).Fatal("Can't decode method getBucketWritersMethodId")
	}

	getBucketReadersMethodId, err := hex.DecodeString(getBucketReadersMethod)
	if err != nil {
		log.WithError(err).WithField("method", getBucketReadersMethod).Fatal("Can't decode method getBucketReadersMethodId")
	}

	bucketSetWriterPermMethodId, err := hex.DecodeString(bucketSetWriterPermMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketSetWriterPermMethod).Fatal("Can't decode method bucketSetWriterPermMethod")
	}

	bucketRevokeWriterPermMethodId, err := hex.DecodeString(bucketRevokeWriterPermMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketRevokeWriterPermMethod).Fatal("Can't decode method bucketRevokeWriterPermMethodId")
	}

	bucketSetReaderPermMethodId, err := hex.DecodeString(bucketSetReaderPermMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketSetReaderPermMethod).Fatal("Can't decode method bucketSetReaderPermMethodId")
	}

	bucketRevokeReaderPermMethodId, err := hex.DecodeString(bucketRevokeReaderPermMethod)
	if err != nil {
		log.WithError(err).WithField("method", bucketRevokeReaderPermMethod).Fatal("Can't decode method bucketRevokeReaderPermMethodId")
	}

	eventDispatcher := make(map[types.Hash]pkg.ContractEventDispatchEntry)
	for k, v := range eventDispatchTable {
		if eventKey, err := types.NewHashFromHexString(k); err != nil {
			log.WithError(err).WithField("hash", k).Fatalf("Bad event hash for event %s", v.Name())
		} else {
			eventDispatcher[eventKey] = pkg.ContractEventDispatchEntry{ArgumentType: v}
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
		accountDepositMethodId:                 accountDepositMethodId,
		accountBondMethodId:                    accountBondMethodId,
		accountUnbondMethodId:                  accountUnbondMethodId,
		accountGetUsdPerCereMethodId:           accountGetUsdPerCereMethodId,
		accountSetUsdPerCereMethodId:           accountSetUsdPerCereMethodId,
		accountWithdrawUnbondedMethodId:        accountWithdrawUnbondedMethodId,
		getAccountsMethodId:                    getAccountsMethodId,
		bucketCreateMethodId:                   bucketCreateMethodId,
		bucketChangeOwnerMethodId:              bucketChangeOwnerMethodId,
		bucketAllocIntoClusterMethodId:         bucketAllocIntoClusterMethodId,
		bucketSettlePaymentMethodId:            bucketSettlePaymentMethodId,
		bucketChangeParamsMethodId:             bucketChangeParamsMethodId,
		bucketListMethodId:                     bucketListMethodId,
		bucketListForAccountMethodId:           bucketListForAccountMethodId,
		bucketSetAvailabilityMethodId:          bucketSetAvailabilityMethodId,
		bucketSetResourceCapMethodId:           bucketSetResourceCapMethodId,
		getBucketWritersMethodId:               getBucketWritersMethodId,
		getBucketReadersMethodId:               getBucketReadersMethodId,
		bucketSetWriterPermMethodId:            bucketSetWriterPermMethodId,
		bucketRevokeWriterPermMethodId:         bucketRevokeWriterPermMethodId,
		bucketSetReaderPermMethodId:            bucketSetReaderPermMethodId,
		bucketRevokeReaderPermMethodId:         bucketRevokeReaderPermMethodId,
	}
}

func (d *ddcBucketContract) BucketGet(bucketId BucketId) (*BucketInfo, error) {
	res := &BucketInfo{}
	err := d.callToRead(res, d.bucketGetMethodId, types.U32(bucketId))

	return res, err
}

func (d *ddcBucketContract) ClusterGet(clusterId ClusterId) (*ClusterInfo, error) {
	res := &ClusterInfo{}
	err := d.callToRead(res, d.clusterGetMethodId, types.U32(clusterId))

	return res, err
}

func (d *ddcBucketContract) NodeGet(nodeKey NodeKey) (*NodeInfo, error) {
	res := &NodeInfo{}
	err := d.callToRead(res, d.nodeGetMethodId, nodeKey)

	return res, err
}

func (d *ddcBucketContract) CdnNodeGet(nodeKey CdnNodeKey) (*CdnNodeInfo, error) {
	res := &CdnNodeInfo{}
	err := d.callToRead(res, d.cdnNodeGetMethodId, nodeKey)

	return res, err
}

func (d *ddcBucketContract) AccountGet(account AccountId) (*Account, error) {
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

func (d *ddcBucketContract) callToReadNoResult(res interface{}, method []byte, args ...interface{}) error {
	data, err := d.contract.CallToReadEncoded(d.contractAddressSS58, d.contractAddressSS58, method, args...)
	if err != nil {
		return err
	}

	d.lastAccessTime = time.Now()

	return codec.DecodeFromHex(data, res)
}

func (d *ddcBucketContract) AddContractEventHandler(event string, handler func(interface{})) error {
	eventKey, err := types.NewHashFromHexString(event)
	if err != nil {
		return err
	}
	entry, found := d.eventDispatcher[eventKey]
	if !found {
		return errors.New("Event not found")
	}
	if entry.Handler != nil {
		return errors.New("Contract event handler already set for " + event)
	}
	entry.Handler = handler
	d.eventDispatcher[eventKey] = entry
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

func (d *ddcBucketContract) ClusterCreate(cluster *NewCluster) (clusterId ClusterId, err error) {
	err = d.callToRead(clusterId, d.clusterCreateMethodId, cluster)
	return clusterId, err
}

func (d *ddcBucketContract) ClusterAddNode(clusterId ClusterId, nodeKey NodeKey, vNodes [][]Token) error {
	err := d.callToRead(clusterId, d.clusterAddNodeMethodId, clusterId, nodeKey, vNodes)
	return err
}

func (d *ddcBucketContract) ClusterRemoveNode(clusterId ClusterId, nodeKey NodeKey) error {
	err := d.callToRead(clusterId, d.clusterRemoveNodeMethodId, clusterId, nodeKey)
	return err
}

func (d *ddcBucketContract) ClusterResetNode(clusterId ClusterId, nodeKey NodeKey, vNodes [][]Token) error {
	err := d.callToRead(clusterId, d.clusterResetNodeMethodId, clusterId, nodeKey, vNodes)
	return err
}

func (d *ddcBucketContract) ClusterReplaceNode(clusterId ClusterId, vNodes [][]Token, newNodeKey NodeKey) error {
	err := d.callToRead(clusterId, d.clusterReplaceNodeMethodId, clusterId, vNodes, newNodeKey)
	return err
}

func (d *ddcBucketContract) ClusterAddCdnNode(clusterId ClusterId, nodeKey CdnNodeKey) error {
	err := d.callToRead(clusterId, d.clusterAddCdnNodeMethodId, clusterId, nodeKey)
	return err
}

func (d *ddcBucketContract) ClusterRemoveCdnNode(clusterId ClusterId, nodeKey CdnNodeKey) error {
	err := d.callToRead(clusterId, d.clusterRemoveCdnNodeMethodId, clusterId, nodeKey)
	return err
}

func (d *ddcBucketContract) ClusterSetParams(clusterId ClusterId, params Params) error {
	err := d.callToRead(clusterId, d.clusterSetParamsMethodId, clusterId, params)
	return err
}

func (d *ddcBucketContract) ClusterRemove(clusterId ClusterId) error {
	err := d.callToRead(clusterId, d.clusterRemoveMethodId, clusterId)
	return err
}

func (d *ddcBucketContract) ClusterSetNodeStatus(clusterId ClusterId, nodeKey NodeKey, statusInCluster string) error {
	err := d.callToRead(clusterId, d.clusterSetNodeStatusMethodId, clusterId, nodeKey, statusInCluster)
	return err
}

func (d *ddcBucketContract) ClusterSetCdnNodeStatus(clusterId ClusterId, nodeKey CdnNodeKey, statusInCluster string) error {
	err := d.callToRead(clusterId, d.clusterSetCdnNodeStatusMethodId, clusterId, nodeKey, statusInCluster)
	return err
}

func (d *ddcBucketContract) ClusterList(offset types.U32, limit types.U32, filterManagerId types.OptionAccountID) (*ClusterListInfo, error) {
	res := ClusterListInfo{}
	err := d.callToReadNoResult(&res, d.clusterListMethodId, offset, limit, filterManagerId)
	return &res, err
}

func (d *ddcBucketContract) NodeCreate(nodeKey NodeKey, params Params, capacity Resource, rent Rent) (key NodeKey, err error) {
	err = d.callToRead(nodeKey, d.nodeCreateMethodId, nodeKey, params, capacity, rent)
	return key, err
}

func (d *ddcBucketContract) NodeRemove(nodeKey NodeKey) error {
	err := d.callToRead(nodeKey, d.nodeRemoveMethodId, nodeKey)
	return err
}

func (d *ddcBucketContract) NodeSetParams(nodeKey NodeKey, params Params) error {
	err := d.callToRead(nodeKey, d.nodeSetParamsMethodId, nodeKey, params)
	return err
}

func (d *ddcBucketContract) NodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*NodeListInfo, error) {
	res := NodeListInfo{}
	err := d.callToReadNoResult(&res, d.nodeListMethodId, offset, limit, filterProviderId)
	return &res, err
}

func (d *ddcBucketContract) CdnNodeCreate(nodeKey CdnNodeKey, params CDNNodeParams) error {
	err := d.callToRead(nodeKey, d.cdnNodeCreateMethodId, nodeKey, params)
	return err
}

func (d *ddcBucketContract) CdnNodeRemove(nodeKey CdnNodeKey) error {
	err := d.callToRead(nodeKey, d.cdnNodeRemoveMethodId, nodeKey)
	return err
}

func (d *ddcBucketContract) CdnNodeSetParams(nodeKey CdnNodeKey, params CDNNodeParams) error {
	err := d.callToRead(nodeKey, d.cdnNodeSetParamsMethodId, nodeKey, params)
	return err
}

func (d *ddcBucketContract) CdnNodeList(offset types.U32, limit types.U32, filterProviderId types.OptionAccountID) (*CdnNodeListInfo, error) {
	res := CdnNodeListInfo{}
	err := d.callToReadNoResult(&res, d.cdnNodeListMethodId, offset, limit, filterProviderId)
	return &res, err
}

func (d *ddcBucketContract) HasPermission(account AccountId, permission string) (has bool, err error) {
	err = d.callToRead(has, d.hasPermissionMethodId, account, permission)
	return has, err
}

func (d *ddcBucketContract) GrantTrustedManagerPermission(managerId AccountId) error {
	err := d.callToRead(managerId, d.grantTrustedManagerPermissionMethodId, managerId)
	return err
}

func (d *ddcBucketContract) RevokeTrustedManagerPermission(managerId AccountId) error {
	err := d.callToRead(managerId, d.revokeTrustedManagerPermissionMethodId, managerId)
	return err
}

func (d *ddcBucketContract) AdminGrantPermission(grantee AccountId, permission string) error {
	err := d.callToRead(grantee, d.adminGrantPermissionMethodId, grantee, permission)
	return err
}

func (d *ddcBucketContract) AdminRevokePermission(grantee AccountId, permission string) error {
	err := d.callToRead(grantee, d.adminRevokePermissionMethodId, grantee, permission)
	return err
}

func (d *ddcBucketContract) AdminTransferNodeOwnership(nodeKey NodeKey, newOwner AccountId) error {
	err := d.callToRead(newOwner, d.adminTransferNodeOwnershipMethodId, nodeKey, newOwner)
	return err
}

func (d *ddcBucketContract) AdminTransferCdnNodeOwnership(nodeKey CdnNodeKey, newOwner AccountId) error {
	err := d.callToRead(newOwner, d.adminTransferCdnNodeOwnershipMethodId, nodeKey, newOwner)
	return err
}

func (d *ddcBucketContract) AccountDeposit() error {
	err := d.callToRead(nil, d.accountDepositMethodId, nil)
	return err
}

func (d *ddcBucketContract) AccountBond(bondAmount Balance) error {
	err := d.callToRead(bondAmount, d.accountBondMethodId, bondAmount)
	return err
}

func (d *ddcBucketContract) AccountUnbond(bondAmount Balance) error {
	err := d.callToRead(bondAmount, d.accountUnbondMethodId)
	return err
}

func (d *ddcBucketContract) AccountGetUsdPerCere() (balance Balance, err error) {
	err = d.callToRead(balance, d.accountGetUsdPerCereMethodId, balance)
	return balance, err
}

func (d *ddcBucketContract) AccountSetUsdPerCere(usdPerCere Balance) error {
	err := d.callToRead(usdPerCere, d.accountSetUsdPerCereMethodId)
	return err
}

func (d *ddcBucketContract) AccountWithdrawUnbonded() error {
	err := d.callToRead(nil, d.accountWithdrawUnbondedMethodId, nil)
	return err
}

func (d *ddcBucketContract) GetAccounts() (accounts []types.AccountID, err error) {
	err = d.callToRead(accounts, d.getAccountsMethodId, accounts)
	return accounts, err
}

func (d *ddcBucketContract) BucketCreate(bucketParams BucketParams, clusterId ClusterId, ownerId types.OptionAccountID) (bucketId BucketId, err error) {
	err = d.callToRead(bucketId, d.bucketCreateMethodId, bucketParams, clusterId, ownerId)
	return bucketId, err
}

func (d *ddcBucketContract) BucketChangeOwner(bucketId BucketId, newOwnerId AccountId) error {
	err := d.callToRead(newOwnerId, d.bucketChangeOwnerMethodId, bucketId, newOwnerId)
	return err
}

func (d *ddcBucketContract) BucketAllocIntoCluster(bucketId types.U32, resource Resource) error {
	err := d.callToRead(bucketId, d.bucketAllocIntoClusterMethodId, resource, bucketId)
	return err
}

func (d *ddcBucketContract) BucketSettlePayment(bucketId types.U32) error {
	err := d.callToRead(bucketId, d.bucketSettlePaymentMethodId, bucketId)
	return err
}

func (d *ddcBucketContract) BucketChangeParams(bucketId types.U32, bucketParams BucketParams) error {
	err := d.callToRead(bucketId, d.bucketChangeParamsMethodId, bucketParams, bucketId)
	return err
}

func (d *ddcBucketContract) BucketList(offset types.U32, limit types.U32, filterOwnerId types.OptionAccountID) (*BucketListInfo, error) {
	res := BucketListInfo{}
	err := d.callToReadNoResult(&res, d.bucketListMethodId, offset, limit, filterOwnerId)
	return &res, err
}

func (d *ddcBucketContract) BucketListForAccount(ownerId AccountId) (buckets []*Bucket, err error) {
	err = d.callToRead(buckets, d.bucketListForAccountMethodId, ownerId)
	return buckets, err
}

func (d *ddcBucketContract) BucketSetAvailability(bucketId types.U32, publicAvailability bool) error {
	err := d.callToRead(bucketId, d.bucketSetAvailabilityMethodId, publicAvailability, bucketId)
	return err
}

func (d *ddcBucketContract) BucketSetResourceCap(bucketId types.U32, newResourceCap Resource) error {
	err := d.callToRead(bucketId, d.bucketSetResourceCapMethodId, newResourceCap, bucketId)
	return err
}

func (d *ddcBucketContract) GetBucketWriters(bucketId types.U32) (writers []types.AccountID, err error) {
	err = d.callToRead(writers, d.getBucketWritersMethodId, bucketId)
	return writers, err
}

func (d *ddcBucketContract) GetBucketReaders(bucketId types.U32) (readers []types.AccountID, err error) {
	err = d.callToRead(readers, d.getBucketReadersMethodId, bucketId)
	return readers, err
}

func (d *ddcBucketContract) BucketSetWriterPerm(bucketId types.U32, writer AccountId) error {
	err := d.callToRead(writer, d.bucketSetWriterPermMethodId, bucketId, writer)
	return err
}

func (d *ddcBucketContract) BucketRevokeWriterPerm(bucketId types.U32, writer AccountId) error {
	err := d.callToRead(writer, d.bucketRevokeWriterPermMethodId, bucketId, writer)
	return err
}

func (d *ddcBucketContract) BucketSetReaderPerm(bucketId types.U32, reader AccountId) error {
	err := d.callToRead(reader, d.bucketSetReaderPermMethodId, bucketId, reader)
	return err
}

func (d *ddcBucketContract) BucketRevokeReaderPerm(bucketId types.U32, reader AccountId) error {
	err := d.callToRead(reader, d.bucketRevokeReaderPermMethodId, bucketId, reader)
	return err
}
