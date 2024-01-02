package pallets

import (
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords

	DdcClusters_ClusterCreated      []EventDdcClustersClusterCreated      //nolint:stylecheck,golint
	DdcClusters_ClusterNodeAdded    []EventDdcClustersClusterNodeAdded    //nolint:stylecheck,golint
	DdcClusters_ClusterNodeRemoved  []EventDdcClustersClusterNodeRemoved  //nolint:stylecheck,golint
	DdcClusters_ClusterParamsSet    []EventDdcClustersClusterParamsSet    //nolint:stylecheck,golint
	DdcClusters_ClusterGovParamsSet []EventDdcClustersClusterGovParamsSet //nolint:stylecheck,golint
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

type subscriber struct {
	ch   chan *parser.Event
	done chan struct{}
}
