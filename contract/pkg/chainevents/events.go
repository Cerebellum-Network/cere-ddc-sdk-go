package chainevents

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Weight = types.U64

// EventClaimsClaimed is emitted when an account claims some DOTs
type EventClaimsClaimed struct {
	Phase           Phase
	Who             types.AccountID
	EthereumAddress types.H160
	Amount          types.U128
	Topics          []types.Hash
}

// EventBalancesEndowed is emitted when an account is created with some free balance
type EventBalancesEndowed struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventDustLost is emitted when an account is removed with a balance that is
// non-zero but below ExistentialDeposit, resulting in a loss.
type EventBalancesDustLost struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventBalancesTransfer is emitted when a transfer succeeded (from, to, value)
type EventBalancesTransfer struct {
	Phase  Phase
	From   types.AccountID
	To     types.AccountID
	Value  types.U128
	Topics []types.Hash
}

// EventBalanceSet is emitted when a balance is set by root
type EventBalancesBalanceSet struct {
	Phase    Phase
	Who      types.AccountID
	Free     types.U128
	Reserved types.U128
	Topics   []types.Hash
}

// EventDeposit is emitted when an account receives some free balance
type EventBalancesDeposit struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventBalancesReserved is emitted when some balance was reserved (moved from free to reserved)
type EventBalancesReserved struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventBalancesUnreserved is emitted when some balance was unreserved (moved from reserved to free)
type EventBalancesUnreserved struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventBalancesReserveRepatriated is emitted when some balance was moved from the reserve of the first account to the
// second account.
type EventBalancesReserveRepatriated struct {
	Phase             Phase
	From              types.AccountID
	To                types.AccountID
	Balance           types.U128
	DestinationStatus types.BalanceStatus
	Topics            []types.Hash
}

// EventBalancesWithdraw is emitted when some amount was withdrawn from the account (e.g. for transaction fees)
type EventBalancesWithdraw struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventBalancesSlashed is emitted when some amount was removed from the account (e.g. for misbehavior)
type EventBalancesSlashed struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventGrandpaNewAuthorities is emitted when a new authority set has been applied
type EventGrandpaNewAuthorities struct {
	Phase          Phase
	NewAuthorities []struct {
		AuthorityID     types.AuthorityID
		AuthorityWeight types.U64
	}
	Topics []types.Hash
}

// EventGrandpaPaused is emitted when the current authority set has been paused
type EventGrandpaPaused struct {
	Phase  Phase
	Topics []types.Hash
}

// EventGrandpaResumed is emitted when the current authority set has been resumed
type EventGrandpaResumed struct {
	Phase  Phase
	Topics []types.Hash
}

// EventHRMPOpenChannelRequested is emitted when an open HRMP channel is requested.
type EventHRMPOpenChannelRequested struct {
	Phase                  Phase
	Sender                 ParachainID
	Recipient              ParachainID
	ProposedMaxCapacity    types.U32
	ProposedMaxMessageSize types.U32
	Topics                 []types.Hash
}

// EventHRMPOpenChannelCanceled is emitted when an HRMP channel request
// sent by the receiver was canceled by either party.
type EventHRMPOpenChannelCanceled struct {
	Phase       Phase
	ByParachain ParachainID
	ChannelID   types.HRMPChannelID
	Topics      []types.Hash
}

// EventHRMPOpenChannelAccepted is emitted when an open HRMP channel is accepted.
type EventHRMPOpenChannelAccepted struct {
	Phase     Phase
	Sender    ParachainID
	Recipient ParachainID
	Topics    []types.Hash
}

// EventHRMPChannelClosed is emitted when an HRMP channel is closed.
type EventHRMPChannelClosed struct {
	Phase       Phase
	ByParachain ParachainID
	ChannelID   types.HRMPChannelID
	Topics      []types.Hash
}

// EventImOnlineHeartbeatReceived is emitted when a new heartbeat was received from AuthorityId
type EventImOnlineHeartbeatReceived struct {
	Phase       Phase
	AuthorityID types.AuthorityID
	Topics      []types.Hash
}

// EventImOnlineAllGood is emitted when at the end of the session, no offence was committed
type EventImOnlineAllGood struct {
	Phase  Phase
	Topics []types.Hash
}

// Exposure lists the own and nominated stake of a validator
type Exposure struct {
	Total  types.UCompact
	Own    types.UCompact
	Others []IndividualExposure
}

// IndividualExposure contains the nominated stake by one specific third party
type IndividualExposure struct {
	Who   types.AccountID
	Value types.UCompact
}

// EventImOnlineSomeOffline is emitted when the end of the session, at least once validator was found to be offline
type EventImOnlineSomeOffline struct {
	Phase                Phase
	IdentificationTuples []struct {
		ValidatorID        types.AccountID
		FullIdentification Exposure
	}
	Topics []types.Hash
}

// EventIndicesIndexAssigned is emitted when an index is assigned to an AccountID.
type EventIndicesIndexAssigned struct {
	Phase        Phase
	AccountID    types.AccountID
	AccountIndex types.AccountIndex
	Topics       []types.Hash
}

// EventIndicesIndexFreed is emitted when an index is unassigned.
type EventIndicesIndexFreed struct {
	Phase        Phase
	AccountIndex types.AccountIndex
	Topics       []types.Hash
}

// EventIndicesIndexFrozen is emitted when an index is frozen to its current account ID.
type EventIndicesIndexFrozen struct {
	Phase        Phase
	AccountIndex types.AccountIndex
	AccountID    types.AccountID
	Topics       []types.Hash
}

// EventLotteryLotteryStarted is emitted when a lottery has been started.
type EventLotteryLotteryStarted struct {
	Phase  Phase
	Topics []types.Hash
}

// EventLotteryCallsUpdated is emitted when a new set of calls has been set.
type EventLotteryCallsUpdated struct {
	Phase  Phase
	Topics []types.Hash
}

// EventLotteryWinner is emitted when a winner has been chosen.
type EventLotteryWinner struct {
	Phase          Phase
	Winner         types.AccountID
	LotteryBalance types.U128
	Topics         []types.Hash
}

// EventLotteryTicketBought is emitted when a ticket has been bought.
type EventLotteryTicketBought struct {
	Phase     Phase
	Who       types.AccountID
	CallIndex types.LotteryCallIndex
	Topics    []types.Hash
}

// EventOffencesOffence is emitted when there is an offence reported of the given kind happened at the session_index
// and (kind-specific) time slot. This event is not deposited for duplicate slashes
type EventOffencesOffence struct {
	Phase          Phase
	Kind           types.Bytes16
	OpaqueTimeSlot types.Bytes
	Topics         []types.Hash
}

// EventParasCurrentCodeUpdated is emitted when the current code has been updated for a Para.
type EventParasCurrentCodeUpdated struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasCurrentHeadUpdated is emitted when the current head has been updated for a Para.
type EventParasCurrentHeadUpdated struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasCodeUpgradeScheduled is emitted when a code upgrade has been scheduled for a Para.
type EventParasCodeUpgradeScheduled struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasNewHeadNoted is emitted when a new head has been noted for a Para.
type EventParasNewHeadNoted struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasActionQueued is emitted when a para has been queued to execute pending actions.
type EventParasActionQueued struct {
	Phase        Phase
	ParachainID  ParachainID
	SessionIndex types.U32
	Topics       []types.Hash
}

