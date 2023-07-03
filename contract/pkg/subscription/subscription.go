package subscription

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/state"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

//go:generate mockgen -destination ../mock/subscription_ChainSubscription.go -package mock github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription ChainSubscription
//go:generate mockgen -destination ../mock/subscription_ChainSubscriptionFactory.go -package mock github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription ChainSubscriptionFactory

type ChainSubscription interface {
	Chan() <-chan types.StorageChangeSet
	Err() <-chan error
	Unsubscribe()
}

type ChainSubscriptionFactory interface {
	NewChainSubscription(sub *state.StorageSubscription) ChainSubscription
}

var _ ChainSubscriptionFactory = &chainSubscriptionFactory{}

type chainSubscriptionFactory struct {
}

func NewChainFactory() ChainSubscriptionFactory {
	return &chainSubscriptionFactory{}
}
func (p *chainSubscriptionFactory) NewChainSubscription(sub *state.StorageSubscription) ChainSubscription {
	return &chainSubscription{sub}
}

var _ ChainSubscription = &chainSubscription{}

type chainSubscription struct {
	sub *state.StorageSubscription
}

func (c *chainSubscription) Chan() <-chan types.StorageChangeSet {
	return c.sub.Chan()
}

func (c *chainSubscription) Err() <-chan error {
	return c.sub.Err()
}

func (c *chainSubscription) Unsubscribe() {
	c.sub.Unsubscribe()
}
