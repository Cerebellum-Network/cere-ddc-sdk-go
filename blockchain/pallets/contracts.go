package pallets

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// Events
type (
	// These commented-out names are already defined in go-substrate-rpc-client, but some other
	// types are missing.
	// https://github.com/centrifuge/go-substrate-rpc-client/blob/8f01d19/types/event_record.go#L127
	//
	// EventContractsCodeRemoved
	// EventContractsCodeStored
	// EventContractsContractCodeUpdated
	// EventContractsContractEmitted
	// EventContractsInstantiated
	// EventContractsTerminated

	EventContractsCalled struct {
		Phase    types.Phase
		Caller   types.AccountID
		Contract types.AccountID
		Topics   []types.Hash
	}

	EventContractsDelegateCalled struct {
		Phase    types.Phase
		Contract types.AccountID
		CodeHash types.Hash
		Topics   []types.Hash
	}

	EventContractsStorageDepositTransferredAndHeld struct {
		Phase  types.Phase
		From   types.AccountID
		To     types.AccountID
		Amount types.U128
		Topics []types.Hash
	}

	EventContractsStorageDepositTransferredAndReleased struct {
		Phase  types.Phase
		From   types.AccountID
		To     types.AccountID
		Amount types.U128
		Topics []types.Hash
	}
)