// EventParasPvfCheckStarted is emitted when the given para either initiated or subscribed to a PVF
// check for the given validation code.
type EventParasPvfCheckStarted struct {
	Phase       Phase
	CodeHash    types.Hash
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasPvfCheckAccepted is emitted when the given validation code was accepted by the PVF pre-checking vote.
type EventParasPvfCheckAccepted struct {
	Phase       Phase
	CodeHash    types.Hash
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasPvfCheckRejected is emitted when the given validation code was rejected by the PVF pre-checking vote.
type EventParasPvfCheckRejected struct {
	Phase       Phase
	CodeHash    types.Hash
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventParasDisputesDisputeInitiated is emitted when a dispute has been initiated.
type EventParasDisputesDisputeInitiated struct {
	Phase           Phase
	CandidateHash   types.Hash
	DisputeLocation types.DisputeLocation
	Topics          []types.Hash
}

// EventParasDisputesDisputeConcluded is emitted when a dispute has concluded for or against a candidate.
type EventParasDisputesDisputeConcluded struct {
	Phase           Phase
	CandidateHash   types.Hash
	DisputeLocation types.DisputeResult
	Topics          []types.Hash
}

// EventParasDisputesDisputeTimedOut is emitted when a dispute has timed out due to insufficient participation.
type EventParasDisputesDisputeTimedOut struct {
	Phase         Phase
	CandidateHash types.Hash
	Topics        []types.Hash
}

// EventParasDisputesRevert is emitted when a dispute has concluded with supermajority against a candidate.
// Block authors should no longer build on top of this head and should
// instead revert the block at the given height. This should be the
// number of the child of the last known valid block in the chain.
type EventParasDisputesRevert struct {
	Phase       Phase
	BlockNumber types.U32
	Topics      []types.Hash
}

type HeadData []types.U8

type CoreIndex types.U32

type GroupIndex types.U32

// EventParaInclusionCandidateBacked is emitted when a candidate was backed.
type EventParaInclusionCandidateBacked struct {
	Phase            Phase
	CandidateReceipt types.CandidateReceipt
	HeadData         HeadData
	CoreIndex        CoreIndex
	GroupIndex       GroupIndex
	Topics           []types.Hash
}

// EventParaInclusionCandidateIncluded is emitted when a candidate was included.
type EventParaInclusionCandidateIncluded struct {
	Phase            Phase
	CandidateReceipt types.CandidateReceipt
	HeadData         HeadData
	CoreIndex        CoreIndex
	GroupIndex       GroupIndex
	Topics           []types.Hash
}

// EventParaInclusionCandidateTimedOut is emitted when a candidate timed out.
type EventParaInclusionCandidateTimedOut struct {
	Phase            Phase
	CandidateReceipt types.CandidateReceipt
	HeadData         HeadData
	CoreIndex        CoreIndex
	Topics           []types.Hash
}

// EventParachainSystemValidationFunctionStored is emitted when the validation function has been scheduled to apply.
type EventParachainSystemValidationFunctionStored struct {
	Phase  Phase
	Topics []types.Hash
}

// EventParachainSystemValidationFunctionApplied is emitted when the validation function was applied
// as of the contained relay chain block number.
type EventParachainSystemValidationFunctionApplied struct {
	Phase                 Phase
	RelayChainBlockNumber types.U32
	Topics                []types.Hash
}

// EventParachainSystemValidationFunctionDiscarded is emitted when the relay-chain aborted the upgrade process.
type EventParachainSystemValidationFunctionDiscarded struct {
	Phase  Phase
	Topics []types.Hash
}

// EventParachainSystemUpgradeAuthorized is emitted when an upgrade has been authorized.
type EventParachainSystemUpgradeAuthorized struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventParachainSystemDownwardMessagesReceived is emitted when some downward messages
// have been received and will be processed.
type EventParachainSystemDownwardMessagesReceived struct {
	Phase  Phase
	Count  types.U32
	Topics []types.Hash
}

// EventParachainSystemDownwardMessagesProcessed is emitted when downward messages
// were processed using the given weight.
type EventParachainSystemDownwardMessagesProcessed struct {
	Phase         Phase
	Weight        Weight
	ResultMqcHead types.Hash
	Topics        []types.Hash
}

// EventSessionNewSession is emitted when a new session has happened. Note that the argument is the session index,
// not the block number as the type might suggest
type EventSessionNewSession struct {
	Phase        Phase
	SessionIndex types.U32
	Topics       []types.Hash
}

// EventSlotsNewLeasePeriod is emitted when a new `[lease_period]` is beginning.
type EventSlotsNewLeasePeriod struct {
	Phase       Phase
	LeasePeriod types.U32
	Topics      []types.Hash
}

type ParachainID types.U32

// EventSlotsLeased is emitted when a para has won the right to a continuous set of lease periods as a parachain.
// First balance is any extra amount reserved on top of the para's existing deposit.
// Second balance is the total amount reserved.
type EventSlotsLeased struct {
	Phase         Phase
	ParachainID   ParachainID
	Leaser        types.AccountID
	PeriodBegin   types.U32
	PeriodCount   types.U32
	ExtraReserved types.U128
	TotalAmount   types.U128
	Topics        []types.Hash
}

// EventStakingEraPaid is emitted when the era payout has been set;
type EventStakingEraPaid struct {
	Phase           Phase
	EraIndex        types.U32
	ValidatorPayout types.U128
	Remainder       types.U128
	Topics          []types.Hash
}

// EventStakingRewarded is emitted when the staker has been rewarded by this amount.
type EventStakingRewarded struct {
	Phase  Phase
	Stash  types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventStakingSlashed is emitted when one validator (and its nominators) has been slashed by the given amount
type EventStakingSlashed struct {
	Phase     Phase
	AccountID types.AccountID
	Balance   types.U128
	Topics    []types.Hash
}

// EventStakingOldSlashingReportDiscarded is emitted when an old slashing report from a prior era was discarded because
// it could not be processed
type EventStakingOldSlashingReportDiscarded struct {
	Phase        Phase
	SessionIndex types.U32
	Topics       []types.Hash
}

// EventStakingStakersElected is emitted when a new set of stakers was elected
type EventStakingStakersElected struct {
	Phase  Phase
	Topics []types.Hash
}

// EventStakingStakingElectionFailed is emitted when the election failed. No new era is planned.
type EventStakingStakingElectionFailed struct {
	Phase  Phase
	Topics []types.Hash
}

// EventStakingSolutionStored is emitted when a new solution for the upcoming election has been stored
type EventStakingSolutionStored struct {
	Phase   Phase
	Compute types.ElectionCompute
	Topics  []types.Hash
}

// EventStakingBonded is emitted when an account has bonded this amount
type EventStakingBonded struct {
	Phase  Phase
	Stash  types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventStakingChilled is emitted when an account has stopped participating as either a validator or nominator
type EventStakingChilled struct {
	Phase  Phase
	Stash  types.AccountID
	Topics []types.Hash
}

// EventStakingKicked is emitted when a nominator has been kicked from a validator.
type EventStakingKicked struct {
	Phase     Phase
	Nominator types.AccountID
	Stash     types.AccountID
	Topics    []types.Hash
}

// EventStakingPayoutStarted is emitted when the stakers' rewards are getting paid
type EventStakingPayoutStarted struct {
	Phase    Phase
	EraIndex types.U32
	Stash    types.AccountID
	Topics   []types.Hash
}

// EventStakingUnbonded is emitted when an account has unbonded this amount
type EventStakingUnbonded struct {
	Phase  Phase
	Stash  types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventStakingWithdrawn is emitted when an account has called `withdraw_unbonded` and removed unbonding chunks
// worth `Balance` from the unlocking queue.
type EventStakingWithdrawn struct {
	Phase  Phase
	Stash  types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventStateTrieMigrationMigrated is emitted when the given number of `(top, child)` keys were migrated respectively,
// with the given `compute`.
type EventStateTrieMigrationMigrated struct {
	Phase   Phase
	Top     types.U32
	Child   types.U32
	Compute types.MigrationCompute
	Topics  []types.Hash
}

// EventStateTrieMigrationSlashed is emitted when some account got slashed by the given amount.
type EventStateTrieMigrationSlashed struct {
	Phase  Phase
	Who    types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventStateTrieMigrationAutoMigrationFinished is emitted when the auto migration task has finished.
type EventStateTrieMigrationAutoMigrationFinished struct {
	Phase  Phase
	Topics []types.Hash
}

// EventStateTrieMigrationHalted is emitted when the migration got halted.
type EventStateTrieMigrationHalted struct {
	Phase  Phase
	Topics []types.Hash
}

// EventSystemExtrinsicSuccessV8 is emitted when an extrinsic completed successfully
//
// Deprecated: EventSystemExtrinsicSuccessV8 exists to allow users to simply implement their own EventRecords struct if
// they are on metadata version 8 or below. Use EventSystemExtrinsicSuccess otherwise
type EventSystemExtrinsicSuccessV8 struct {
	Phase  Phase
	Topics []types.Hash
}

// EventSystemExtrinsicSuccess is emitted when an extrinsic completed successfully
type EventSystemExtrinsicSuccess struct {
	Phase        Phase
	DispatchInfo DispatchInfo
	Topics       []types.Hash
}

type Pays struct {
	IsYes bool
	IsNo  bool
}

func (p *Pays) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		p.IsYes = true
	case 1:
		p.IsNo = true
	}

	return nil
}

func (p Pays) Encode(encoder scale.Encoder) error {
	var err error
	if p.IsYes {
		err = encoder.PushByte(0)
	} else if p.IsNo {
		err = encoder.PushByte(1)
	}
	return err
}

// DispatchInfo contains a bundle of static information collected from the `#[weight = $x]` attributes.
type DispatchInfo struct {
	// Weight of this transaction
	Weight Weight
	// Class of this transaction
	Class DispatchClass
	// PaysFee indicates whether this transaction pays fees
	PaysFee Pays
}

func (d *DispatchInfo) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&d.Weight); err != nil {
		return err
	}

	if err := decoder.Decode(&d.Class); err != nil {
		return err
	}

	return decoder.Decode(&d.PaysFee)
}

// DispatchClass is a generalized group of dispatch types. This is only distinguishing normal, user-triggered
// transactions (`Normal`) and anything beyond which serves a higher purpose to the system (`Operational`).
type DispatchClass struct {
	// A normal dispatch
	IsNormal bool
	// An operational dispatch
	IsOperational bool
	// A mandatory dispatch
	IsMandatory bool
}

func (d *DispatchClass) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.IsNormal = true
	case 1:
		d.IsOperational = true
	case 2:
		d.IsMandatory = true
	}

	return nil
}

func (d DispatchClass) Encode(encoder scale.Encoder) error {
	switch {
	case d.IsNormal:
		return encoder.PushByte(0)
	case d.IsOperational:
		return encoder.PushByte(1)
	case d.IsMandatory:
		return encoder.PushByte(2)
	}

	return nil
}

// EventSystemExtrinsicFailedV8 is emitted when an extrinsic failed
//
// Deprecated: EventSystemExtrinsicFailedV8 exists to allow users to simply implement their own EventRecords struct if
// they are on metadata version 8 or below. Use EventSystemExtrinsicFailed otherwise
type EventSystemExtrinsicFailedV8 struct {
	Phase         Phase
	DispatchError types.DispatchError
	Topics        []types.Hash
}

// EventSystemExtrinsicFailed is emitted when an extrinsic failed
type EventSystemExtrinsicFailed struct {
	Phase         Phase
	DispatchError types.DispatchError
	DispatchInfo  DispatchInfo
	Topics        []types.Hash
}

// EventSystemCodeUpdated is emitted when the runtime code (`:code`) is updated
type EventSystemCodeUpdated struct {
	Phase  Phase
	Topics []types.Hash
}

// EventSystemNewAccount is emitted when a new account was created
type EventSystemNewAccount struct {
	Phase  Phase
	Who    types.AccountID
	Topics []types.Hash
}

// EventSystemRemarked is emitted when an on-chain remark happened
type EventSystemRemarked struct {
	Phase  Phase
	Who    types.AccountID
	Hash   types.Hash
	Topics []types.Hash
}

// EventSystemKilledAccount is emitted when an account is reaped
type EventSystemKilledAccount struct {
	Phase  Phase
	Who    types.AccountID
	Topics []types.Hash
}

// EventAssetIssued is emitted when an asset is issued.
type EventAssetIssued struct {
	Phase   Phase
	AssetID types.U32
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventAssetCreated is emitted when an asset is created.
type EventAssetCreated struct {
	Phase   Phase
	AssetID types.U32
	Creator types.AccountID
	Owner   types.AccountID
	Topics  []types.Hash
}

// EventAssetTransferred is emitted when an asset is transferred.
type EventAssetTransferred struct {
	Phase   Phase
	AssetID types.U32
	To      types.AccountID
	From    types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventAssetBurned is emitted when an asset is destroyed.
type EventAssetBurned struct {
	Phase   Phase
	AssetID types.U32
	Owner   types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventAssetTeamChanged is emitted when the management team changed.
type EventAssetTeamChanged struct {
	Phase   Phase
	AssetID types.U32
	Issuer  types.AccountID
	Admin   types.AccountID
	Freezer types.AccountID
	Topics  []types.Hash
}

// EventAssetOwnerChanged is emitted when the owner changed.
type EventAssetOwnerChanged struct {
	Phase   Phase
	AssetID types.U32
	Owner   types.AccountID
	Topics  []types.Hash
}

// EventAssetFrozen is emitted when some account `who` was frozen.
type EventAssetFrozen struct {
	Phase   Phase
	AssetID types.U32
	Who     types.AccountID
	Topics  []types.Hash
}

// EventAssetThawed is emitted when some account `who` was thawed.
type EventAssetThawed struct {
	Phase   Phase
	AssetID types.U32
	Who     types.AccountID
	Topics  []types.Hash
}

// EventAssetAssetFrozen is emitted when some asset `asset_id` was frozen.
type EventAssetAssetFrozen struct {
	Phase   Phase
	AssetID types.U32
	Topics  []types.Hash
}

// EventAssetAssetThawed is emitted when some asset `asset_id` was thawed.
type EventAssetAssetThawed struct {
	Phase   Phase
	AssetID types.U32
	Topics  []types.Hash
}

// EventAssetDestroyed is emitted when an asset class is destroyed.
type EventAssetDestroyed struct {
	Phase   Phase
	AssetID types.U32
	Topics  []types.Hash
}

// EventAssetForceCreated is emitted when some asset class was force-created.
type EventAssetForceCreated struct {
	Phase   Phase
	AssetID types.U32
	Owner   types.AccountID
	Topics  []types.Hash
}

type MetadataSetName []byte
type MetadataSetSymbol []byte

// EventAssetMetadataSet is emitted when new metadata has been set for an asset.
type EventAssetMetadataSet struct {
	Phase    Phase
	AssetID  types.U32
	Name     MetadataSetName
	Symbol   MetadataSetSymbol
	Decimals types.U8
	IsFrozen bool
	Topics   []types.Hash
}

// EventAssetMetadataCleared is emitted when metadata has been cleared for an asset.
type EventAssetMetadataCleared struct {
	Phase   Phase
	AssetID types.U32
	Topics  []types.Hash
}

// EventAssetApprovedTransfer is emitted when (additional) funds have been approved
// for transfer to a destination account.
type EventAssetApprovedTransfer struct {
	Phase    Phase
	AssetID  types.U32
	Source   types.AccountID
	Delegate types.AccountID
	Amount   types.U128
	Topics   []types.Hash
}

// EventAssetApprovalCancelled is emitted when an approval for account `delegate` was cancelled by `owner`.
type EventAssetApprovalCancelled struct {
	Phase    Phase
	AssetID  types.U32
	Owner    types.AccountID
	Delegate types.AccountID
	Topics   []types.Hash
}

// EventAssetTransferredApproved is emitted when an `amount` was transferred in its
// entirety from `owner` to `destination` by the approved `delegate`.
type EventAssetTransferredApproved struct {
	Phase       Phase
	AssetID     types.U32
	Owner       types.AccountID
	Delegate    types.AccountID
	Destination types.AccountID
	Amount      types.U128
	Topics      []types.Hash
}

// EventAssetAssetStatusChanged is emitted when an asset has had its attributes changed by the `Force` origin.
type EventAssetAssetStatusChanged struct {
	Phase   Phase
	AssetID types.U32
	Topics  []types.Hash
}

// EventAuctionsAuctionStarted is emitted when an auction started. Provides its index and the block number
// where it will begin to close and the first lease period of the quadruplet that is auctioned.
type EventAuctionsAuctionStarted struct {
	Phase        Phase
	AuctionIndex types.U32
	LeasePeriod  types.U32
	Ending       types.U32
	Topics       []types.Hash
}

// EventAuctionsAuctionClosed is emitted when an auction ended. All funds become unreserved.
type EventAuctionsAuctionClosed struct {
	Phase        Phase
	AuctionIndex types.U32
	Topics       []types.Hash
}

// EventAuctionsReserved is emitted when funds were reserved for a winning bid.
// First balance is the extra amount reserved. Second is the total.
type EventAuctionsReserved struct {
	Phase         Phase
	Bidder        types.AccountID
	ExtraReserved types.U128
	TotalAmount   types.U128
	Topics        []types.Hash
}

// EventAuctionsUnreserved is emitted when funds were unreserved since bidder is no longer active.
type EventAuctionsUnreserved struct {
	Phase  Phase
	Bidder types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventAuctionsReserveConfiscated is emitted when someone attempted to lease the same slot twice for a parachain.
// The amount is held in reserve but no parachain slot has been leased.
type EventAuctionsReserveConfiscated struct {
	Phase       Phase
	ParachainID ParachainID
	Leaser      types.AccountID
	Amount      types.U128
	Topics      []types.Hash
}

// EventAuctionsBidAccepted is emitted when a new bid has been accepted as the current winner.
type EventAuctionsBidAccepted struct {
	Phase       Phase
	Who         types.AccountID
	ParachainID ParachainID
	Amount      types.U128
	FirstSlot   types.U32
	LastSlot    types.U32
	Topics      []types.Hash
}

// EventAuctionsWinningOffset is emitted when the winning offset was chosen for an auction.
// This will map into the `Winning` storage map.
type EventAuctionsWinningOffset struct {
	Phase        Phase
	AuctionIndex types.U32
	BlockNumber  types.U32
	Topics       []types.Hash
}

// EventBagsListRebagged is emitted when an account was moved from one bag to another.
type EventBagsListRebagged struct {
	Phase  Phase
	Who    types.AccountID
	From   types.U64
	To     types.U64
	Topics []types.Hash
}

// EventDemocracyProposed is emitted when a motion has been proposed by a public account.
type EventDemocracyProposed struct {
	Phase         Phase
	ProposalIndex types.U32
	Balance       types.U128
	Topics        []types.Hash
}

// EventDemocracyTabled is emitted when a public proposal has been tabled for referendum vote.
type EventDemocracyTabled struct {
	Phase         Phase
	ProposalIndex types.U32
	Balance       types.U128
	Accounts      []types.AccountID
	Topics        []types.Hash
}

// EventDemocracyExternalTabled is emitted when an external proposal has been tabled.
type EventDemocracyExternalTabled struct {
	Phase  Phase
	Topics []types.Hash
}

// VoteThreshold is a means of determining if a vote is past pass threshold.
type VoteThreshold byte

const (
	// SuperMajorityApprove require super majority of approvals is needed to pass this vote.
	SuperMajorityApprove VoteThreshold = 0
	// SuperMajorityAgainst require super majority of rejects is needed to fail this vote.
	SuperMajorityAgainst VoteThreshold = 1
	// SimpleMajority require simple majority of approvals is needed to pass this vote.
	SimpleMajority VoteThreshold = 2
)

func (v *VoteThreshold) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := VoteThreshold(b)
	switch vb {
	case SuperMajorityApprove, SuperMajorityAgainst, SimpleMajority:
		*v = vb
	default:
		return fmt.Errorf("unknown VoteThreshold enum: %v", vb)
	}
	return err
}

func (v VoteThreshold) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(v))
}

type DemocracyConviction byte

const (
	// None 0.1x votes, unlocked
	None = 0
	// Locked1x votes, locked for an enactment period following a successful vote.
	Locked1x = 1
	// Locked2x votes, locked for 2x enactment periods following a successful vote.
	Locked2x = 2
	// Locked3x votes, locked for 4x...
	Locked3x = 3
	// Locked4x votes, locked for 8x...
	Locked4x = 4
	// Locked5x votes, locked for 16x...
	Locked5x = 5
	// Locked6x votes, locked for 32x...
	Locked6x = 6
)

func (dc *DemocracyConviction) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := DemocracyConviction(b)
	switch vb {
	case None, Locked1x, Locked2x, Locked3x, Locked4x, Locked5x, Locked6x:
		*dc = vb
	default:
		return fmt.Errorf("unknown DemocracyConviction enum: %v", vb)
	}
	return err
}

func (dc DemocracyConviction) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(dc))
}

