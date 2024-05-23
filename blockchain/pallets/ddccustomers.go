package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

type AccountsLedger struct {
	Owner     types.AccountID
	Total     types.UCompact
	Active    types.UCompact
	Unlocking []UnlockChunk
}

type Bucket struct {
	BucketId  BucketId
	OwnerId   types.AccountID
	ClusterId ClusterId
	IsPublic  types.Bool
	IsRemoved types.Bool
}

type UnlockChunk struct {
	Value types.U128
	Block types.BlockNumber
}

type DdcCustomersApi interface {
	GetBuckets(bucketId BucketId) (types.Option[Bucket], error)
	GetBucketsCount() (types.U64, error)
	GetLedger(owner types.AccountID) (types.Option[AccountsLedger], error)
}

type ddcCustomersApi struct {
	substrateApi *gsrpc.SubstrateAPI
}

func NewDdcCustomersApi(substrateApi *gsrpc.SubstrateAPI) DdcCustomersApi {
	return &ddcCustomersApi{substrateApi}
}

func (api *ddcCustomersApi) GetBuckets(bucketId BucketId) (types.Option[Bucket], error) {
	maybeBucket := types.NewEmptyOption[Bucket]()

	meta, err := api.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return maybeBucket, err
	}

	bytes, err := codec.Encode(bucketId)
	if err != nil {
		return maybeBucket, err
	}

	key, err := types.CreateStorageKey(meta, "DdcCustomers", "Buckets", bytes)
	if err != nil {
		return maybeBucket, err
	}

	var bucket Bucket
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &bucket)
	if !ok || err != nil {
		return maybeBucket, err
	}

	maybeBucket.SetSome(bucket)

	return maybeBucket, nil
}

func (api *ddcCustomersApi) GetBucketsCount() (types.U64, error) {
	meta, err := api.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return 0, err
	}

	key, err := types.CreateStorageKey(meta, "DdcCustomers", "BucketsCount")
	if err != nil {
		return 0, err
	}

	var bucketsCount types.U64
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &bucketsCount)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}

	return bucketsCount, nil
}

func (api *ddcCustomersApi) GetLedger(owner types.AccountID) (types.Option[AccountsLedger], error) {
	maybeLedger := types.NewEmptyOption[AccountsLedger]()

	meta, err := api.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return maybeLedger, err
	}

	bytes, err := codec.Encode(owner)
	if err != nil {
		return maybeLedger, err
	}

	key, err := types.CreateStorageKey(meta, "DdcCustomers", "Ledger", bytes)
	if err != nil {
		return maybeLedger, err
	}

	var accountsLedger AccountsLedger
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &accountsLedger)
	if !ok || err != nil {
		return maybeLedger, err
	}

	maybeLedger.SetSome(accountsLedger)

	return maybeLedger, nil
}
