package bucket

import (
	_ "embed"
	"encoding/hex"
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

	BucketCreatedEvent                = "004464634275636b65743a3a4275636b65744372656174656400000000000000"
	BucketAllocatedEvent              = "004464634275636b65743a3a4275636b6574416c6c6f63617465640000000000"
	BucketSettlePaymentEvent          = "004464634275636b65743a3a4275636b6574536574746c655061796d656e7400"
	ClusterCreatedEvent               = "004464634275636b65743a3a436c757374657243726561746564000000000000"
	ClusterNodeReplacedEvent          = "004464634275636b65743a3a436c75737465724e6f64655265706c6163656400"
	ClusterReserveResourceEvent       = "84d6d26a3275dced8e359779bf21488762a1d88029f52522d8fc27607759399e"
	ClusterDistributeRevenuesEvent    = "65441936759a16fb28d0e059b878f2e48283ca2eac57c396a8035cce9e2acdd3"
	CdnClusterCreatedEvent            = "004464634275636b65743a3a43646e436c757374657243726561746564000000"
	CdnClusterDistributeRevenuesEvent = "4e19fc4683a9a741a09d89a1d62b22d592a8bf10e2b8b6eff7c6742a0ed88bb4"
	CdnNodeCreatedEvent               = "004464634275636b65743a3a43646e4e6f646543726561746564000000000000"
	NodeCreatedEvent                  = "004464634275636b65743a3a4e6f646543726561746564000000000000000000"
	DepositEvent                      = "004464634275636b65743a3a4465706f73697400000000000000000000000000"
	GrantPermissionEvent              = "004464634275636b65743a3a4772616e745065726d697373696f6e0000000000"
	RevokePermissionEvent             = "004464634275636b65743a3a5265766f6b655065726d697373696f6e00000000"
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
	}
)

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

func (d *ddcBucketContract) NodeGet(nodeId uint32) (*NodeStatus, error) {
	res := &NodeStatus{}
	err := d.callToRead(res, d.nodeGetMethodId, types.U32(nodeId))

	return res, err
}

func (d *ddcBucketContract) CDNClusterGet(clusterId uint32) (*CDNClusterStatus, error) {
	res := &CDNClusterStatus{}
	err := d.callToRead(res, d.cdnClusterGetMethodId, types.U32(clusterId))

	return res, err
}

func (d *ddcBucketContract) CDNNodeGet(nodeId uint32) (*CDNNodeStatus, error) {
	res := &CDNNodeStatus{}
	err := d.callToRead(res, d.cdnNodeGetMethodId, types.U32(nodeId))

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

func (d *ddcBucketContract) GetContractAddress() string {
	return d.contractAddressSS58
}

func (d *ddcBucketContract) GetLastAccessTime() time.Time {
	return d.lastAccessTime
}