type DemocracyVote struct {
	Aye        bool
	Conviction DemocracyConviction
}

const (
	aye uint8 = 1 << 7
)

//nolint:lll
func (d *DemocracyVote) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	// As per:
	// https://github.com/paritytech/substrate/blob/6a946fc36d68b89599d7ca1ab03803d10c78468c/frame/democracy/src/vote.rs#L44

	d.Aye = (b & aye) == aye
	d.Conviction = DemocracyConviction(b & (aye - 1))

	return nil
}

//nolint:lll
func (d DemocracyVote) Encode(encoder scale.Encoder) error {
	// As per:
	// https://github.com/paritytech/substrate/blob/6a946fc36d68b89599d7ca1ab03803d10c78468c/frame/democracy/src/vote.rs#L37

	var val uint8

	if d.Aye {
		val = aye
	}

	return encoder.PushByte(uint8(d.Conviction) | val)
}

type VoteAccountVoteAsStandard struct {
	Vote    DemocracyVote
	Balance types.U128
}

func (v *VoteAccountVoteAsStandard) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&v.Vote); err != nil {
		return err
	}

	return decoder.Decode(&v.Balance)
}

func (v VoteAccountVoteAsStandard) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(v.Vote); err != nil {
		return err
	}

	return encoder.Encode(v.Balance)
}

type VoteAccountVoteAsSplit struct {
	Aye types.U128
	Nay types.U128
}

type VoteAccountVote struct {
	IsStandard bool
	AsStandard VoteAccountVoteAsStandard
	IsSplit    bool
	AsSplit    VoteAccountVoteAsSplit
}

func (vv *VoteAccountVote) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		vv.IsStandard = true

		return decoder.Decode(&vv.AsStandard)
	case 1:
		vv.IsSplit = true

		return decoder.Decode(&vv.AsSplit)
	}

	return nil
}

func (vv VoteAccountVote) Encode(encoder scale.Encoder) error {
	switch {
	case vv.IsStandard:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(vv.AsStandard)
	case vv.IsSplit:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(vv.AsSplit)
	}

	return nil
}

// EventDemocracyStarted is emitted when a referendum has begun.
type EventDemocracyStarted struct {
	Phase           Phase
	ReferendumIndex types.U32
	VoteThreshold   VoteThreshold
	Topics          []types.Hash
}

// EventDemocracyPassed is emitted when a proposal has been approved by referendum.
type EventDemocracyPassed struct {
	Phase           Phase
	ReferendumIndex types.U32
	Topics          []types.Hash
}

// EventDemocracyNotPassed is emitted when a proposal has been rejected by referendum.
type EventDemocracyNotPassed struct {
	Phase           Phase
	ReferendumIndex types.U32
	Topics          []types.Hash
}

// EventDemocracyCancelled is emitted when a referendum has been cancelled.
type EventDemocracyCancelled struct {
	Phase           Phase
	ReferendumIndex types.U32
	Topics          []types.Hash
}

// EventDemocracyExecuted is emitted when a proposal has been enacted.
type EventDemocracyExecuted struct {
	Phase           Phase
	ReferendumIndex types.U32
	Result          DispatchResult
	Topics          []types.Hash
}

// EventDemocracyDelegated is emitted when an account has delegated their vote to another account.
type EventDemocracyDelegated struct {
	Phase  Phase
	Who    types.AccountID
	Target types.AccountID
	Topics []types.Hash
}

// EventDemocracyUndelegated is emitted when an account has cancelled a previous delegation operation.
type EventDemocracyUndelegated struct {
	Phase  Phase
	Target types.AccountID
	Topics []types.Hash
}

// EventDemocracyVetoed is emitted when an external proposal has been vetoed.
type EventDemocracyVetoed struct {
	Phase       Phase
	Who         types.AccountID
	Hash        types.Hash
	BlockNumber types.U32
	Topics      []types.Hash
}

// EventDemocracyVoted is emitted when an account has voted in a referendum.
type EventDemocracyVoted struct {
	Phase           Phase
	Who             types.AccountID
	ReferendumIndex types.U32
	Vote            VoteAccountVote
	Topics          []types.Hash
}

