package pallets

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// Events
type (
	EventDdcStakingBonded struct {
		Phase  types.Phase
		Stash  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}

	EventDdcStakingChilled struct {
		Phase  types.Phase
		Stash  types.AccountID
		Topics []types.Hash
	}

	EventDdcStakingChillSoon struct {
		Phase     types.Phase
		Stash     types.AccountID
		ClusterId ClusterId
		Block     types.BlockNumber
		Topics    []types.Hash
	}

	EventDdcStakingUnbonded struct {
		Phase  types.Phase
		Stash  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}

	EventDdcStakingWithdrawn struct {
		Phase  types.Phase
		Stash  types.AccountID
		Amount types.U128
		Topics []types.Hash
	}

	EventDdcStakingActivated struct {
		Phase  types.Phase
		Stash  types.AccountID
		Topics []types.Hash
	}

	EventDdcStakingLeaveSoon struct {
		Phase  types.Phase
		Stash  types.AccountID
		Topics []types.Hash
	}

	EventDdcStakingLeft struct {
		Phase  types.Phase
		Stash  types.AccountID
		Topics []types.Hash
	}
)
