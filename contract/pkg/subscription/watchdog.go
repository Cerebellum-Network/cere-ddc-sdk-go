package subscription

import "time"

//go:generate mockgen -destination=../mock/subscription_WatchdogFactory.go -package=mock github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription WatchdogFactory
//go:generate mockgen -destination=../mock/subscription_Watchdog.go -package=mock github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription Watchdog

type Watchdog interface {
	C() <-chan time.Time
}

var _ Watchdog = &watchdog{}

type watchdog struct {
	*time.Ticker
}

func (w *watchdog) C() <-chan time.Time {
	return w.Ticker.C
}

type WatchdogFactory interface {
	NewWatchdog(timeout time.Duration) Watchdog
}

type watchdogFactory struct {
}

func (w *watchdogFactory) NewWatchdog(timeout time.Duration) Watchdog {
	return &watchdog{
		Ticker: time.NewTicker(timeout),
	}
}

func NewWatchdogFactory() WatchdogFactory {
	return &watchdogFactory{}
}
