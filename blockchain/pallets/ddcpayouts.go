package pallets

import (
	"math"
	"reflect"
	"sync"

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

type ddcPayoutsEventsSubs struct {
	billingReportInitialized    map[int]subscriber[EventDdcPayoutsBillingReportInitialized]
	chargingStarted             map[int]subscriber[EventDdcPayoutsChargingStarted]
	charged                     map[int]subscriber[EventDdcPayoutsCharged]
	chargeFailed                map[int]subscriber[EventDdcPayoutsChargeFailed]
	indebted                    map[int]subscriber[EventDdcPayoutsIndebted]
	chargingFinished            map[int]subscriber[EventDdcPayoutsChargingFinished]
	treasuryFeesCollected       map[int]subscriber[EventDdcPayoutsTreasuryFeesCollected]
	clusterReserveFeesCollected map[int]subscriber[EventDdcPayoutsClusterReserveFeesCollected]
	validatorFeesCollected      map[int]subscriber[EventDdcPayoutsValidatorFeesCollected]
	rewardingStarted            map[int]subscriber[EventDdcPayoutsRewardingStarted]
	rewarded                    map[int]subscriber[EventDdcPayoutsRewarded]
	rewardingFinished           map[int]subscriber[EventDdcPayoutsRewardingFinished]
	billingReportFinalized      map[int]subscriber[EventDdcPayoutsBillingReportFinalized]
	authorisedCaller            map[int]subscriber[EventDdcPayoutsAuthorisedCaller]
}

type ddcPayoutsApi struct {
	substrateApi *gsrpc.SubstrateAPI
	meta         *types.Metadata

	subs *ddcPayoutsEventsSubs
	mu   sync.Mutex
}

func NewDdcPayoutsApi(substrateApi *gsrpc.SubstrateAPI, meta *types.Metadata, events <-chan *Events) DdcPayoutsApi {
	subs := &ddcPayoutsEventsSubs{
		billingReportInitialized:    make(map[int]subscriber[EventDdcPayoutsBillingReportInitialized]),
		chargingStarted:             make(map[int]subscriber[EventDdcPayoutsChargingStarted]),
		charged:                     make(map[int]subscriber[EventDdcPayoutsCharged]),
		chargeFailed:                make(map[int]subscriber[EventDdcPayoutsChargeFailed]),
		indebted:                    make(map[int]subscriber[EventDdcPayoutsIndebted]),
		chargingFinished:            make(map[int]subscriber[EventDdcPayoutsChargingFinished]),
		treasuryFeesCollected:       make(map[int]subscriber[EventDdcPayoutsTreasuryFeesCollected]),
		clusterReserveFeesCollected: make(map[int]subscriber[EventDdcPayoutsClusterReserveFeesCollected]),
		validatorFeesCollected:      make(map[int]subscriber[EventDdcPayoutsValidatorFeesCollected]),
		rewardingStarted:            make(map[int]subscriber[EventDdcPayoutsRewardingStarted]),
		rewarded:                    make(map[int]subscriber[EventDdcPayoutsRewarded]),
		rewardingFinished:           make(map[int]subscriber[EventDdcPayoutsRewardingFinished]),
		billingReportFinalized:      make(map[int]subscriber[EventDdcPayoutsBillingReportFinalized]),
		authorisedCaller:            make(map[int]subscriber[EventDdcPayoutsAuthorisedCaller]),
	}

	api := &ddcPayoutsApi{
		substrateApi: substrateApi,
		meta:         meta,
		subs:         subs,
		mu:           sync.Mutex{},
	}

	go func() {
		for blockEvents := range events {
			for _, e := range blockEvents.DdcPayouts_BillingReportInitialized {
				api.mu.Lock()
				for i, sub := range api.subs.billingReportInitialized {
					select {
					case <-sub.done:
						delete(api.subs.billingReportInitialized, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_ChargingStarted {
				api.mu.Lock()
				for i, sub := range api.subs.chargingStarted {
					select {
					case <-sub.done:
						delete(api.subs.chargingStarted, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_Charged {
				api.mu.Lock()
				for i, sub := range api.subs.charged {
					select {
					case <-sub.done:
						delete(api.subs.charged, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_ChargeFailed {
				api.mu.Lock()
				for i, sub := range api.subs.chargeFailed {
					select {
					case <-sub.done:
						delete(api.subs.chargeFailed, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_Indebted {
				api.mu.Lock()
				for i, sub := range api.subs.indebted {
					select {
					case <-sub.done:
						delete(api.subs.indebted, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_ChargingFinished {
				api.mu.Lock()
				for i, sub := range api.subs.chargingFinished {
					select {
					case <-sub.done:
						delete(api.subs.chargingFinished, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_TreasuryFeesCollected {
				api.mu.Lock()
				for i, sub := range api.subs.treasuryFeesCollected {
					select {
					case <-sub.done:
						delete(api.subs.treasuryFeesCollected, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_ClusterReserveFeesCollected {
				api.mu.Lock()
				for i, sub := range api.subs.clusterReserveFeesCollected {
					select {
					case <-sub.done:
						delete(api.subs.clusterReserveFeesCollected, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_ValidatorFeesCollected {
				api.mu.Lock()
				for i, sub := range api.subs.validatorFeesCollected {
					select {
					case <-sub.done:
						delete(api.subs.validatorFeesCollected, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_RewardingStarted {
				api.mu.Lock()
				for i, sub := range api.subs.rewardingStarted {
					select {
					case <-sub.done:
						delete(api.subs.rewardingStarted, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_Rewarded {
				api.mu.Lock()
				for i, sub := range api.subs.rewarded {
					select {
					case <-sub.done:
						delete(api.subs.rewarded, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_RewardingFinished {
				api.mu.Lock()
				for i, sub := range api.subs.rewardingFinished {
					select {
					case <-sub.done:
						delete(api.subs.rewardingFinished, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_BillingReportFinalized {
				api.mu.Lock()
				for i, sub := range api.subs.billingReportFinalized {
					select {
					case <-sub.done:
						delete(api.subs.billingReportFinalized, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcPayouts_AuthorisedCaller {
				api.mu.Lock()
				for i, sub := range api.subs.authorisedCaller {
					select {
					case <-sub.done:
						delete(api.subs.authorisedCaller, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}
		}
	}()

	return api
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

func (api *ddcPayoutsApi) SubscribeNewBillingReportInitialized() (*NewEventSubscription[EventDdcPayoutsBillingReportInitialized], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.billingReportInitialized == nil {
		api.subs.billingReportInitialized = make(map[int]subscriber[EventDdcPayoutsBillingReportInitialized])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.billingReportInitialized[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsBillingReportInitialized)
	done := make(chan struct{})
	api.subs.billingReportInitialized[idx] = subscriber[EventDdcPayoutsBillingReportInitialized]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsBillingReportInitialized]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.billingReportInitialized, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewChargingStarted() (*NewEventSubscription[EventDdcPayoutsChargingStarted], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.chargingStarted == nil {
		api.subs.chargingStarted = make(map[int]subscriber[EventDdcPayoutsChargingStarted])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.chargingStarted[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsChargingStarted)
	done := make(chan struct{})
	api.subs.chargingStarted[idx] = subscriber[EventDdcPayoutsChargingStarted]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsChargingStarted]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.chargingStarted, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewCharged() (*NewEventSubscription[EventDdcPayoutsCharged], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.charged == nil {
		api.subs.charged = make(map[int]subscriber[EventDdcPayoutsCharged])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.charged[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsCharged)
	done := make(chan struct{})
	api.subs.charged[idx] = subscriber[EventDdcPayoutsCharged]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsCharged]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.charged, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewChargeFailed() (*NewEventSubscription[EventDdcPayoutsChargeFailed], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.chargeFailed == nil {
		api.subs.chargeFailed = make(map[int]subscriber[EventDdcPayoutsChargeFailed])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.chargeFailed[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsChargeFailed)
	done := make(chan struct{})
	api.subs.chargeFailed[idx] = subscriber[EventDdcPayoutsChargeFailed]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsChargeFailed]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.chargeFailed, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewIndebted() (*NewEventSubscription[EventDdcPayoutsIndebted], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.indebted == nil {
		api.subs.indebted = make(map[int]subscriber[EventDdcPayoutsIndebted])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.indebted[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsIndebted)
	done := make(chan struct{})
	api.subs.indebted[idx] = subscriber[EventDdcPayoutsIndebted]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsIndebted]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.indebted, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewChargingFinished() (*NewEventSubscription[EventDdcPayoutsChargingFinished], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.chargingFinished == nil {
		api.subs.chargingFinished = make(map[int]subscriber[EventDdcPayoutsChargingFinished])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.chargingFinished[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsChargingFinished)
	done := make(chan struct{})
	api.subs.chargingFinished[idx] = subscriber[EventDdcPayoutsChargingFinished]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsChargingFinished]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.chargingFinished, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewTreasuryFeesCollected() (*NewEventSubscription[EventDdcPayoutsTreasuryFeesCollected], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.treasuryFeesCollected == nil {
		api.subs.treasuryFeesCollected = make(map[int]subscriber[EventDdcPayoutsTreasuryFeesCollected])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.treasuryFeesCollected[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsTreasuryFeesCollected)
	done := make(chan struct{})
	api.subs.treasuryFeesCollected[idx] = subscriber[EventDdcPayoutsTreasuryFeesCollected]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsTreasuryFeesCollected]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.treasuryFeesCollected, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewClusterReserveFeesCollected() (*NewEventSubscription[EventDdcPayoutsClusterReserveFeesCollected], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.clusterReserveFeesCollected == nil {
		api.subs.clusterReserveFeesCollected = make(map[int]subscriber[EventDdcPayoutsClusterReserveFeesCollected])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.clusterReserveFeesCollected[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsClusterReserveFeesCollected)
	done := make(chan struct{})
	api.subs.clusterReserveFeesCollected[idx] = subscriber[EventDdcPayoutsClusterReserveFeesCollected]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsClusterReserveFeesCollected]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.clusterReserveFeesCollected, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewValidatorFeesCollected() (*NewEventSubscription[EventDdcPayoutsValidatorFeesCollected], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.validatorFeesCollected == nil {
		api.subs.validatorFeesCollected = make(map[int]subscriber[EventDdcPayoutsValidatorFeesCollected])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.validatorFeesCollected[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsValidatorFeesCollected)
	done := make(chan struct{})
	api.subs.validatorFeesCollected[idx] = subscriber[EventDdcPayoutsValidatorFeesCollected]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsValidatorFeesCollected]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.validatorFeesCollected, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewRewardingStarted() (*NewEventSubscription[EventDdcPayoutsRewardingStarted], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.rewardingStarted == nil {
		api.subs.rewardingStarted = make(map[int]subscriber[EventDdcPayoutsRewardingStarted])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.rewardingStarted[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsRewardingStarted)
	done := make(chan struct{})
	api.subs.rewardingStarted[idx] = subscriber[EventDdcPayoutsRewardingStarted]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsRewardingStarted]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.rewardingStarted, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewRewarded() (*NewEventSubscription[EventDdcPayoutsRewarded], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.rewarded == nil {
		api.subs.rewarded = make(map[int]subscriber[EventDdcPayoutsRewarded])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.rewarded[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsRewarded)
	done := make(chan struct{})
	api.subs.rewarded[idx] = subscriber[EventDdcPayoutsRewarded]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsRewarded]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.rewarded, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewRewardingFinished() (*NewEventSubscription[EventDdcPayoutsRewardingFinished], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.rewardingFinished == nil {
		api.subs.rewardingFinished = make(map[int]subscriber[EventDdcPayoutsRewardingFinished])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.rewardingFinished[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsRewardingFinished)
	done := make(chan struct{})
	api.subs.rewardingFinished[idx] = subscriber[EventDdcPayoutsRewardingFinished]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsRewardingFinished]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.rewardingFinished, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewBillingReportFinalized() (*NewEventSubscription[EventDdcPayoutsBillingReportFinalized], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.billingReportFinalized == nil {
		api.subs.billingReportFinalized = make(map[int]subscriber[EventDdcPayoutsBillingReportFinalized])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.billingReportFinalized[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsBillingReportFinalized)
	done := make(chan struct{})
	api.subs.billingReportFinalized[idx] = subscriber[EventDdcPayoutsBillingReportFinalized]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsBillingReportFinalized]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.billingReportFinalized, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcPayoutsApi) SubscribeNewAuthorisedCaller() (*NewEventSubscription[EventDdcPayoutsAuthorisedCaller], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.authorisedCaller == nil {
		api.subs.authorisedCaller = make(map[int]subscriber[EventDdcPayoutsAuthorisedCaller])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.authorisedCaller[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcPayoutsAuthorisedCaller)
	done := make(chan struct{})
	api.subs.authorisedCaller[idx] = subscriber[EventDdcPayoutsAuthorisedCaller]{ch, done}

	return &NewEventSubscription[EventDdcPayoutsAuthorisedCaller]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.authorisedCaller, idx)
			api.mu.Unlock()
		},
	}, nil
}