// EventElectionProviderMultiPhaseSolutionStored is emitted when a solution was stored with the given compute.
//
// If the solution is signed, this means that it hasn't yet been processed. If the
// solution is unsigned, this means that it has also been processed.
//
// The `bool` is `true` when a previous solution was ejected to make room for this one.
type EventElectionProviderMultiPhaseSolutionStored struct {
	Phase           Phase
	ElectionCompute types.ElectionCompute
	PrevEjected     bool
	Topics          []types.Hash
}

// EventElectionProviderMultiPhaseElectionFinalized is emitted when the election has been finalized,
// with `Some` of the given computation, or else if the election failed, `None`.
type EventElectionProviderMultiPhaseElectionFinalized struct {
	Phase           Phase
	ElectionCompute types.OptionElectionCompute
	Topics          []types.Hash
}

// EventElectionProviderMultiPhaseRewarded is emitted when an account has been rewarded for their
// signed submission being finalized.
type EventElectionProviderMultiPhaseRewarded struct {
	Phase   Phase
	Account types.AccountID
	Value   types.U128
	Topics  []types.Hash
}

// EventElectionProviderMultiPhaseSlashed is emitted when an account has been slashed for
// submitting an invalid signed submission.
type EventElectionProviderMultiPhaseSlashed struct {
	Phase   Phase
	Account types.AccountID
	Value   types.U128
	Topics  []types.Hash
}

// EventElectionProviderMultiPhaseSignedPhaseStarted is emitted when the signed phase of the given round has started.
type EventElectionProviderMultiPhaseSignedPhaseStarted struct {
	Phase  Phase
	Round  types.U32
	Topics []types.Hash
}

// EventElectionProviderMultiPhaseUnsignedPhaseStarted is emitted when the unsigned phase of
// the given round has started.
type EventElectionProviderMultiPhaseUnsignedPhaseStarted struct {
	Phase  Phase
	Round  types.U32
	Topics []types.Hash
}

// EventDemocracyPreimageNoted is emitted when a proposal's preimage was noted, and the deposit taken.
type EventDemocracyPreimageNoted struct {
	Phase     Phase
	Hash      types.Hash
	AccountID types.AccountID
	Balance   types.U128
	Topics    []types.Hash
}

// EventDemocracyPreimageUsed is emitted when a proposal preimage was removed and used (the deposit was returned).
type EventDemocracyPreimageUsed struct {
	Phase     Phase
	Hash      types.Hash
	AccountID types.AccountID
	Balance   types.U128
	Topics    []types.Hash
}

// EventDemocracyPreimageInvalid is emitted when a proposal could not be executed because its preimage was invalid.
type EventDemocracyPreimageInvalid struct {
	Phase           Phase
	Hash            types.Hash
	ReferendumIndex types.U32
	Topics          []types.Hash
}

// EventDemocracyPreimageMissing is emitted when a proposal could not be executed because its preimage was missing.
type EventDemocracyPreimageMissing struct {
	Phase           Phase
	Hash            types.Hash
	ReferendumIndex types.U32
	Topics          []types.Hash
}

// EventDemocracyPreimageReaped is emitted when a registered preimage was removed
// and the deposit collected by the reaper (last item).
type EventDemocracyPreimageReaped struct {
	Phase    Phase
	Hash     types.Hash
	Provider types.AccountID
	Balance  types.U128
	Who      types.AccountID
	Topics   []types.Hash
}

// EventDemocracySeconded is emitted when an account has seconded a proposal.
type EventDemocracySeconded struct {
	Phase     Phase
	AccountID types.AccountID
	Balance   types.U128
	Topics    []types.Hash
}

// EventDemocracyBlacklisted is emitted when A proposal has been blacklisted permanently
type EventDemocracyBlacklisted struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventCouncilProposed is emitted when a motion (given hash) has been proposed (by given account)
// with a threshold (given `MemberCount`).
type EventCouncilProposed struct {
	Phase         Phase
	Who           types.AccountID
	ProposalIndex types.U32
	Proposal      types.Hash
	MemberCount   types.U32
	Topics        []types.Hash
}

// EventCollectiveVote is emitted when a motion (given hash) has been voted on by given account, leaving
// a tally (yes votes and no votes given respectively as `MemberCount`).
type EventCouncilVoted struct {
	Phase    Phase
	Who      types.AccountID
	Proposal types.Hash
	Approve  bool
	YesCount types.U32
	NoCount  types.U32
	Topics   []types.Hash
}

// EventCrowdloanCreated is emitted when a new crowdloaning campaign is created.
type EventCrowdloanCreated struct {
	Phase     Phase
	FundIndex types.U32
	Topics    []types.Hash
}

// EventCrowdloanContributed is emitted when `who` contributed to a crowd sale.
type EventCrowdloanContributed struct {
	Phase     Phase
	Who       types.AccountID
	FundIndex types.U32
	Amount    types.U128
	Topics    []types.Hash
}

// EventCrowdloanWithdrew is emitted when the full balance of a contributor was withdrawn.
type EventCrowdloanWithdrew struct {
	Phase     Phase
	Who       types.AccountID
	FundIndex types.U32
	Amount    types.U128
	Topics    []types.Hash
}

// EventCrowdloanPartiallyRefunded is emitted when the loans in a fund have been partially dissolved, i.e.
// there are some left over child keys that still need to be killed.
type EventCrowdloanPartiallyRefunded struct {
	Phase     Phase
	FundIndex types.U32
	Topics    []types.Hash
}

// EventCrowdloanAllRefunded is emitted when all loans in a fund have been refunded.
type EventCrowdloanAllRefunded struct {
	Phase     Phase
	FundIndex types.U32
	Topics    []types.Hash
}

// EventCrowdloanDissolved is emitted when the fund is dissolved.
type EventCrowdloanDissolved struct {
	Phase     Phase
	FundIndex types.U32
	Topics    []types.Hash
}

// EventCrowdloanHandleBidResult is emitted when trying to submit a new bid to the Slots pallet.
type EventCrowdloanHandleBidResult struct {
	Phase          Phase
	FundIndex      types.U32
	DispatchResult DispatchResult
	Topics         []types.Hash
}

// EventCrowdloanEdited is emitted when the configuration to a crowdloan has been edited.
type EventCrowdloanEdited struct {
	Phase     Phase
	FundIndex types.U32
	Topics    []types.Hash
}

type CrowloadMemo []byte

// EventCrowdloanMemoUpdated is emitted when a memo has been updated.
type EventCrowdloanMemoUpdated struct {
	Phase     Phase
	Who       types.AccountID
	FundIndex types.U32
	Memo      CrowloadMemo
	Topics    []types.Hash
}

// EventCrowdloanAddedToNewRaise is emitted when a parachain has been moved to `NewRaise`.
type EventCrowdloanAddedToNewRaise struct {
	Phase     Phase
	FundIndex types.U32
	Topics    []types.Hash
}

// EventCouncilAptypes.proved is emitted when a motion was approved by the required threshold.
type EventCouncilApproved struct {
	Phase    Phase
	Proposal types.Hash
	Topics   []types.Hash
}

// EventCouncilDisapproved is emitted when a motion was not approved by the required threshold.
type EventCouncilDisapproved struct {
	Phase    Phase
	Proposal types.Hash
	Topics   []types.Hash
}

// EventCouncilExecuted is emitted when a motion was executed; `result` is true if returned without error.
type EventCouncilExecuted struct {
	Phase    Phase
	Proposal types.Hash
	Result   DispatchResult
	Topics   []types.Hash
}

// EventCouncilMemberExecuted is emitted when a single member did some action;
// `result` is true if returned without error.
type EventCouncilMemberExecuted struct {
	Phase    Phase
	Proposal types.Hash
	Result   DispatchResult
	Topics   []types.Hash
}

// EventCouncilClosed is emitted when a proposal was closed after its duration was up.
type EventCouncilClosed struct {
	Phase    Phase
	Proposal types.Hash
	YesCount types.U32
	NoCount  types.U32
	Topics   []types.Hash
}

// EventTechnicalCommitteeProposed is emitted when a motion (given hash) has been proposed (by given account)
// with a threshold (given, `MemberCount`)
type EventTechnicalCommitteeProposed struct {
	Phase         Phase
	Account       types.AccountID
	ProposalIndex types.U32
	Proposal      types.Hash
	Threshold     types.U32
	Topics        []types.Hash
}

// EventTechnicalCommitteeVoted is emitted when a motion (given hash) has been voted on by given account, leaving,
// a tally (yes votes and no votes given respectively as `MemberCount`).
type EventTechnicalCommitteeVoted struct {
	Phase    Phase
	Account  types.AccountID
	Proposal types.Hash
	Voted    bool
	YesCount types.U32
	NoCount  types.U32
	Topics   []types.Hash
}

// EventTechnicalCommitteeApproved is emitted when a motion was approved by the required threshold.
type EventTechnicalCommitteeApproved struct {
	Phase    Phase
	Proposal types.Hash
	Topics   []types.Hash
}

// EventTechnicalCommitteeDisapproved is emitted when a motion was not approved by the required threshold.
type EventTechnicalCommitteeDisapproved struct {
	Phase    Phase
	Proposal types.Hash
	Topics   []types.Hash
}

// EventTechnicalCommitteeExecuted is emitted when a motion was executed;
// result will be `Ok` if it returned without error.
type EventTechnicalCommitteeExecuted struct {
	Phase    Phase
	Proposal types.Hash
	Result   DispatchResult
	Topics   []types.Hash
}

// EventTechnicalCommitteeMemberExecuted is emitted when a single member did some action;
// result will be `Ok` if it returned without error
type EventTechnicalCommitteeMemberExecuted struct {
	Phase    Phase
	Proposal types.Hash
	Result   DispatchResult
	Topics   []types.Hash
}

// EventTechnicalCommitteeClosed is emitted when A proposal was closed because its threshold was reached
// or after its duration was up
type EventTechnicalCommitteeClosed struct {
	Phase    Phase
	Proposal types.Hash
	YesCount types.U32
	NoCount  types.U32
	Topics   []types.Hash
}

// EventTechnicalMembershipMemberAdded is emitted when the given member was added; see the transaction for who
type EventTechnicalMembershipMemberAdded struct {
	Phase  Phase
	Topics []types.Hash
}

// EventTechnicalMembershipMemberRemoved is emitted when the given member was removed; see the transaction for who
type EventTechnicalMembershipMemberRemoved struct {
	Phase  Phase
	Topics []types.Hash
}

// EventTechnicalMembershipMembersSwapped is emitted when two members were swapped;; see the transaction for who
type EventTechnicalMembershipMembersSwapped struct {
	Phase  Phase
	Topics []types.Hash
}

// EventTechnicalMembershipMembersReset is emitted when the membership was reset;
// see the transaction for who the new set is.
type EventTechnicalMembershipMembersReset struct {
	Phase  Phase
	Topics []types.Hash
}

// EventTechnicalMembershipKeyChanged is emitted when one of the members' keys changed.
type EventTechnicalMembershipKeyChanged struct {
	Phase  Phase
	Topics []types.Hash
}

