package pallets

import (
	"reflect"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

type BatchIndex = types.U16

type BillingReport struct {
	State                     State
	Vault                     types.AccountID
	TotalCustomerCharges      CustomerCharge
	TotalDistributedReward    types.U128
	TotalNodeUsage            NodeUsage
	ChargingMaxBatchIndex     BatchIndex
	ChargingProcessedBatches  []BatchIndex
	RewardingMaxBatchIndex    BatchIndex
	RewardingProcessedBatches []BatchIndex
}

type CustomerCharge struct {
	Transfer types.U128
	Storage  types.U128
	Puts     types.U128
	Gets     types.U128
}

type NodeUsage struct {
	TransferredBytes types.U64
	StoredBytes      types.U64
	NumberOfPuts     types.U128
	NumberOfGets     types.U128
}

type State struct {
	IsNotInitialized           bool
	IsInitialized              bool
	IsChargingCustomers        bool
	IsCustomersChargedWithFees bool
	IsRewardingProviders       bool
	IsProvidersRewarded        bool
	IsFinalized                bool
}

func (m *State) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	i := int(b)

	v := reflect.ValueOf(m)
	if i > v.NumField() {
		return ErrUnknownVariant
	}

	v.Field(i).SetBool(true)

	return nil
}

func (m State) Encode(encoder scale.Encoder) error {
	var err1, err2 error
	v := reflect.ValueOf(m)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Bool() {
			err1 = encoder.PushByte(byte(i))
			err2 = encoder.Encode(i + 1) // values are defined from 1
			break
		}
		if i == v.NumField()-1 {
			return ErrUnknownVariant
		}
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

// Events
type (
	EventDdcPayoutsBillingReportInitialized struct {
		Phase     types.Phase
		ClusterId ClusterId
		Topics    []types.Hash
	}

	EventDdcPayoutsChargingStarted struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Topics    []types.Hash
	}

	EventDdcPayoutsCharged struct {
		Phase      types.Phase
		ClusterId  ClusterId
		Era        DdcEra
		BatchIndex BatchIndex
		CustomerId types.AccountID
		Amount     types.U128
		Topics     []types.Hash
	}

	EventDdcPayoutsChargeFailed struct {
		Phase      types.Phase
		ClusterId  ClusterId
		Era        DdcEra
		BatchIndex BatchIndex
		CustomerId types.AccountID
		Amount     types.U128
		Topics     []types.Hash
	}

	EventDdcPayoutsIndebted struct {
		Phase      types.Phase
		ClusterId  ClusterId
		Era        DdcEra
		BatchIndex BatchIndex
		CustomerId types.AccountID
		Amount     types.U128
		Topics     []types.Hash
	}

	EventDdcPayoutsChargingFinished struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Topics    []types.Hash
	}

	EventDdcPayoutsTreasuryFeesCollected struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Topics    []types.Hash
	}

	EventDdcPayoutsClusterReserveFeesCollected struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Amount    types.U128
		Topics    []types.Hash
	}

	EventDdcPayoutsValidatorFeesCollected struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Amount    types.U128
		Topics    []types.Hash
	}

	EventDdcPayoutsRewardingStarted struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Topics    []types.Hash
	}

	EventDdcPayouts struct {
		Phase     types.Phase
		ClusterId ClusterId
		Topics    []types.Hash
	}

	EventDdcPayoutsRewarded struct {
		Phase          types.Phase
		ClusterId      ClusterId
		Era            DdcEra
		NodeProviderId types.AccountID
		Amount         types.U128
		Topics         []types.Hash
	}

	EventDdcPayoutsRewardingFinished struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Topics    []types.Hash
	}

	EventDdcPayoutsBillingReportFinalized struct {
		Phase     types.Phase
		ClusterId ClusterId
		Era       DdcEra
		Topics    []types.Hash
	}

	EventDdcPayoutsAuthorisedCaller struct {
		Phase            types.Phase
		AuthorisedCaller types.AccountID
		Topics           []types.Hash
	}
)

type DdcPayoutsApi interface {
	GetActiveBillingReports(cluster ClusterId, era DdcEra) (types.Option[BillingReport], error)
	GetAuthorisedCaller() (types.Option[types.AccountID], error)
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

func (api *ddcPayoutsApi) GetActiveBillingReports(cluster ClusterId, era DdcEra) (types.Option[BillingReport], error) {
	maybeV := types.NewEmptyOption[BillingReport]()

	bytesCluster, err := codec.Encode(cluster)
	if err != nil {
		return maybeV, err
	}

	bytesEra, err := codec.Encode(era)
	if err != nil {
		return maybeV, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcPayouts", "DebtorCustomers", bytesCluster, bytesEra)
	if err != nil {
		return maybeV, err
	}

	var v BillingReport
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &v)
	if !ok || err != nil {
		return maybeV, err
	}

	maybeV.SetSome(v)

	return maybeV, nil
}

func (api *ddcPayoutsApi) GetAuthorisedCaller() (types.Option[types.AccountID], error) {
	maybeV := types.NewEmptyOption[types.AccountID]()

	key, err := types.CreateStorageKey(api.meta, "DdcPayouts", "AuthorisedCaller")
	if err != nil {
		return maybeV, err
	}

	var v types.AccountID
	ok, err := api.substrateApi.RPC.State.GetStorageLatest(key, &v)
	if !ok || err != nil {
		return maybeV, err
	}

	maybeV.SetSome(v)

	return maybeV, nil
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
