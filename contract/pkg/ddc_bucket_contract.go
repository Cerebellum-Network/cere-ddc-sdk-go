package pkg

import (
	_ "embed"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/abi"
	"github.com/patractlabs/go-patract/metadata"
	"github.com/patractlabs/go-patract/rpc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"time"
)

type DdcBucketContract interface {
	GetApiUrl() string
	GetAccountId() string

	BucketGet(bucketId uint32) (*BucketStatus, error)
	ClusterGet(clusterId uint32) (*ClusterStatus, error)
	NodeGet(nodeId uint32) (*NodeStatus, error)
}

type ddcBucketContract struct {
	contract       Contract
	lastAccessTime time.Time
	apiUrl         string
	accountId      string
	keyringPair    signature.KeyringPair
}

func CreateDdcBucketContract(apiUrl string, accountId string) DdcBucketContract {
	smartContract, err := rpc.NewContractAPI(apiUrl)
	if err != nil {
		log.WithError(err).WithField("apiUrl", apiUrl).Fatal("Can't initialize ddc bucket contract api")
	}

	if err := smartContract.WithMetaData(abi.DdcBucket); err != nil {
		log.WithError(err).Fatal("Can't initialize ddc bucket contract metadata")
	}

	contractMetadata, _ := metadata.New(abi.DdcBucket)

	log.WithFields(log.Fields{"apiUrl": apiUrl, "accountId": accountId}).Info("Ddc bucket contract configured")
	return &ddcBucketContract{
		contract:    CreateContract(smartContract, accountId, contractMetadata),
		apiUrl:      apiUrl,
		accountId:   accountId,
		keyringPair: signature.KeyringPair{Address: accountId},
	}
}

func (d *ddcBucketContract) BucketGet(bucketId uint32) (*BucketStatus, error) {
	req := types.U32(bucketId)
	ctx := rpc.NewCtx(context.Background()).WithFrom(d.keyringPair)

	data, err := d.contract.CallToReadEncoded(ctx, []string{"bucket_get"}, req)
	if err != nil {
		return nil, err
	}

	d.lastAccessTime = time.Now()

	res := Result{data: &BucketStatus{}}
	if err = res.decodeDdcBucketContract(data); err != nil {
		return nil, err
	}

	return res.data.(*BucketStatus), res.err
}

func (d *ddcBucketContract) ClusterGet(clusterId uint32) (*ClusterStatus, error) {
	req := types.U32(clusterId)
	ctx := rpc.NewCtx(context.Background()).WithFrom(d.keyringPair)

	data, err := d.contract.CallToReadEncoded(ctx, []string{"cluster_get"}, req)
	if err != nil {
		return nil, err
	}

	d.lastAccessTime = time.Now()

	res := Result{data: &ClusterStatus{}}
	if err = res.decodeDdcBucketContract(data); err != nil {
		return nil, err
	}

	return res.data.(*ClusterStatus), res.err
}

func (d *ddcBucketContract) NodeGet(nodeId uint32) (*NodeStatus, error) {
	req := types.U32(nodeId)
	ctx := rpc.NewCtx(context.Background()).WithFrom(d.keyringPair)

	data, err := d.contract.CallToReadEncoded(ctx, []string{"node_get"}, req)
	if err != nil {
		return nil, err
	}

	d.lastAccessTime = time.Now()

	res := Result{data: &NodeStatus{}}
	if err = res.decodeDdcBucketContract(data); err != nil {
		return nil, err
	}

	return res.data.(*NodeStatus), res.err
}

func (d *ddcBucketContract) GetApiUrl() string {
	return d.apiUrl
}

func (d *ddcBucketContract) GetAccountId() string {
	return d.accountId
}

func (d *ddcBucketContract) GetLastAccessTime() time.Time {
	return d.lastAccessTime
}
