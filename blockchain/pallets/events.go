package pallets

import (
	chainbridge "github.com/Cerebellum-Network/chainbridge-substrate-events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords
	chainbridge.Events

	Contracts_Called                               []EventContractsCalled                               //nolint:stylecheck,golint
	Contracts_DelegateCalled                       []EventContractsDelegateCalled                       //nolint:stylecheck,golint
	Contracts_StorageDepositTransferredAndHeld     []EventContractsStorageDepositTransferredAndHeld     //nolint:stylecheck,golint
	Contracts_StorageDepositTransferredAndReleased []EventContractsStorageDepositTransferredAndReleased //nolint:stylecheck,golint

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
	DdcCustomers_BucketRemoved        []EventDdcCustomersBucketRemoved        //nolint:stylecheck,golint

	DdcNodes_NodeCreated       []EventDdcNodesNodeCreated       //nolint:stylecheck,golint
	DdcNodes_NodeDeleted       []EventDdcNodesNodeDeleted       //nolint:stylecheck,golint
	DdcNodes_NodeParamsChanged []EventDdcNodesNodeParamsChanged //nolint:stylecheck,golint

	DdcPayouts_BillingReportInitialized    []EventDdcPayoutsBillingReportInitialized    //nolint:stylecheck,golint
	DdcPayouts_ChargingStarted             []EventDdcPayoutsChargingStarted             //nolint:stylecheck,golint
	DdcPayouts_Charged                     []EventDdcPayoutsCharged                     //nolint:stylecheck,golint
	DdcPayouts_ChargeFailed                []EventDdcPayoutsChargeFailed                //nolint:stylecheck,golint
	DdcPayouts_Indebted                    []EventDdcPayoutsIndebted                    //nolint:stylecheck,golint
	DdcPayouts_ChargingFinished            []EventDdcPayoutsChargingFinished            //nolint:stylecheck,golint
	DdcPayouts_TreasuryFeesCollected       []EventDdcPayoutsTreasuryFeesCollected       //nolint:stylecheck,golint
	DdcPayouts_ClusterReserveFeesCollected []EventDdcPayoutsClusterReserveFeesCollected //nolint:stylecheck,golint
	DdcPayouts_ValidatorFeesCollected      []EventDdcPayoutsValidatorFeesCollected      //nolint:stylecheck,golint
	DdcPayouts_RewardingStarted            []EventDdcPayoutsRewardingStarted            //nolint:stylecheck,golint
	DdcPayouts_Rewarded                    []EventDdcPayoutsRewarded                    //nolint:stylecheck,golint
	DdcPayouts_RewardingFinished           []EventDdcPayoutsRewardingFinished           //nolint:stylecheck,golint
	DdcPayouts_BillingReportFinalized      []EventDdcPayoutsBillingReportFinalized      //nolint:stylecheck,golint
	DdcPayouts_AuthorisedCaller            []EventDdcPayoutsAuthorisedCaller            //nolint:stylecheck,golint

	DdcStaking_Bonded    []EventDdcStakingBonded    //nolint:stylecheck,golint
	DdcStaking_Chilled   []EventDdcStakingChilled   //nolint:stylecheck,golint
	DdcStaking_ChillSoon []EventDdcStakingChillSoon //nolint:stylecheck,golint
	DdcStaking_Unbonded  []EventDdcStakingUnbonded  //nolint:stylecheck,golint
	DdcStaking_Withdrawn []EventDdcStakingWithdrawn //nolint:stylecheck,golint
	DdcStaking_Activated []EventDdcStakingActivated //nolint:stylecheck,golint
	DdcStaking_LeaveSoon []EventDdcStakingLeaveSoon //nolint:stylecheck,golint
	DdcStaking_Left      []EventDdcStakingLeft      //nolint:stylecheck,golint
}