// EventTechnicalMembershipKeyChanged is emitted when - phantom member, never used.
type EventTechnicalMembershipDummy struct {
	Phase  Phase
	Topics []types.Hash
}

// EventElectionsNewTerm is emitted when a new term with new members.
// This indicates that enough candidates existed, not that enough have has been elected.
// The inner value must be examined for this purpose.
type EventElectionsNewTerm struct {
	Phase      Phase
	NewMembers []struct {
		Member  types.AccountID
		Balance types.U128
	}
	Topics []types.Hash
}

// EventElectionsCandidateSlashed is emitted when a candidate was slashed by amount due to failing to obtain a seat
// as member or runner-up. Note that old members and runners-up are also candidates.
type EventElectionsCandidateSlashed struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventElectionsEmptyTerm is emitted when No (or not enough) candidates existed for this round.
type EventElectionsEmptyTerm struct {
	Phase  Phase
	Topics []types.Hash
}

// EventElectionsElectionError is emitted when an internal error happened while trying to perform election
type EventElectionsElectionError struct {
	Phase  Phase
	Topics []types.Hash
}

// EventElectionsMemberKicked is emitted when a member has been removed.
// This should always be followed by either `NewTerm` or `EmptyTerm`.
type EventElectionsMemberKicked struct {
	Phase  Phase
	Member types.AccountID
	Topics []types.Hash
}

// EventElectionsRenounced is emitted when a member has renounced their candidacy.
type EventElectionsRenounced struct {
	Phase  Phase
	Member types.AccountID
	Topics []types.Hash
}

