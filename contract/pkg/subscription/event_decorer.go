package subscription

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

//go:generate mockgen -destination ../mock/subscription_EventDecoder.go -package mock github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription EventDecoder

type EventDecoder interface {
	DecodeEvents(data types.StorageDataRaw, m *types.Metadata) (*types.EventRecords, error)
}

func NewEventDecoder() EventDecoder {
	return &eventDecoder{}
}

var _ EventDecoder = &eventDecoder{}

type eventDecoder struct {
}

func (e *eventDecoder) DecodeEvents(data types.StorageDataRaw, m *types.Metadata) (*types.EventRecords, error) {
	events := types.EventRecords{}
	err := types.EventRecordsRaw(data).DecodeEventRecords(m, &events)
	if err != nil {
		return nil, err
	}
	return &events, nil
}
