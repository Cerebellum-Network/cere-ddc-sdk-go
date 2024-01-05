package pallets

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords

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
}
