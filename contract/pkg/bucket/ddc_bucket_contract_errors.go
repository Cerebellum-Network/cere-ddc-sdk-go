package bucket

import "errors"

const (
	bucketDoesNotExist = iota
	providerDoesNotExist
	unauthorizedProvider
	transferFailed
)

var (
	ErrBucketDoesNotExist   = errors.New("bucket doesn't exist")
	ErrProviderDoesNotExist = errors.New("provider doesn't exist")
	ErrUnauthorizedProvider = errors.New("unauthorized provider")
	ErrTransferFailed       = errors.New("transfer failed")
	ErrUndefined            = errors.New("undefined error")
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
	default:
		return ErrUndefined
	}
}
