package pallets

import (
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
)

type NewEventSubscription[T any] struct {
	ch   chan T
	done chan struct{}
	o    sync.Once
}

func (s *NewEventSubscription[T]) Unsubscribe() {
	s.o.Do(func() {
		s.done <- struct{}{}
		close(s.ch)
	})
}

func (s *NewEventSubscription[T]) Chan() <-chan T {
	return s.ch
}

type subscriber struct {
	ch   chan *parser.Event
	done chan struct{}
}

type Publisher interface {
	Subs() map[string]map[int]subscriber
	Mu() *sync.Mutex
}

func AddSubscriber(pub Publisher, eventName string) int {
	pub.Mu().Lock()
	defer pub.Mu().Unlock()

	if pub.Subs()[eventName] == nil {
		pub.Subs()[eventName] = make(map[int]subscriber)
	}

	key := len(pub.Subs()[eventName])

	pub.Subs()[eventName][key] = subscriber{
		ch:   make(chan *parser.Event, 256),
		done: make(chan struct{}),
	}

	return key
}
