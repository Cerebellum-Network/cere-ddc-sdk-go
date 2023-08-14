package bucket

import "errors"

const (
	bucketDoesNotExist = iota
	providerDoesNotExist
	unauthorizedProvider
	transferFailed
	nodeDoesNotExist
	cdnNodeDoesNotExist
	nodeAlreadyExists
	cdnNodeAlreadyExists
	accountDoesNotExist
	paramsDoesNotExist
	paramsSizeExceedsLimit
	onlyOwner
	onlyNodeProvider
	onlyCdnNodeProvider
	onlyClusterManager
	onlyTrustedClusterManager
	onlyValidator
	onlySuperAdmin
	onlyClusterManagerOrNodeProvider
	onlyClusterManagerOrCdnNodeProvider
	unauthorized
	clusterDoesNotExist
	clusterIsNotEmpty
	topologyIsNotCreated
	topologyAlreadyExists
	nodesSizeExceedsLimit
	cdnNodesSizeExceedsLimit
	vNodesSizeExceedsLimit
	accountsSizeExceedsLimit
	nodeIsNotAddedToCluster
	nodeIsAddedToCluster
	cdnNodeIsNotAddedToCluster
	cdnNodeIsAddedToCluster
	vNodeDoesNotExistsInCluster
	vNodeIsNotAssignedToNode
	vNodeIsAlreadyAssignedToNode
	atLeastOneVNodeHasToBeAssigned
	nodeProviderIsNotSuperAdmin
	cdnNodeOwnerIsNotSuperAdmin
	bondingPeriodNotFinished
	insufficientBalance
	insufficientNodeResources
	insufficientClusterResources
	eraSettingFailed
)

//ToDo update error regarding latest contract
var (
	ErrBucketDoesNotExist                  = errors.New("bucket doesn't exist")
	ErrProviderDoesNotExist                = errors.New("provider doesn't exist")
	ErrUnauthorizedProvider                = errors.New("unauthorized provider")
	ErrTransferFailed                      = errors.New("transfer failed")
	ErrUndefined                           = errors.New("undefined error")
	ErrNodeDoesNotExist                    = errors.New("node does not exist")
	ErrCdnNodeDoesNotExist                 = errors.New("cdn node does not exist")
	ErrNodeAlreadyExists                   = errors.New("node already exists")
	ErrCdnNodeAlreadyExists                = errors.New("cdn node already exists")
	ErrAccountDoesNotExist                 = errors.New("account does not exist")
	ErrParamsDoesNotExist                  = errors.New("params does not exist")
	ErrParamsSizeExceedsLimit              = errors.New("params size exceeds limit")
	ErrOnlyOwner                           = errors.New("only owner")
	ErrOnlyNodeProvider                    = errors.New("only node provider")
	ErrOnlyCdnNodeProvider                 = errors.New("only cdn node provider")
	ErrOnlyClusterManager                  = errors.New("only cluster manager")
	ErrOnlyTrustedClusterManager           = errors.New("only trusted cluster manager")
	ErrOnlyValidator                       = errors.New("only validator")
	ErrOnlySuperAdmin                      = errors.New("only super admin")
	ErrOnlyClusterManagerOrNodeProvider    = errors.New("only cluster manager or node provider")
	ErrOnlyClusterManagerOrCdnNodeProvider = errors.New("only cluster manager or cdn node provider")
	ErrUnauthorized                        = errors.New("unauthorized")
	ErrClusterDoesNotExist                 = errors.New("cluster does not exist")
	ErrClusterIsNotEmpty                   = errors.New("cluster is not empty")
	ErrTopologyIsNotCreated                = errors.New("topology is not created")
	ErrTopologyAlreadyExists               = errors.New("topology already exists")
	ErrNodesSizeExceedsLimit               = errors.New("nodes size exceeds limit")
	ErrCdnNodesSizeExceedsLimit            = errors.New("cdn nodes size exceeds limit")
	ErrVNodesSizeExceedsLimit              = errors.New("vnodes size exceeds limit")
	ErrAccountsSizeExceedsLimit            = errors.New("accounts size exceeds limit")
	ErrNodeIsNotAddedToCluster             = errors.New("node is not added to cluster")
	ErrNodeIsAddedToCluster                = errors.New("node is added to cluster")
	ErrCdnNodeIsNotAddedToCluster          = errors.New("cdn node is not added to cluster")
	ErrCdnNodeIsAddedToCluster             = errors.New("cdn node is added to cluster")
	ErrVNodeDoesNotExistsInCluster         = errors.New("vnode does not exists in cluster")
	ErrVNodeIsNotAssignedToNode            = errors.New("vnode is not assigned to node")
	ErrVNodeIsAlreadyAssignedToNode        = errors.New("vnode is already assigned to node")
	ErrAtLeastOneVNodeHasToBeAssigned      = errors.New("at least one vnode has to be assigned")
	ErrNodeProviderIsNotSuperAdmin         = errors.New("node provider is not super admin")
	ErrCdnNodeOwnerIsNotSuperAdmin         = errors.New("cdn node owner is not super admin")
	ErrBondingPeriodNotFinished            = errors.New("bonding period is not finished")
	ErrInsufficientBalance                 = errors.New("insufficient balance")
	ErrInsufficientNodeResources           = errors.New("insufficient node resources")
	ErrInsufficientClusterResources        = errors.New("insufficient cluster resources")
	ErrEraSettingFailed                    = errors.New("era setting failed")
)

