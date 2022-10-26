package bucket

import (
	_ "embed"
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	bucketGetMethod  = "3802cb77"
	clusterGetMethod = "e75411f5"
	nodeGetMethod    = "847f3997"
	accountGetMethod = "1d4220fa"
)

type (
	DdcBucketContract interface {
		GetContractAddress() string
		GetLastAccessTime() time.Time

		BucketGet(bucketId uint32) (*BucketStatus, error)
		ClusterGet(clusterId uint32) (*ClusterStatus, error)
		NodeGet(nodeId uint32) (*NodeStatus, error)
		AccountGet(account types.AccountID) (*Account, error)
	}

	ddcBucketContract struct {
		contract            pkg.BlockchainClient
		lastAccessTime      time.Time
		contractAddressSS58 string
		keyringPair         signature.KeyringPair
		bucketGetMethodId   []byte
		clusterGetMethodId  []byte
		nodeGetMethodId     []byte
		accountGetMethodId  []byte
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

	accountGetMethodId, err := hex.DecodeString(accountGetMethod)
	if err != nil {
		log.WithError(err).WithField("method", accountGetMethod).Fatal("Can't decode method accountGetMethod")
	}

	return &ddcBucketContract{
		contract:            client,
		contractAddressSS58: contractAddressSS58,
		keyringPair:         signature.KeyringPair{Address: contractAddressSS58},
		bucketGetMethodId:   bucketGetMethodId,
		clusterGetMethodId:  clusterGetMethodId,
		nodeGetMethodId:     nodeGetMethodId,
		accountGetMethodId:  accountGetMethodId,
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
