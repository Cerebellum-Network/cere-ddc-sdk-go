package pallets

import (
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type DdcVerificationApi interface {
	GetActiveInspectors() ([]types.AccountID, error)
}

type ddcVerificationApi struct {
	substrateApi *gsrpc.SubstrateAPI
	meta         *types.Metadata
}

func NewDdcVerificationApi(substrateApi *gsrpc.SubstrateAPI, meta *types.Metadata) DdcVerificationApi {
	return &ddcVerificationApi{
		substrateApi,
		meta,
	}
}

func (api *ddcVerificationApi) GetActiveInspectors() ([]types.AccountID, error) {

	key, err := types.CreateStorageKey(api.meta, "DdcVerification", "ValidatorSet")
	if err != nil {
		return nil, err
	}

	v := make([]types.AccountID, 0)
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &v)
	if !ok || err != nil {
		return v, err
	}

	return v, nil
}