// EventElectionsSeatHolderSlashed is emitted when a seat holder was slashed by amount
// by being forcefully removed from the set
type EventElectionsSeatHolderSlashed struct {
	Phase   Phase
	Who     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

// EventGiltBidPlaced is emitted when a bid was successfully placed.
type EventGiltBidPlaced struct {
	Phase    Phase
	Who      types.AccountID
	Amount   types.U128
	Duration types.U32
	Topics   []types.Hash
}

// EventGiltBidRetracted is emitted when a bid was successfully removed (before being accepted as a gilt).
type EventGiltBidRetracted struct {
	Phase    Phase
	Who      types.AccountID
	Amount   types.U128
	Duration types.U32
	Topics   []types.Hash
}

// EventGiltGiltIssued is emitted when a bid was accepted as a gilt. The balance may not be released until expiry.
type EventGiltGiltIssued struct {
	Phase  Phase
	Index  types.U32
	Expiry types.U32
	Who    types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventGiltGiltThawed is emitted when an expired gilt has been thawed.
type EventGiltGiltThawed struct {
	Phase            Phase
	Index            types.U32
	Who              types.AccountID
	OriginalAmount   types.U128
	AdditionalAmount types.U128
	Topics           []types.Hash
}

// A name was set or reset (which will remove all judgements).
type EventIdentitySet struct {
	Phase    Phase
	Identity types.AccountID
	Topics   []types.Hash
}

// A name was cleared, and the given balance returned.
type EventIdentityCleared struct {
	Phase    Phase
	Identity types.AccountID
	Balance  types.U128
	Topics   []types.Hash
}

// A name was removed and the given balance slashed.
type EventIdentityKilled struct {
	Phase    Phase
	Identity types.AccountID
	Balance  types.U128
	Topics   []types.Hash
}

// A judgement was asked from a registrar.
type EventIdentityJudgementRequested struct {
	Phase          Phase
	Sender         types.AccountID
	RegistrarIndex types.U32
	Topics         []types.Hash
}

// A judgement request was retracted.
type EventIdentityJudgementUnrequested struct {
	Phase          Phase
	Sender         types.AccountID
	RegistrarIndex types.U32
	Topics         []types.Hash
}

// A judgement was given by a registrar.
type EventIdentityJudgementGiven struct {
	Phase          Phase
	Target         types.AccountID
	RegistrarIndex types.U32
	Topics         []types.Hash
}

// A registrar was added.
type EventIdentityRegistrarAdded struct {
	Phase          Phase
	RegistrarIndex types.U32
	Topics         []types.Hash
}

// EventIdentitySubIdentityAdded is emitted when a sub-identity was added to an identity and the deposit paid
type EventIdentitySubIdentityAdded struct {
	Phase   Phase
	Sub     types.AccountID
	Main    types.AccountID
	Deposit types.U128
	Topics  []types.Hash
}

// EventIdentitySubIdentityRemoved is emitted when a sub-identity was removed from an identity and the deposit freed
type EventIdentitySubIdentityRemoved struct {
	Phase   Phase
	Sub     types.AccountID
	Main    types.AccountID
	Deposit types.U128
	Topics  []types.Hash
}

// EventIdentitySubIdentityRevoked is emitted when a sub-identity was cleared, and the given deposit repatriated from
// the main identity account to the sub-identity account.
type EventIdentitySubIdentityRevoked struct {
	Phase   Phase
	Sub     types.AccountID
	Main    types.AccountID
	Deposit types.U128
	Topics  []types.Hash
}

// EventSocietyFounded is emitted when the society is founded by the given identity
type EventSocietyFounded struct {
	Phase   Phase
	Founder types.AccountID
	Topics  []types.Hash
}

// EventSocietyBid is emitted when a membership bid just happened. The given account is the candidate's ID
// and their offer is the second
type EventSocietyBid struct {
	Phase     Phase
	Candidate types.AccountID
	Offer     types.U128
	Topics    []types.Hash
}

// EventSocietyVouch is emitted when a membership bid just happened by vouching.
// The given account is the candidate's ID and, their offer is the second. The vouching party is the third.
type EventSocietyVouch struct {
	Phase     Phase
	Candidate types.AccountID
	Offer     types.U128
	Vouching  types.AccountID
	Topics    []types.Hash
}

// EventSocietyAutoUnbid is emitted when a [candidate] was dropped (due to an excess of bids in the system)
type EventSocietyAutoUnbid struct {
	Phase     Phase
	Candidate types.AccountID
	Topics    []types.Hash
}

// EventSocietyUnbid is emitted when a [candidate] was dropped (by their request)
type EventSocietyUnbid struct {
	Phase     Phase
	Candidate types.AccountID
	Topics    []types.Hash
}

// EventSocietyUnvouch is emitted when a [candidate] was dropped (by request of who vouched for them)
type EventSocietyUnvouch struct {
	Phase     Phase
	Candidate types.AccountID
	Topics    []types.Hash
}

// EventSocietyInducted is emitted when a group of candidates have been inducted.
// The batch's primary is the first value, the batch in full is the second.
type EventSocietyInducted struct {
	Phase      Phase
	Primary    types.AccountID
	Candidates []types.AccountID
	Topics     []types.Hash
}

// EventSocietySuspendedMemberJudgement is emitted when a suspended member has been judged
type EventSocietySuspendedMemberJudgement struct {
	Phase  Phase
	Who    types.AccountID
	Judged bool
	Topics []types.Hash
}

// EventSocietyCandidateSuspended is emitted when a [candidate] has been suspended
type EventSocietyCandidateSuspended struct {
	Phase     Phase
	Candidate types.AccountID
	Topics    []types.Hash
}

// EventSocietyMemberSuspended is emitted when a [member] has been suspended
type EventSocietyMemberSuspended struct {
	Phase  Phase
	Member types.AccountID
	Topics []types.Hash
}

// EventSocietyChallenged is emitted when a [member] has been challenged
type EventSocietyChallenged struct {
	Phase  Phase
	Member types.AccountID
	Topics []types.Hash
}

// EventSocietyVote is emitted when a vote has been placed
type EventSocietyVote struct {
	Phase     Phase
	Candidate types.AccountID
	Voter     types.AccountID
	Vote      bool
	Topics    []types.Hash
}

// EventSocietyDefenderVote is emitted when a vote has been placed for a defending member
type EventSocietyDefenderVote struct {
	Phase  Phase
	Voter  types.AccountID
	Vote   bool
	Topics []types.Hash
}

// EventSocietyNewMaxMembers is emitted when a new [max] member count has been set
type EventSocietyNewMaxMembers struct {
	Phase  Phase
	Max    types.U32
	Topics []types.Hash
}

// EventSocietyUnfounded is emitted when society is unfounded
type EventSocietyUnfounded struct {
	Phase   Phase
	Founder types.AccountID
	Topics  []types.Hash
}

// EventSocietyDeposit is emitted when some funds were deposited into the society account
type EventSocietyDeposit struct {
	Phase  Phase
	Value  types.U128
	Topics []types.Hash
}

// EventRecoveryCreated is emitted when a recovery process has been set up for an account
type EventRecoveryCreated struct {
	Phase  Phase
	Who    types.AccountID
	Topics []types.Hash
}

// EventRecoveryInitiated is emitted when a recovery process has been initiated for account_1 by account_2
type EventRecoveryInitiated struct {
	Phase   Phase
	Account types.AccountID
	Who     types.AccountID
	Topics  []types.Hash
}

// EventRecoveryVouched is emitted when a recovery process for account_1 by account_2 has been vouched for by account_3
type EventRecoveryVouched struct {
	Phase   Phase
	Lost    types.AccountID
	Rescuer types.AccountID
	Who     types.AccountID
	Topics  []types.Hash
}

// EventRegistrarRegistered is emitted when a parachain is registered.
type EventRegistrarRegistered struct {
	Phase       Phase
	ParachainID ParachainID
	Account     types.AccountID
	Topics      []types.Hash
}

// EventRegistrarDeregistered is emitted when a parachain is deregistered.
type EventRegistrarDeregistered struct {
	Phase       Phase
	ParachainID ParachainID
	Topics      []types.Hash
}

// EventRegistrarReserved is emitted when a parachain slot is reserved.
type EventRegistrarReserved struct {
	Phase       Phase
	ParachainID ParachainID
	Account     types.AccountID
	Topics      []types.Hash
}

// EventReferendaSubmitted is emitted when a referendum has been submitted.
type EventReferendaSubmitted struct {
	Phase        Phase
	Index        types.U32
	Track        types.U8
	ProposalHash types.Hash
	Topics       []types.Hash
}

// EventReferendaDecisionDepositPlaced is emitted when the decision deposit has been placed.
type EventReferendaDecisionDepositPlaced struct {
	Phase  Phase
	Index  types.U32
	Who    types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventReferendaDecisionDepositRefunded is emitted when the decision deposit has been refunded.
type EventReferendaDecisionDepositRefunded struct {
	Phase  Phase
	Index  types.U32
	Who    types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventReferendaDecisionSlashed is emitted when a deposit has been slashed.
type EventReferendaDecisionSlashed struct {
	Phase  Phase
	Who    types.AccountID
	Amount types.U128
	Topics []types.Hash
}

// EventReferendaDecisionStarted is emitted when a referendum has moved into the deciding phase.
type EventReferendaDecisionStarted struct {
	Phase        Phase
	Index        types.U32
	Track        types.U8
	ProposalHash types.Hash
	Tally        types.Tally
	Topics       []types.Hash
}

// EventReferendaConfirmStarted is emitted when a referendum has been started.
type EventReferendaConfirmStarted struct {
	Phase  Phase
	Index  types.U32
	Topics []types.Hash
}

// EventReferendaConfirmAborted is emitted when a referendum has been aborted.
type EventReferendaConfirmAborted struct {
	Phase  Phase
	Index  types.U32
	Topics []types.Hash
}

// EventReferendaConfirmed is emitted when a referendum has ended its confirmation phase and is ready for approval.
type EventReferendaConfirmed struct {
	Phase  Phase
	Index  types.U32
	Tally  types.Tally
	Topics []types.Hash
}

// EventReferendaApproved is emitted when a referendum has been approved and its proposal has been scheduled.
type EventReferendaApproved struct {
	Phase  Phase
	Index  types.U32
	Topics []types.Hash
}

// EventReferendaRejected is emitted when a proposal has been rejected by referendum.
type EventReferendaRejected struct {
	Phase  Phase
	Index  types.U32
	Tally  types.Tally
	Topics []types.Hash
}

// EventReferendaTimedOut is emitted when a referendum has been timed out without being decided.
type EventReferendaTimedOut struct {
	Phase  Phase
	Index  types.U32
	Tally  types.Tally
	Topics []types.Hash
}

// EventReferendaCancelled is emitted when a referendum has been cancelled.
type EventReferendaCancelled struct {
	Phase  Phase
	Index  types.U32
	Tally  types.Tally
	Topics []types.Hash
}

// EventReferendaKilled is emitted when a referendum has been killed.
type EventReferendaKilled struct {
	Phase  Phase
	Index  types.U32
	Tally  types.Tally
	Topics []types.Hash
}

// EventRecoveryClosed is emitted when a recovery process for account_1 by account_2 has been closed
type EventRecoveryClosed struct {
	Phase   Phase
	Who     types.AccountID
	Rescuer types.AccountID
	Topics  []types.Hash
}

// EventRecoveryAccountRecovered is emitted when account_1 has been successfully recovered by account_2
type EventRecoveryAccountRecovered struct {
	Phase   Phase
	Who     types.AccountID
	Rescuer types.AccountID
	Topics  []types.Hash
}

// EventRecoveryRemoved is emitted when a recovery process has been removed for an account
type EventRecoveryRemoved struct {
	Phase  Phase
	Who    types.AccountID
	Topics []types.Hash
}

// EventVestingVestingUpdated is emitted when the amount vested has been updated.
// This could indicate more funds are available.
// The balance given is the amount which is left unvested (and thus locked)
type EventVestingVestingUpdated struct {
	Phase    Phase
	Account  types.AccountID
	Unvested types.U128
	Topics   []types.Hash
}

// EventVoterListRebagged is emitted when an account is moved from one bag to another.
type EventVoterListRebagged struct {
	Phase  Phase
	Who    types.AccountID
	From   types.U64
	To     types.U64
	Topics []types.Hash
}

// EventVoterListScoreUpdated is emitted when the score of an account is updated to the given amount.
type EventVoterListScoreUpdated struct {
	Phase    Phase
	Who      types.AccountID
	NewScore types.U64
	Topics   []types.Hash
}

// EventWhitelistCallWhitelisted is emitted when a call has been whitelisted.
type EventWhitelistCallWhitelisted struct {
	Phase    Phase
	CallHash types.Hash
	Topics   []types.Hash
}

// EventWhitelistWhitelistedCallRemoved is emitted when a whitelisted call has been removed.
type EventWhitelistWhitelistedCallRemoved struct {
	Phase    Phase
	CallHash types.Hash
	Topics   []types.Hash
}

// EventWhitelistWhitelistedCallDispatched is emitted when a whitelisted call has been dispatched.
type EventWhitelistWhitelistedCallDispatched struct {
	Phase    Phase
	CallHash types.Hash
	Result   DispatchResult
	Topics   []types.Hash
}

// EventXcmPalletAttempted is emitted when the execution of an XCM message was attempted.
type EventXcmPalletAttempted struct {
	Phase   Phase
	Outcome types.Outcome
	Topics  []types.Hash
}

// EventXcmPalletSent is emitted when an XCM message was sent.
type EventXcmPalletSent struct {
	Phase       Phase
	Origin      types.MultiLocationV1
	Destination types.MultiLocationV1
	Message     []types.Instruction
	Topics      []types.Hash
}

// EventXcmPalletUnexpectedResponse is emitted when a query response which does not match a registered query
// is received.
// This may be because a matching query was never registered, it may be because it is a duplicate response, or
// because the query timed out.
type EventXcmPalletUnexpectedResponse struct {
	Phase          Phase
	OriginLocation types.MultiLocationV1
	QueryID        types.U64
	Topics         []types.Hash
}

// EventXcmPalletResponseReady is emitted when a query response has been received and is ready for
// taking with `take_response`. There is no registered notification call.
type EventXcmPalletResponseReady struct {
	Phase    Phase
	QueryID  types.U64
	Response types.Response
	Topics   []types.Hash
}

// EventXcmPalletNotified is emitted when a query response has been received and query is removed.
// The registered notification has been dispatched and executed successfully.
type EventXcmPalletNotified struct {
	Phase       Phase
	QueryID     types.U64
	PalletIndex types.U8
	CallIndex   types.U8
	Topics      []types.Hash
}

// EventXcmPalletNotifyOverweight is emitted when a query response has been received and query is removed.
// The registered notification could not be dispatched because the dispatch weight is greater than
// the maximum weight originally budgeted by this runtime for the query result.
type EventXcmPalletNotifyOverweight struct {
	Phase             Phase
	QueryID           types.U64
	PalletIndex       types.U8
	CallIndex         types.U8
	ActualWeight      Weight
	MaxBudgetedWeight Weight
	Topics            []types.Hash
}

// EventXcmPalletNotifyDispatchError is emitted when a query response has been received and query is removed.
// There was a general error with dispatching the notification call.
type EventXcmPalletNotifyDispatchError struct {
	Phase       Phase
	QueryID     types.U64
	PalletIndex types.U8
	CallIndex   types.U8
	Topics      []types.Hash
}

// EventXcmPalletNotifyDecodeFailed is emitted when a query response has been received and query is removed.
// The dispatch was unable to be decoded into a `Call`; this might be due to dispatch function having a signature
// which is not `(origin, QueryId, Response)`.
type EventXcmPalletNotifyDecodeFailed struct {
	Phase       Phase
	QueryID     types.U64
	PalletIndex types.U8
	CallIndex   types.U8
	Topics      []types.Hash
}

// EventXcmPalletInvalidResponder is emitted when the expected query response
// has been received but the origin location of the response does not match that expected.
// The query remains registered for a later, valid, response to be received and acted upon.
type EventXcmPalletInvalidResponder struct {
	Phase            Phase
	OriginLocation   types.MultiLocationV1
	QueryID          types.U64
	ExpectedLocation types.OptionMultiLocationV1
	Topics           []types.Hash
}

// EventXcmPalletInvalidResponderVersion is emitted when the expected query response
// has been received but the expected origin location placed in storage by this runtime
// previously cannot be decoded. The query remains registered.
// This is unexpected (since a location placed in storage in a previously executing
// runtime should be readable prior to query timeout) and dangerous since the possibly
// valid response will be dropped. Manual governance intervention is probably going to be
// needed.
type EventXcmPalletInvalidResponderVersion struct {
	Phase          Phase
	OriginLocation types.MultiLocationV1
	QueryID        types.U64
	Topics         []types.Hash
}

// EventXcmPalletResponseTaken is emitted when the received query response has been read and removed.
type EventXcmPalletResponseTaken struct {
	Phase   Phase
	QueryID types.U64
	Topics  []types.Hash
}

// EventXcmPalletAssetsTrapped is emitted when some assets have been placed in an asset trap.
type EventXcmPalletAssetsTrapped struct {
	Phase  Phase
	Hash   types.H256
	Origin types.MultiLocationV1
	Assets types.VersionedMultiAssets
	Topics []types.Hash
}

type XcmVersion types.U32

// EventXcmPalletVersionChangeNotified is emitted when an XCM version change notification
// message has been attempted to be sent.
type EventXcmPalletVersionChangeNotified struct {
	Phase       Phase
	Destination types.MultiLocationV1
	Result      XcmVersion
	Topics      []types.Hash
}

// EventXcmPalletSupportedVersionChanged is emitted when the supported version of a location has been changed.
// This might be through an automatic notification or a manual intervention.
type EventXcmPalletSupportedVersionChanged struct {
	Phase      Phase
	Location   types.MultiLocationV1
	XcmVersion XcmVersion
	Topics     []types.Hash
}

// EventXcmPalletNotifyTargetSendFail is emitted when a given location which had a version change
// subscription was dropped owing to an error sending the notification to it.
type EventXcmPalletNotifyTargetSendFail struct {
	Phase    Phase
	Location types.MultiLocationV1
	QueryID  types.U64
	XcmError types.XCMError
	Topics   []types.Hash
}

// EventXcmPalletNotifyTargetMigrationFail is emitted when a given location which had a
// version change subscription was dropped owing to an error migrating the location to our new XCM format.
type EventXcmPalletNotifyTargetMigrationFail struct {
	Phase    Phase
	Location types.VersionedMultiLocation
	QueryID  types.U64
	Topics   []types.Hash
}

// EventVestingVestingCompleted is emitted when an [account] has become fully vested. No further vesting can happen
type EventVestingVestingCompleted struct {
	Phase   Phase
	Account types.AccountID
	Topics  []types.Hash
}

// EventSchedulerScheduled is emitted when scheduled some task
type EventSchedulerScheduled struct {
	Phase  Phase
	When   types.U32
	Index  types.U32
	Topics []types.Hash
}

// EventSchedulerCanceled is emitted when canceled some task
type EventSchedulerCanceled struct {
	Phase  Phase
	When   types.U32
	Index  types.U32
	Topics []types.Hash
}

// EventSchedulerDispatched is emitted when dispatched some task
type EventSchedulerDispatched struct {
	Phase  Phase
	Task   TaskAddress
	ID     types.OptionBytes
	Result DispatchResult
	Topics []types.Hash
}

type SchedulerLookupError byte

const (
	// Unknown A call of this hash was not known.
	Unknown = 0
	// BadFormat The preimage for this hash was known but could not be decoded into a Call.
	BadFormat = 1
)

func (sle *SchedulerLookupError) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := SchedulerLookupError(b)
	switch vb {
	case Unknown, BadFormat:
		*sle = vb
	default:
		return fmt.Errorf("unknown SchedulerLookupError enum: %v", vb)
	}
	return err
}

func (sle SchedulerLookupError) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(sle))
}

// EventSchedulerCallLookupFailed is emitted when the call for the provided hash was not found
// so the task has been aborted.
type EventSchedulerCallLookupFailed struct {
	Phase  Phase
	Task   TaskAddress
	ID     types.OptionBytes
	Error  SchedulerLookupError
	Topics []types.Hash
}

// EventPreimageCleared is emitted when a preimage has been cleared
type EventPreimageCleared struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventPreimageNoted is emitted when a preimage has been noted
type EventPreimageNoted struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventPreimageRequested is emitted when a preimage has been requested
type EventPreimageRequested struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventProxyProxyExecuted is emitted when a proxy was executed correctly, with the given [result]
type EventProxyProxyExecuted struct {
	Phase  Phase
	Result DispatchResult
	Topics []types.Hash
}

// EventProxyAnonymousCreated is emitted when an anonymous account has been created by new proxy with given,
// disambiguation index and proxy type.
type EventProxyAnonymousCreated struct {
	Phase               Phase
	Anonymous           types.AccountID
	Who                 types.AccountID
	ProxyType           types.U8
	DisambiguationIndex types.U16
	Topics              []types.Hash
}

// EventProxyProxyAdded is emitted when a proxy was added.
type EventProxyProxyAdded struct {
	Phase     Phase
	Delegator types.AccountID
	Delegatee types.AccountID
	ProxyType types.U8
	Delay     types.U32
	Topics    []types.Hash
}

// EventProxyProxyRemoved is emitted when a proxy was removed.
type EventProxyProxyRemoved struct {
	Phase       Phase
	Delegator   types.AccountID
	Delegatee   types.AccountID
	ProxyType   types.U8
	BlockNumber types.U32
	Topics      []types.Hash
}

// EventProxyAnnounced is emitted when an announcement was placed to make a call in the future
type EventProxyAnnounced struct {
	Phase    Phase
	Real     types.AccountID
	Proxy    types.AccountID
	CallHash types.Hash
	Topics   []types.Hash
}

// EventSudoSudid is emitted when a sudo just took place.
type EventSudoSudid struct {
	Phase  Phase
	Result DispatchResult
	Topics []types.Hash
}

// EventSudoKeyChanged is emitted when the sudoer just switched identity; the old key is supplied.
type EventSudoKeyChanged struct {
	Phase     Phase
	AccountID types.AccountID
	Topics    []types.Hash
}

// A sudo just took place.
type EventSudoAsDone struct {
	Phase  Phase
	Done   bool
	Topics []types.Hash
}

// EventTreasuryProposed is emitted when New proposal.
type EventTreasuryProposed struct {
	Phase         Phase
	ProposalIndex types.U32
	Topics        []types.Hash
}

// EventTreasurySpending is emitted when we have ended a spend period and will now allocate funds.
type EventTreasurySpending struct {
	Phase           Phase
	BudgetRemaining types.U128
	Topics          []types.Hash
}

// EventTreasuryAwarded is emitted when some funds have been allocated.
type EventTreasuryAwarded struct {
	Phase         Phase
	ProposalIndex types.U32
	Amount        types.U128
	Beneficiary   types.AccountID
	Topics        []types.Hash
}

// EventTreasuryRejected is emitted when s proposal was rejected; funds were slashed.
type EventTreasuryRejected struct {
	Phase         Phase
	ProposalIndex types.U32
	Amount        types.U128
	Topics        []types.Hash
}

// EventTreasuryBurnt is emitted when some of our funds have been burnt.
type EventTreasuryBurnt struct {
	Phase  Phase
	Burn   types.U128
	Topics []types.Hash
}

// EventTreasuryRollover is emitted when spending has finished; this is the amount that rolls over until next spend.
type EventTreasuryRollover struct {
	Phase           Phase
	BudgetRemaining types.U128
	Topics          []types.Hash
}

// EventTreasuryDeposit is emitted when some funds have been deposited.
type EventTreasuryDeposit struct {
	Phase     Phase
	Deposited types.U128
	Topics    []types.Hash
}

// EventTipsNewTip is emitted when a new tip suggestion has been opened.
type EventTipsNewTip struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventTipsTipClosing is emitted when a tip suggestion has reached threshold and is closing.
type EventTipsTipClosing struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

// EventTipsTipClosed is emitted when a tip suggestion has been closed.
type EventTipsTipClosed struct {
	Phase     Phase
	Hash      types.Hash
	AccountID types.AccountID
	Balance   types.U128
	Topics    []types.Hash
}

// EventTipsTipSlashed is emitted when a tip suggestion has been slashed.
type EventTipsTipSlashed struct {
	Phase     Phase
	Hash      types.Hash
	AccountID types.AccountID
	Balance   types.U128
	Topics    []types.Hash
}

// EventTransactionStorageStored is emitted when data is stored under a specific index.
type EventTransactionStorageStored struct {
	Phase  Phase
	Index  types.U32
	Topics []types.Hash
}

// EventTransactionStorageRenewed is emitted when data is renewed under a specific index.
type EventTransactionStorageRenewed struct {
	Phase  Phase
	Index  types.U32
	Topics []types.Hash
}

// EventTransactionStorageProofChecked is emitted when storage proof was successfully checked.
type EventTransactionStorageProofChecked struct {
	Phase  Phase
	Topics []types.Hash
}

type EventTransactionPaymentTransactionFeePaid struct {
	Phase     Phase
	Who       types.AccountID
	ActualFee types.U128
	Tip       types.U128
	Topics    []types.Hash
}

// EventTipsTipRetracted is emitted when a tip suggestion has been retracted.
type EventTipsTipRetracted struct {
	Phase  Phase
	Hash   types.Hash
	Topics []types.Hash
}

type BountyIndex types.U32

// EventBountiesBountyProposed is emitted for a new bounty proposal.
type EventBountiesBountyProposed struct {
	Phase         Phase
	ProposalIndex BountyIndex
	Topics        []types.Hash
}

// EventBountiesBountyRejected is emitted when a bounty proposal was rejected; funds were slashed.
type EventBountiesBountyRejected struct {
	Phase         Phase
	ProposalIndex BountyIndex
	Bond          types.U128
	Topics        []types.Hash
}

// EventBountiesBountyBecameActive is emitted when a bounty proposal is funded and became active
type EventBountiesBountyBecameActive struct {
	Phase  Phase
	Index  BountyIndex
	Topics []types.Hash
}

// EventBountiesBountyAwarded is emitted when a bounty is awarded to a beneficiary
type EventBountiesBountyAwarded struct {
	Phase       Phase
	Index       BountyIndex
	Beneficiary types.AccountID
	Topics      []types.Hash
}

// EventBountiesBountyClaimed is emitted when a bounty is claimed by beneficiary
type EventBountiesBountyClaimed struct {
	Phase       Phase
	Index       BountyIndex
	Payout      types.U128
	Beneficiary types.AccountID
	Topics      []types.Hash
}

// EventBountiesBountyCanceled is emitted when a bounty is cancelled.
type EventBountiesBountyCanceled struct {
	Phase  Phase
	Index  BountyIndex
	Topics []types.Hash
}

// EventBountiesBountyExtended is emitted when a bounty is extended.
type EventBountiesBountyExtended struct {
	Phase  Phase
	Index  BountyIndex
	Topics []types.Hash
}

// EventChildBountiesAdded is emitted when a child-bounty is added.
type EventChildBountiesAdded struct {
	Phase      Phase
	Index      BountyIndex
	ChildIndex BountyIndex
	Topics     []types.Hash
}

// EventChildBountiesAwarded is emitted when a child-bounty is awarded to a beneficiary.
type EventChildBountiesAwarded struct {
	Phase       Phase
	Index       BountyIndex
	ChildIndex  BountyIndex
	Beneficiary types.AccountID
	Topics      []types.Hash
}

// EventChildBountiesClaimed is emitted when a child-bounty is claimed by a beneficiary.
type EventChildBountiesClaimed struct {
	Phase       Phase
	Index       BountyIndex
	ChildIndex  BountyIndex
	Payout      types.U128
	Beneficiary types.AccountID
	Topics      []types.Hash
}

// EventChildBountiesCanceled is emitted when a child-bounty is canceled.
type EventChildBountiesCanceled struct {
	Phase      Phase
	Index      BountyIndex
	ChildIndex BountyIndex
	Topics     []types.Hash
}

// EventUniquesApprovalCancelled is emitted when an approval for a delegate account to transfer the instance of
// an asset class was cancelled by its owner
type EventUniquesApprovalCancelled struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Owner        types.AccountID
	Delegate     types.AccountID
	Topics       []types.Hash
}