func parseDdcBucketContractError(error uint8) error {
	switch error {
	case bucketDoesNotExist:
		return ErrBucketDoesNotExist
	case providerDoesNotExist:
		return ErrProviderDoesNotExist
	case unauthorizedProvider:
		return ErrUnauthorizedProvider
	case transferFailed:
		return ErrTransferFailed
	case nodeDoesNotExist:
		return ErrNodeDoesNotExist
	case cdnNodeDoesNotExist:
		return ErrCdnNodeDoesNotExist
	case nodeAlreadyExists:
		return ErrNodeAlreadyExists
	case cdnNodeAlreadyExists:
		return ErrCdnNodeAlreadyExists
	case accountDoesNotExist:
		return ErrAccountDoesNotExist
	case paramsDoesNotExist:
		return ErrParamsDoesNotExist
	case paramsSizeExceedsLimit:
		return ErrParamsSizeExceedsLimit
	case onlyOwner:
		return ErrOnlyOwner
	case onlyNodeProvider:
		return ErrOnlyNodeProvider
	case onlyCdnNodeProvider:
		return ErrOnlyCdnNodeProvider
	case onlyClusterManager:
		return ErrOnlyClusterManager
	case onlyTrustedClusterManager:
		return ErrOnlyTrustedClusterManager
	case onlyValidator:
		return ErrOnlyValidator
	case onlySuperAdmin:
		return ErrOnlySuperAdmin
	case onlyClusterManagerOrNodeProvider:
		return ErrOnlyClusterManagerOrNodeProvider
	case onlyClusterManagerOrCdnNodeProvider:
		return ErrOnlyClusterManagerOrCdnNodeProvider
	case unauthorized:
		return ErrUnauthorized
	case clusterDoesNotExist:
		return ErrClusterDoesNotExist
	case clusterIsNotEmpty:
		return ErrClusterIsNotEmpty
	case topologyIsNotCreated:
		return ErrTopologyIsNotCreated
	case topologyAlreadyExists:
		return ErrTopologyAlreadyExists
	case nodesSizeExceedsLimit:
		return ErrNodesSizeExceedsLimit
	case cdnNodesSizeExceedsLimit:
		return ErrCdnNodesSizeExceedsLimit
	case vNodesSizeExceedsLimit:
		return ErrVNodesSizeExceedsLimit
	case accountsSizeExceedsLimit:
		return ErrAccountsSizeExceedsLimit
	case nodeIsNotAddedToCluster:
		return ErrNodeIsNotAddedToCluster
	case nodeIsAddedToCluster:
		return ErrNodeIsAddedToCluster
	case cdnNodeIsNotAddedToCluster:
		return ErrCdnNodeIsNotAddedToCluster
	case cdnNodeIsAddedToCluster:
		return ErrCdnNodeIsAddedToCluster
	case vNodeDoesNotExistsInCluster:
		return ErrVNodeDoesNotExistsInCluster
	case vNodeIsNotAssignedToNode:
		return ErrVNodeIsNotAssignedToNode
	case vNodeIsAlreadyAssignedToNode:
		return ErrVNodeIsAlreadyAssignedToNode
	case atLeastOneVNodeHasToBeAssigned:
		return ErrAtLeastOneVNodeHasToBeAssigned
	case nodeProviderIsNotSuperAdmin:
		return ErrNodeProviderIsNotSuperAdmin
	case cdnNodeOwnerIsNotSuperAdmin:
		return ErrCdnNodeOwnerIsNotSuperAdmin
	case bondingPeriodNotFinished:
		return ErrBondingPeriodNotFinished
	case insufficientBalance:
		return ErrInsufficientBalance
	case insufficientNodeResources:
		return ErrInsufficientNodeResources
	case insufficientClusterResources:
		return ErrInsufficientClusterResources
	case eraSettingFailed:
		return ErrEraSettingFailed
	default:
		return ErrUndefined
	}
}
