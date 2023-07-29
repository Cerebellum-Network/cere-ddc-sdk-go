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
	bucketGetMethod     = "3802cb77"
	clusterGetMethod    = "e75411f5"
	nodeGetMethod       = "847f3997"
	accountGetMethod    = "1d4220fa"
	cdnClusterGetMethod = "4b22fbf1"
	cdnNodeGetMethod    = "f9a5a813"

	BucketCreatedEventId                = "004464634275636b65743a3a4275636b65744372656174656400000000000000"
	BucketAllocatedEventId              = "004464634275636b65743a3a4275636b6574416c6c6f63617465640000000000"
	BucketSettlePaymentEventId          = "004464634275636b65743a3a4275636b6574536574746c655061796d656e7400"
	BucketAvailabilityUpdatedId         = "8d8714b3df602b0ce92b8a3de12daedf222ff9198078f834d57176ca2a06359c"
	ClusterCreatedEventId               = "004464634275636b65743a3a436c757374657243726561746564000000000000"
	ClusterNodeReplacedEventId          = "004464634275636b65743a3a436c75737465724e6f64655265706c6163656400"
	ClusterReserveResourceEventId       = "84d6d26a3275dced8e359779bf21488762a1d88029f52522d8fc27607759399e"
	ClusterDistributeRevenuesEventId    = "65441936759a16fb28d0e059b878f2e48283ca2eac57c396a8035cce9e2acdd3"
	CdnClusterCreatedEventId            = "004464634275636b65743a3a43646e436c757374657243726561746564000000"
	CdnClusterDistributeRevenuesEventId = "4e19fc4683a9a741a09d89a1d62b22d592a8bf10e2b8b6eff7c6742a0ed88bb4"
	CdnNodeCreatedEventId               = "004464634275636b65743a3a43646e4e6f646543726561746564000000000000"
	NodeCreatedEventId                  = "004464634275636b65743a3a4e6f646543726561746564000000000000000000"
	DepositEventId                      = "004464634275636b65743a3a4465706f73697400000000000000000000000000"
	GrantPermissionEventId              = "004464634275636b65743a3a4772616e745065726d697373696f6e0000000000"
	RevokePermissionEventId             = "004464634275636b65743a3a5265766f6b655065726d697373696f6e00000000"
)

type (
	DdcBucketContract interface {
		GetContractAddress() string
		GetLastAccessTime() time.Time

		BucketGet(bucketId uint32) (*BucketStatus, error)
		ClusterGet(clusterId uint32) (*ClusterStatus, error)
		NodeGet(nodeId uint32) (*NodeStatus, error)
		CDNClusterGet(clusterId uint32) (*CDNClusterStatus, error)
		CDNNodeGet(nodeId uint32) (*CDNNodeStatus, error)
		AccountGet(account types.AccountID) (*Account, error)
		AddContractEventHandler(event string, handler func(interface{})) error
		GetEventDispatcher() map[types.Hash]pkg.ContractEventDispatchEntry
	}

	ddcBucketContract struct {
		contract              pkg.BlockchainClient
		lastAccessTime        time.Time
		contractAddressSS58   string
		keyringPair           signature.KeyringPair
		bucketGetMethodId     []byte
		clusterGetMethodId    []byte
		nodeGetMethodId       []byte
		cdnClusterGetMethodId []byte
		cdnNodeGetMethodId    []byte
		accountGetMethodId    []byte
		eventDispatcher       map[types.Hash]pkg.ContractEventDispatchEntry
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
	CdnClusterCreatedEventId:            reflect.TypeOf(CdnClusterCreatedEvent{}),
	CdnClusterDistributeRevenuesEventId: reflect.TypeOf(CdnClusterDistributeRevenuesEvent{}),
	CdnNodeCreatedEventId:               reflect.TypeOf(CdnNodeCreatedEvent{}),
	NodeCreatedEventId:                  reflect.TypeOf(NodeCreatedEvent{}),
	DepositEventId:                      reflect.TypeOf(DepositEvent{}),
	GrantPermissionEventId:              reflect.TypeOf(GrantPermissionEvent{}),
	RevokePermissionEventId:             reflect.TypeOf(RevokePermissionEvent{})}

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

	cdnClusterGetMethodId, err := hex.DecodeString(cdnClusterGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnClusterGetMethod).Fatal("Can't decode method cdnClusterGetMethod")
	}

	cdnNodeGetMethodId, err := hex.DecodeString(cdnNodeGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", cdnNodeGetMethod).Fatal("Can't decode method cdnNodeGetMethod")
	}

	accountGetMethodId, err := hex.DecodeString(accountGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountGetMethod).Fatal("Can't decode method accountGetMethod")
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
		contract:              client,
		contractAddressSS58:   contractAddressSS58,
		keyringPair:           signature.KeyringPair{Address: contractAddressSS58},
		bucketGetMethodId:     bucketGetMethodId,
		clusterGetMethodId:    clusterGetMethodId,
		nodeGetMethodId:       nodeGetMethodId,
		cdnClusterGetMethodId: cdnClusterGetMethodId,
		cdnNodeGetMethodId:    cdnNodeGetMethodId,
		accountGetMethodId:    accountGetMethodId,
		eventDispatcher:       eventDispatcher,
	}
}

func (d *ddcBucketContract) BucketGet(bucketId uint32) (*BucketStatus, error) {
	res := &BucketStatus{}
	err := d.callToRead(res, d.bucketGetMethodId, types.U32(bucketId))

	return res, err
}

func (d *ddcBucketContract) ClusterGet(clusterId uint32) (*ClusterStatus, error) {
	res := &ClusterStatus{}
	log.Warnf("1. ClusterGet --------> cdnClusterGetMethodId=%x clusterId=%v", d.clusterGetMethodId, clusterId)
	err := d.callToRead(res, d.clusterGetMethodId, types.U32(clusterId))
	log.Warnf("2. ClusterGet after --------> res: %v", res)

	return res, err
}

func (d *ddcBucketContract) NodeGet(nodeId uint32) (*NodeStatus, error) {
	res := &NodeStatus{}
	err := d.callToRead(res, d.nodeGetMethodId, types.U32(nodeId))

	return res, err
}

func (d *ddcBucketContract) CDNClusterGet(clusterId uint32) (*CDNClusterStatus, error) {
	res := &CDNClusterStatus{}
	log.Warnf("1. CDNClusterGet --------> cdnClusterGetMethodId=%x clusterId=%v", d.cdnClusterGetMethodId, clusterId)
	err := d.callToRead(res, d.cdnClusterGetMethodId, types.U32(clusterId))
	log.Warnf("2. CDNClusterGet after --------> res: %v", res)

	return res, err
}

func (d *ddcBucketContract) CDNNodeGet(nodeId uint32) (*CDNNodeStatus, error) {
	res := &CDNNodeStatus{}
	log.Warnf("1. CDNNodeGet --------> cdnNodeGetMethodId=%x nodeId=%v", d.cdnNodeGetMethodId, nodeId)
	err := d.callToRead(res, d.cdnNodeGetMethodId, types.U32(nodeId))
	log.Warnf("2. CDNNodeGet after --------> res: %v", res)

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
	log.Warnf("callToRead method: %x ---> data: %v", method, data)
	if err != nil {
		return err
	}

	d.lastAccessTime = time.Now()

	res := Result{data: result}
	if err = res.decodeDdcBucketContract(data); err != nil {
		log.Warnf("callToRead method: %x ---> Failed decodeDdcBucketContract", method)
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
