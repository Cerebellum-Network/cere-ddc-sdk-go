package bucket

import (
	"errors"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

const (
	okPrefix  = "0x00"
	errPrefix = "0x01"
)

type Result struct {
	data interface{}
	err  error
}

func (result *Result) decodeDdcBucketContract(encodedData string) error {

	if strings.HasPrefix(encodedData, okPrefix) {
		encodedData = strings.TrimPrefix(encodedData, okPrefix)
		if err := codec.DecodeFromHex(encodedData, result.data); err != nil {
			return err
		}
		return nil
	}

	if strings.HasPrefix(encodedData, errPrefix) {
		encodedData = strings.TrimPrefix(encodedData, errPrefix)
		var errRes types.U8
		if err := codec.DecodeFromHex(encodedData, &errRes); err != nil {
			return err
		}

		result.err = parseDdcBucketContractError(uint8(errRes))
		return nil
	}

	return errors.New("can't decode storage contract result")
}
