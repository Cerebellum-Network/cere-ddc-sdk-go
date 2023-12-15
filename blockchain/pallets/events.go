package pallets

import "sync"

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
