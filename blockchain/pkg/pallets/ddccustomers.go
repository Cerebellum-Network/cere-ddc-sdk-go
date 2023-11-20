package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/cerebellum-network/cere-ddc-sdk-go/blockchain/pkg/ddcprimitives"
)

type AccountsLedger struct {
	Owner     types.AccountID
	Total     types.U128
	Active    types.U128
	Unlocking []UnlockChunk
}

type Bucket struct {
	BucketID  ddcprimitives.BucketID
	OwnerID   types.AccountID
	ClusterID ddcprimitives.ClusterID
}

type BucketDetails struct {
	BucketID ddcprimitives.BucketID
	Amount   types.U128
}

type UnlockChunk struct {
	Value types.U128
	Block types.BlockNumber
}

type Buckets map[ddcprimitives.BucketID]types.Option[Bucket]

type Ledger map[types.AccountID]types.Option[AccountsLedger]

type DDCCustomersAPI struct {
	substrateAPI *gsrpc.SubstrateAPI
	meta         *types.Metadata
}

func NewDDCCustomersAPI(substrateAPI *gsrpc.SubstrateAPI, meta *types.Metadata) *DDCCustomersAPI {
	return &DDCCustomersAPI{
		substrateAPI,
		meta,
	}
}

func (api *DDCCustomersAPI) GetBucket(bucketID ddcprimitives.BucketID) (types.Option[Bucket], error) {
	maybeBucket := types.NewEmptyOption[Bucket]()

	bytes, err := codec.Encode(bucketID)
	if err != nil {
		return maybeBucket, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcCustomers", "Buckets", bytes)
	if err != nil {
		return maybeBucket, err
	}

	var bucket Bucket
	ok, err := api.substrateAPI.RPC.State.GetStorageLatest(key, &bucket)
	if !ok || err != nil {
		return maybeBucket, err
	}

	maybeBucket.SetSome(bucket)

	return maybeBucket, nil
}

func (api *DDCCustomersAPI) GetBucketsCount() (types.U64, error) {
	key, err := types.CreateStorageKey(api.meta, "DdcCustomers", "BucketsCount")
	if err != nil {
		return 0, err
	}

	var bucketsCount types.U64
	ok, err := api.substrateAPI.RPC.State.GetStorageLatest(key, &bucketsCount)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}

	return bucketsCount, nil
}

func (api *DDCCustomersAPI) GetLedger(owner types.AccountID) (types.Option[AccountsLedger], error) {
	maybeLedger := types.NewEmptyOption[AccountsLedger]()

	bytes, err := codec.Encode(owner)
	if err != nil {
		return maybeLedger, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcCustomers", "Ledger", bytes)
	if err != nil {
		return maybeLedger, err
	}

	var accountsLedger AccountsLedger
	ok, err := api.substrateAPI.RPC.State.GetStorageLatest(key, &accountsLedger)
	if !ok || err != nil {
		return maybeLedger, err
	}

	maybeLedger.SetSome(accountsLedger)

	return maybeLedger, nil
}