// EventUniquesApprovedTransfer is emitted when an `instance` of an asset `class` has been approved by the `owner`
// for transfer by a `delegate`.
type EventUniquesApprovedTransfer struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Owner        types.AccountID
	Delegate     types.AccountID
	Topics       []types.Hash
}

// EventUniquesAssetStatusChanged is emitted when an asset `class` has had its attributes changed by the `Force` origin
type EventUniquesAssetStatusChanged struct {
	Phase        Phase
	CollectionID types.U64
	Topics       []types.Hash
}

// EventUniquesAttributeCleared is emitted when an attribute metadata has been cleared for an asset class or instance
type EventUniquesAttributeCleared struct {
	Phase        Phase
	CollectionID types.U64
	MaybeItem    types.Option[types.U128]
	Key          types.Bytes
	Topics       []types.Hash
}

// EventUniquesAttributeSet is emitted when a new attribute metadata has been set for an asset class or instance
type EventUniquesAttributeSet struct {
	Phase        Phase
	CollectionID types.U64
	MaybeItem    types.Option[types.U128]
	Key          types.Bytes
	Value        types.Bytes
	Topics       []types.Hash
}

// EventUniquesBurned is emitted when an asset `instance` was destroyed
type EventUniquesBurned struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Owner        types.AccountID
	Topics       []types.Hash
}

// EventUniquesClassFrozen is emitted when some asset `class` was frozen
type EventUniquesClassFrozen struct {
	Phase        Phase
	CollectionID types.U64
	Topics       []types.Hash
}

// EventUniquesClassMetadataCleared is emitted when metadata has been cleared for an asset class
type EventUniquesClassMetadataCleared struct {
	Phase        Phase
	CollectionID types.U64
	Topics       []types.Hash
}

// EventUniquesClassMetadataSet is emitted when new metadata has been set for an asset class
type EventUniquesClassMetadataSet struct {
	Phase        Phase
	CollectionID types.U64
	Data         types.Bytes
	IsFrozen     types.Bool
	Topics       []types.Hash
}

// EventUniquesClassThawed is emitted when some asset `class` was thawed
type EventUniquesClassThawed struct {
	Phase        Phase
	CollectionID types.U64
	Topics       []types.Hash
}

// EventUniquesCreated is emitted when an asset class was created
type EventUniquesCreated struct {
	Phase        Phase
	CollectionID types.U64
	Creator      types.AccountID
	Owner        types.AccountID
	Topics       []types.Hash
}

// EventUniquesDestroyed is emitted when an asset `class` was destroyed
type EventUniquesDestroyed struct {
	Phase        Phase
	CollectionID types.U64
	Topics       []types.Hash
}

