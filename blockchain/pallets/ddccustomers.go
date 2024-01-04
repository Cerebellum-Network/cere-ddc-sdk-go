package pallets

import (
	"math"
	"sync"

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
}

type UnlockChunk struct {
	Value types.U128
	Block types.BlockNumber
}

// Events
type (
	EventDdcCustomersDeposited struct {
		Phase  types.Phase
		Owner  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}
	EventDdcCustomersInitialDepositUnlock struct {
		Phase  types.Phase
		Owner  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}
	EventDdcCustomersWithdrawn struct {
		Phase  types.Phase
		Owner  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}
	EventDdcCustomersCharged struct {
		Phase  types.Phase
		Owner  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}
	EventDdcCustomersBucketCreated struct {
		Phase    types.Phase
		BucketId BucketId
		Topics   []types.Hash
	}
	EventDdcCustomersBucketUpdated struct {
		Phase    types.Phase
		BucketId BucketId
		Topics   []types.Hash
	}
)

type DdcCustomersApi interface {
	GetBuckets(bucketId BucketId) (types.Option[Bucket], error)
	GetBucketsCount() (types.U64, error)
	GetLedger(owner types.AccountID) (types.Option[AccountsLedger], error)
}

type ddcCustomersEventsSubs struct {
	deposited            map[int]subscriber[EventDdcCustomersDeposited]
	initialDepositUnlock map[int]subscriber[EventDdcCustomersInitialDepositUnlock]
	withdrawn            map[int]subscriber[EventDdcCustomersWithdrawn]
	charged              map[int]subscriber[EventDdcCustomersCharged]
	bucketCreated        map[int]subscriber[EventDdcCustomersBucketCreated]
	bucketUpdated        map[int]subscriber[EventDdcCustomersBucketUpdated]
}

type ddcCustomersApi struct {
	substrateApi *gsrpc.SubstrateAPI
	meta         *types.Metadata

	subs *ddcCustomersEventsSubs
	mu   sync.Mutex
}

