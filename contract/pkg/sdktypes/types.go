package sdktypes

import (
	"context"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"reflect"
)

type ContractEventDispatchEntry struct {
	ArgumentType reflect.Type
	Handler      ContractEventHandler
}

type ContractEventHandler func(interface{})

type BlockchainClient interface {
	CallToReadEncoded(contractAddressSS58 string, fromAddress string, method []byte, args ...interface{}) (string, error)
	CallToExec(ctx context.Context, contractCall ContractCall) (types.Hash, error)
	Deploy(ctx context.Context, deployCall DeployCall) (types.AccountID, error)
	SetEventDispatcher(contractAddressSS58 string, dispatcher map[types.Hash]ContractEventDispatchEntry) error
}

type Response struct {
	DebugMessage string `json:"debugMessage"`
	GasConsumed  int    `json:"gasConsumed"`
	Result       struct {
		Ok struct {
			Data  string `json:"data"`
			Flags int    `json:"flags"`
		} `json:"Ok"`
	} `json:"result"`
}

type Request struct {
	Origin    string `json:"origin"`
	Dest      string `json:"dest"`
	GasLimit  uint   `json:"gasLimit"`
	InputData string `json:"inputData"`
	Value     int    `json:"value"`
}

type DeployCall struct {
	Code     []byte
	Salt     []byte
	From     signature.KeyringPair
	Value    float64
	GasLimit float64
	Method   []byte
	Args     []interface{}
}

type ContractCall struct {
	ContractAddress     types.AccountID
	ContractAddressSS58 string
	From                signature.KeyringPair
	Value               float64
	GasLimit            float64
	Method              []byte
	Args                []interface{}
}