// EventUniquesForceCreated is emitted when an asset class was force-created
type EventUniquesForceCreated struct {
	Phase        Phase
	CollectionID types.U64
	Owner        types.AccountID
	Topics       []types.Hash
}

// EventUniquesFrozen is emitted when some asset `instance` was frozen
type EventUniquesFrozen struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Topics       []types.Hash
}

// EventUniquesIssued is emitted when an asset instance was issued
type EventUniquesIssued struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Owner        types.AccountID
	Topics       []types.Hash
}

// EventUniquesMetadataCleared is emitted when metadata has been cleared for an asset instance
type EventUniquesMetadataCleared struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Topics       []types.Hash
}

// EventUniquesMetadataSet is emitted when metadata has been set for an asset instance
type EventUniquesMetadataSet struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Data         types.Bytes
	IsFrozen     types.Bool
	Topics       []types.Hash
}

// EventUniquesOwnerChanged is emitted when the owner changed
type EventUniquesOwnerChanged struct {
	Phase        Phase
	CollectionID types.U64
	NewOwner     types.AccountID
	Topics       []types.Hash
}

// EventUniquesRedeposited is emitted when metadata has been cleared for an asset instance
type EventUniquesRedeposited struct {
	Phase           Phase
	CollectionID    types.U64
	SuccessfulItems []types.U128
	Topics          []types.Hash
}

// EventUniquesTeamChanged is emitted when the management team changed
type EventUniquesTeamChanged struct {
	Phase        Phase
	CollectionID types.U64
	Issuer       types.AccountID
	Admin        types.AccountID
	Freezer      types.AccountID
	Topics       []types.Hash
}

// EventUniquesThawed is emitted when some asset instance was thawed
type EventUniquesThawed struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	Topics       []types.Hash
}

// EventUniquesTransferred is emitted when some asset instance was transferred
type EventUniquesTransferred struct {
	Phase        Phase
	CollectionID types.U64
	ItemID       types.U128
	From         types.AccountID
	To           types.AccountID
	Topics       []types.Hash
}

// EventUMPInvalidFormat is emitted when the upward message is invalid XCM.
type EventUMPInvalidFormat struct {
	Phase     Phase
	MessageID [32]types.U8
	Topics    []types.Hash
}

// EventUMPUnsupportedVersion is emitted when the upward message is unsupported version of XCM.
type EventUMPUnsupportedVersion struct {
	Phase     Phase
	MessageID [32]types.U8
	Topics    []types.Hash
}

// EventUMPExecutedUpward is emitted when the upward message executed with the given outcome.
type EventUMPExecutedUpward struct {
	Phase     Phase
	MessageID [32]types.U8
	Outcome   types.Outcome
	Topics    []types.Hash
}

// EventUMPWeightExhausted is emitted when the weight limit for handling upward messages was reached.
type EventUMPWeightExhausted struct {
	Phase     Phase
	MessageID [32]types.U8
	Remaining Weight
	Required  Weight
	Topics    []types.Hash
}

// EventUMPUpwardMessagesReceived is emitted when some upward messages have been received and will be processed.
type EventUMPUpwardMessagesReceived struct {
	Phase       Phase
	ParachainID ParachainID
	Count       types.U32
	Size        types.U32
	Topics      []types.Hash
}

// EventUMPOverweightEnqueued is emitted when the weight budget was exceeded for an individual upward message.
// This message can be later dispatched manually using `service_overweight` dispatchable using
// the assigned `overweight_index`.
type EventUMPOverweightEnqueued struct {
	Phase           Phase
	ParachainID     ParachainID
	MessageID       [32]types.U8
	OverweightIndex types.U64
	RequiredWeight  Weight
	Topics          []types.Hash
}

// EventUMPOverweightServiced is emitted when the upward message from the
// overweight queue was executed with the given actual weight used.
type EventUMPOverweightServiced struct {
	Phase           Phase
	OverweightIndex types.U64
	Used            Weight
	Topics          []types.Hash
}

// EventContractsInstantiated is emitted when a contract is deployed by address at the specified address
type EventContractsInstantiated struct {
	Phase    Phase
	Deployer types.AccountID
	Contract types.AccountID
	Topics   []types.Hash
}

// EventContractsTerminated The only way for a contract to be removed and emitting this event is by calling
// `seal_terminate`
type EventContractsTerminated struct {
	Phase       Phase
	Contract    types.AccountID
	Beneficiary types.AccountID
	Topics      []types.Hash
}

// EventConvictionVotingDelegated is emitted when an account has delegated their vote to another account.
type EventConvictionVotingDelegated struct {
	Phase  Phase
	Who    types.AccountID
	Target types.AccountID
	Topics []types.Hash
}

// EventConvictionVotingUndelegated is emitted when an account has delegated their vote to another account.
type EventConvictionVotingUndelegated struct {
	Phase  Phase
	Who    types.AccountID
	Target types.AccountID
	Topics []types.Hash
}

// EventContractsContractEmitted is emitted when a custom event emitted by the contract
type EventContractsContractEmitted struct {
	Phase    Phase
	Contract types.AccountID
	Data     types.Bytes
	Topics   []types.Hash
}

// EventContractsCalled is emitted when a contract is called
type EventContractsCalled struct {
	Phase    Phase
	Caller   types.AccountID
	Contract types.AccountID
	Topics   []types.Hash
}

// EventContractsContractCodeUpdated is emitted when a contract's code was updated
type EventContractsContractCodeUpdated struct {
	Phase       Phase
	Contract    types.AccountID
	NewCodeHash types.Hash
	OldCodeHash types.Hash
	Topics      []types.Hash
}

type EventCollatorSelectionNewInvulnerables struct {
	Phase            Phase
	NewInvulnerables []types.AccountID
	Topics           []types.Hash
}

type EventCollatorSelectionNewDesiredCandidates struct {
	Phase                Phase
	NewDesiredCandidates types.U32
	Topics               []types.Hash
}

type EventCollatorSelectionNewCandidacyBond struct {
	Phase            Phase
	NewCandidacyBond types.U128
	Topics           []types.Hash
}

type EventCollatorSelectionCandidateAdded struct {
	Phase          Phase
	CandidateAdded types.AccountID
	Bond           types.U128
	Topics         []types.Hash
}

type EventCollatorSelectionCandidateRemoved struct {
	Phase            Phase
	CandidateRemoved types.AccountID
	Topics           []types.Hash
}

// EventContractsCodeRemoved is emitted when code with the specified hash was removed
type EventContractsCodeRemoved struct {
	Phase    Phase
	CodeHash types.Hash
	Topics   []types.Hash
}

// EventContractsCodeStored is emitted when code with the specified hash has been stored
type EventContractsCodeStored struct {
	Phase    Phase
	CodeHash types.Hash
	Topics   []types.Hash
}

// EventContractsScheduleUpdated is triggered when the current [schedule] is updated
type EventContractsScheduleUpdated struct {
	Phase    Phase
	Schedule types.U32
	Topics   []types.Hash
}

// EventContractsContractExecution is triggered when an event deposited upon execution of a contract from the account
type EventContractsContractExecution struct {
	Phase   Phase
	Account types.AccountID
	Data    types.Bytes
	Topics  []types.Hash
}

// EventUtilityBatchInterrupted is emitted when a batch of dispatches did not complete fully.
// Index of first failing dispatch given, as well as the error.
type EventUtilityBatchInterrupted struct {
	Phase         Phase
	Index         types.U32
	DispatchError types.DispatchError
	Topics        []types.Hash
}

// EventUtilityBatchCompleted is emitted when a batch of dispatches completed fully with no error.
type EventUtilityBatchCompleted struct {
	Phase  Phase
	Topics []types.Hash
}

// EventUtilityDispatchedAs is emitted when a call was dispatched
type EventUtilityDispatchedAs struct {
	Phase  Phase
	Index  types.U32
	Result DispatchResult
	Topics []types.Hash
}

// EventUtilityItemCompleted is emitted when a single item within a Batch of dispatches has completed with no error
type EventUtilityItemCompleted struct {
	Phase  Phase
	Topics []types.Hash
}

// EventUtilityNewMultisig is emitted when a new multisig operation has begun.
// First param is the account that is approving, second is the multisig account, third is hash of the call.
type EventMultisigNewMultisig struct {
	Phase    Phase
	Who, ID  types.AccountID
	CallHash types.Hash
	Topics   []types.Hash
}

// EventNftSalesForSale is emitted when an NFT is out for sale.
type EventNftSalesForSale struct {
	Phase      Phase
	ClassID    types.U64
	InstanceID types.U128
	Sale       types.Sale
	Topics     []types.Hash
}

// EventNftSalesRemoved is emitted when an NFT is removed.
type EventNftSalesRemoved struct {
	Phase      Phase
	ClassID    types.U64
	InstanceID types.U128
	Topics     []types.Hash
}

// EventNftSalesSold is emitted when an NFT is sold.
type EventNftSalesSold struct {
	Phase      Phase
	ClassID    types.U64
	InstanceID types.U128
	Sale       types.Sale
	Buyer      types.AccountID
	Topics     []types.Hash
}

// TimePoint is a global extrinsic index, formed as the extrinsic index within a block,
// together with that block's height.
type TimePoint struct {
	Height types.U32
	Index  types.U32
}

// TaskAddress holds the location of a scheduled task that can be used to remove it
type TaskAddress struct {
	When  types.U32
	Index types.U32
}

// EventUtility is emitted when a multisig operation has been approved by someone. First param is the account that is
// approving, third is the multisig account, fourth is hash of the call.
type EventMultisigApproval struct {
	Phase     Phase
	Who       types.AccountID
	TimePoint TimePoint
	ID        types.AccountID
	CallHash  types.Hash
	Topics    []types.Hash
}

// DispatchResult can be returned from dispatchable functions
type DispatchResult struct {
	Ok    bool
	Error types.DispatchError
}

func (d *DispatchResult) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.Ok = true
		return nil
	default:
		return decoder.Decode(&d.Error)
	}
}

func (d DispatchResult) Encode(encoder scale.Encoder) error {
	if d.Ok {
		return encoder.PushByte(0)
	}

	if err := encoder.PushByte(1); err != nil {
		return err
	}

	return encoder.Encode(d.Error)
}

// EventUtility is emitted when a multisig operation has been executed. First param is the account that is
// approving, third is the multisig account, fourth is hash of the call to be executed.
type EventMultisigExecuted struct {
	Phase     Phase
	Who       types.AccountID
	TimePoint TimePoint
	ID        types.AccountID
	CallHash  types.Hash
	Result    DispatchResult
	Topics    []types.Hash
}

// EventUtility is emitted when a multisig operation has been cancelled. First param is the account that is
// cancelling, third is the multisig account, fourth is hash of the call.
type EventMultisigCancelled struct {
	Phase     Phase
	Who       types.AccountID
	TimePoint TimePoint
	ID        types.AccountID
	CallHash  types.Hash
	Topics    []types.Hash
}