func NewDdcCustomersApi(substrateApi *gsrpc.SubstrateAPI, meta *types.Metadata, events <-chan *Events) DdcCustomersApi {
	subs := &ddcCustomersEventsSubs{
		deposited:            make(map[int]subscriber[EventDdcCustomersDeposited]),
		initialDepositUnlock: make(map[int]subscriber[EventDdcCustomersInitialDepositUnlock]),
		withdrawn:            make(map[int]subscriber[EventDdcCustomersWithdrawn]),
		charged:              make(map[int]subscriber[EventDdcCustomersCharged]),
		bucketCreated:        make(map[int]subscriber[EventDdcCustomersBucketCreated]),
		bucketUpdated:        make(map[int]subscriber[EventDdcCustomersBucketUpdated]),
	}

	api := &ddcCustomersApi{
		substrateApi: substrateApi,
		meta:         meta,
		subs:         subs,
		mu:           sync.Mutex{},
	}

	go func() {
		for blockEvents := range events {
			for _, e := range blockEvents.DdcCustomers_Deposited {
				api.mu.Lock()
				for i, sub := range api.subs.deposited {
					select {
					case <-sub.done:
						delete(api.subs.deposited, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcCustomers_InitialDepositUnlock {
				api.mu.Lock()
				for i, sub := range api.subs.initialDepositUnlock {
					select {
					case <-sub.done:
						delete(api.subs.initialDepositUnlock, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcCustomers_Withdrawn {
				api.mu.Lock()
				for i, sub := range api.subs.withdrawn {
					select {
					case <-sub.done:
						delete(api.subs.withdrawn, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcCustomers_Charged {
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

			for _, e := range blockEvents.DdcCustomers_BucketCreated {
				api.mu.Lock()
				for i, sub := range api.subs.bucketCreated {
					select {
					case <-sub.done:
						delete(api.subs.bucketCreated, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}

			for _, e := range blockEvents.DdcCustomers_BucketUpdated {
				api.mu.Lock()
				for i, sub := range api.subs.bucketUpdated {
					select {
					case <-sub.done:
						delete(api.subs.bucketUpdated, i)
					case sub.ch <- e:
					}
				}
				api.mu.Unlock()
			}
		}
	}()

	return api
}

func (api *ddcCustomersApi) GetBuckets(bucketId BucketId) (types.Option[Bucket], error) {
	maybeBucket := types.NewEmptyOption[Bucket]()

	bytes, err := codec.Encode(bucketId)
	if err != nil {
		return maybeBucket, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcCustomers", "Buckets", bytes)
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
	key, err := types.CreateStorageKey(api.meta, "DdcCustomers", "BucketsCount")
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

	bytes, err := codec.Encode(owner)
	if err != nil {
		return maybeLedger, err
	}

	key, err := types.CreateStorageKey(api.meta, "DdcCustomers", "Ledger", bytes)
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

func (api *ddcCustomersApi) SubscribeNewDeposited() (*NewEventSubscription[EventDdcCustomersDeposited], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.deposited == nil {
		api.subs.deposited = make(map[int]subscriber[EventDdcCustomersDeposited])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.deposited[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcCustomersDeposited)
	done := make(chan struct{})
	api.subs.deposited[idx] = subscriber[EventDdcCustomersDeposited]{ch, done}

	return &NewEventSubscription[EventDdcCustomersDeposited]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.deposited, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcCustomersApi) SubscribeNewInitialDepositUnlock() (*NewEventSubscription[EventDdcCustomersInitialDepositUnlock], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.initialDepositUnlock == nil {
		api.subs.initialDepositUnlock = make(map[int]subscriber[EventDdcCustomersInitialDepositUnlock])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.initialDepositUnlock[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcCustomersInitialDepositUnlock)
	done := make(chan struct{})
	api.subs.initialDepositUnlock[idx] = subscriber[EventDdcCustomersInitialDepositUnlock]{ch, done}

	return &NewEventSubscription[EventDdcCustomersInitialDepositUnlock]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.initialDepositUnlock, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcCustomersApi) SubscribeNewWithdrawn() (*NewEventSubscription[EventDdcCustomersWithdrawn], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.withdrawn == nil {
		api.subs.withdrawn = make(map[int]subscriber[EventDdcCustomersWithdrawn])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.withdrawn[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcCustomersWithdrawn)
	done := make(chan struct{})
	api.subs.withdrawn[idx] = subscriber[EventDdcCustomersWithdrawn]{ch, done}

	return &NewEventSubscription[EventDdcCustomersWithdrawn]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.withdrawn, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcCustomersApi) SubscribeNewCharged() (*NewEventSubscription[EventDdcCustomersCharged], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.charged == nil {
		api.subs.charged = make(map[int]subscriber[EventDdcCustomersCharged])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.charged[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcCustomersCharged)
	done := make(chan struct{})
	api.subs.charged[idx] = subscriber[EventDdcCustomersCharged]{ch, done}

	return &NewEventSubscription[EventDdcCustomersCharged]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.charged, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcCustomersApi) SubscribeNewBucketCreated() (*NewEventSubscription[EventDdcCustomersBucketCreated], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.bucketCreated == nil {
		api.subs.bucketCreated = make(map[int]subscriber[EventDdcCustomersBucketCreated])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.bucketCreated[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcCustomersBucketCreated)
	done := make(chan struct{})
	api.subs.bucketCreated[idx] = subscriber[EventDdcCustomersBucketCreated]{ch, done}

	return &NewEventSubscription[EventDdcCustomersBucketCreated]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.bucketCreated, idx)
			api.mu.Unlock()
		},
	}, nil
}

func (api *ddcCustomersApi) SubscribeNewBucketUpdated() (*NewEventSubscription[EventDdcCustomersBucketUpdated], error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.subs.bucketUpdated == nil {
		api.subs.bucketUpdated = make(map[int]subscriber[EventDdcCustomersBucketUpdated])
	}

	var idx int
	for i := 0; i <= math.MaxInt; i++ {
		if _, ok := api.subs.bucketUpdated[i]; !ok {
			idx = i
			break
		}
	}

	ch := make(chan EventDdcCustomersBucketUpdated)
	done := make(chan struct{})
	api.subs.bucketUpdated[idx] = subscriber[EventDdcCustomersBucketUpdated]{ch, done}

	return &NewEventSubscription[EventDdcCustomersBucketUpdated]{
		ch:   ch,
		done: done,
		onDone: func() {
			api.mu.Lock()
			delete(api.subs.bucketUpdated, idx)
			api.mu.Unlock()
		},
	}, nil
}
