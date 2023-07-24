package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/decred/base58"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

const addressLength = 32 + 1 + 2

var defaultSS58Prefix = []byte("SS58PRE")

func DecodeAccountIDFromSS58(address string) (types.AccountID, error) {
	a := base58.Decode(address)

	if len(a) == 0 {
		return types.AccountID{}, errors.New("no address bytes encode")
	}

	if len(a) == addressLength {
		addr := a[:addressLength-2]

		hash, err := blake2b.New512([]byte{})
		if err != nil {
			return types.AccountID{}, fmt.Errorf("[DecodeAccountID] invalid blake2b: %w", err)
		}

		buf := make([]byte, 0, len(defaultSS58Prefix)+len(addr)+1)
		buf = append(buf, defaultSS58Prefix...)
		buf = append(buf, addr...)

		_, err = hash.Write(buf)
		if err != nil {
			return types.AccountID{}, fmt.Errorf("[DecodeAccountID] invalid blake2b write: %w", err)
		}

		h := hash.Sum(nil)

		if (a[addressLength-2] == h[0]) && (a[addressLength-1] == h[1]) {
			id, err := types.NewAccountID(a[1:33])
			if err != nil {
				return types.AccountID{}, err
			}

			return *id, nil
		}

		return types.AccountID{},
			fmt.Errorf("invalid checksum %x%x, expected %x%x", a[addressLength-2], a[addressLength-1], h[0], h[1])
	}

	return types.AccountID{}, errors.New("invalid length")
}

func GetContractData(method []byte, args ...interface{}) ([]byte, error) {
	log.Warnf("=====> GetContractData 1")
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	log.Warnf("=====> GetContractData 2")
	buf.Write(method)
	log.Warnf("=====> GetContractData 3")

	encoder := scale.NewEncoder(buf)
	log.Warnf("=====> GetContractData 4")
	for _, v := range args {
		log.Warnf("=====> GetContractData 5, value: %v hex: %x", v, v)
		err := encoder.Encode(v)
		log.Warnf("=====> GetContractData 6, value_encoded: %v hex_encoded: %x", v, v)
		if err != nil {
			log.Warnf("=====> GetContractData 7")
			return nil, err
		}
	}

	log.Warnf("=====> GetContractData 8")
	return buf.Bytes(), nil
}

func isClosedNetworkError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "use of closed network connection")
}
