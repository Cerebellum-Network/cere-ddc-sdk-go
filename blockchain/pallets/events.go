package pallets

import (
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords

	DdcClusters_ClusterCreated      []EventDdcClustersClusterCreated      //nolint:stylecheck,golint
	DdcClusters_ClusterNodeAdded    []EventDdcClustersClusterNodeAdded    //nolint:stylecheck,golint
	DdcClusters_ClusterNodeRemoved  []EventDdcClustersClusterNodeRemoved  //nolint:stylecheck,golint
	DdcClusters_ClusterParamsSet    []EventDdcClustersClusterParamsSet    //nolint:stylecheck,golint
	DdcClusters_ClusterGovParamsSet []EventDdcClustersClusterGovParamsSet //nolint:stylecheck,golint

	DdcCustomers_Deposited            []EventDdcCustomersDeposited            //nolint:stylecheck,golint
	DdcCustomers_InitialDepositUnlock []EventDdcCustomersInitialDepositUnlock //nolint:stylecheck,golint
	DdcCustomers_Withdrawn            []EventDdcCustomersWithdrawn            //nolint:stylecheck,golint
	DdcCustomers_Charged              []EventDdcCustomersCharged              //nolint:stylecheck,golint
	DdcCustomers_BucketCreated        []EventDdcCustomersBucketCreated        //nolint:stylecheck,golint
	DdcCustomers_BucketUpdated        []EventDdcCustomersBucketUpdated        //nolint:stylecheck,golint
}

type NewEventSubscription[T any] struct {
	ch     chan T
	done   chan struct{}
	onDone func()
	o      sync.Once
}

func (s *NewEventSubscription[T]) Unsubscribe() {
	s.o.Do(func() {
		s.done <- struct{}{}
		close(s.ch)
		s.onDone()
	})
}

func (s *NewEventSubscription[T]) Chan() <-chan T {
	return s.ch
}

type subscriber[T any] struct {
	ch   chan T
	done chan struct{}
}
