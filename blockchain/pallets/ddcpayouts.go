package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

type DdcPayoutsApi interface {
	GetDebtorCustomers(cluster ClusterId, account types.AccountID) (types.Option[types.U128], error)
}

type ddcPayoutsApi struct {
	substrateApi *gsrpc.SubstrateAPI
	meta         *types.Metadata
}

func NewDdcPayoutsApi(substrateApi *gsrpc.SubstrateAPI, meta *types.Metadata) DdcPayoutsApi {
	return &ddcPayoutsApi{
		substrateApi,
		meta,
	}
}

func (api *ddcPayoutsApi) GetDebtorCustomers(cluster ClusterId, account types.AccountID) (types.Option[types.U128], error) {
	maybeV := types.NewEmptyOption[types.U128]()

	bytesCluster, err := codec.Encode(cluster)
	if err != nil {
		return maybeV, err
	}

	bytesAccount, err := codec.Encode(account)
	if err != nil {
		return maybeV, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcPayouts", "DebtorCustomers", bytesCluster, bytesAccount)
	if err != nil {
		return maybeV, err
	}

	var v types.U128
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &v)
	if !ok || err != nil {
		return maybeV, err
	}

	maybeV.SetSome(v)

	return maybeV, nil
}
