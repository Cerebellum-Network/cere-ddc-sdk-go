package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/patractlabs/go-patract/api"
	"github.com/patractlabs/go-patract/metadata"
	"github.com/patractlabs/go-patract/rpc"
	"github.com/pkg/errors"
)

type Contract interface {
	CallToRead(ctx api.Context, result interface{}, call []string, args ...interface{}) error
	CallToReadEncoded(ctx api.Context, call []string, args ...interface{}) (string, error)
	GetAccountIDSS58() string
}

type contract struct {
	*rpc.Contract

	accountIDSS58 string
	metadata      *metadata.Data
}

type NopLogger struct{}

func (l *NopLogger) Flush()                       {}
func (l *NopLogger) Debug(string, ...interface{}) {}
func (l *NopLogger) Info(string, ...interface{})  {}
func (l *NopLogger) Warn(string, ...interface{})  {}
func (l *NopLogger) Error(string, ...interface{}) {}

func CreateContract(rpcContract *rpc.Contract, accountIDSS58 string, metadata *metadata.Data) Contract {
	contract := &contract{
		Contract:      rpcContract,
		accountIDSS58: accountIDSS58,
		metadata:      metadata,
	}
	contract.WithLogger(&NopLogger{})
	return contract
}

func (c *contract) CallToRead(ctx api.Context, result interface{}, call []string, args ...interface{}) error {
	params := struct {
		Origin    string `json:"origin"`
		Dest      string `json:"dest"`
		GasLimit  uint   `json:"gasLimit"`
		InputData string `json:"inputData"`
		Value     int    `json:"value"`
	}{
		Origin:   ctx.From().Address,
		Dest:     c.accountIDSS58,
		GasLimit: rpc.DefaultGasLimitForCall,
	}

	data, err := c.GetMessageData(call, args...)
	if err != nil {
		return errors.Wrap(err, "getMessagesData")
	}

	params.InputData = types.HexEncodeToString(data)

	res := struct {
		Success struct {
			Data  string `json:"data"`
			Flags int    `json:"flags"`
		} `json:"success"`
	}{}

	err = c.Native().Cli.Call(&res, "contracts_call", params)
	if err != nil {
		return errors.Wrap(err, "call")
	}

	message, err := c.metadata.Raw.GetMessage(call)
	if err != nil {
		return err
	}

	bz, err := types.HexDecodeString(res.Success.Data)
	if err != nil {
		return errors.Wrap(err, "hex from string error")
	}

	if len(bz) == 0 {
		return errors.Errorf("no data got")
	}

	err = c.metadata.Decode(result, message.ReturnType, bz)
	return errors.Wrapf(err, "decode error %s.", res.Success.Data)
}

func (c *contract) CallToReadEncoded(ctx api.Context, call []string, args ...interface{}) (string, error) {
	params := struct {
		Origin    string `json:"origin"`
		Dest      string `json:"dest"`
		GasLimit  uint   `json:"gasLimit"`
		InputData string `json:"inputData"`
		Value     int    `json:"value"`
	}{
		Origin:   ctx.From().Address,
		Dest:     c.accountIDSS58,
		GasLimit: rpc.DefaultGasLimitForCall,
	}

	data, err := c.GetMessageData(call, args...)
	if err != nil {
		return "", errors.Wrap(err, "getMessagesData")
	}

	params.InputData = types.HexEncodeToString(data)

	res := struct {
		Success struct {
			Data  string `json:"data"`
			Flags int    `json:"flags"`
		} `json:"success"`
	}{}

	err = c.Native().Cli.Call(&res, "contracts_call", params)
	if err != nil {
		return "", errors.Wrap(err, "call")
	}

	return res.Success.Data, nil
}

func (c *contract) GetAccountIDSS58() string {
	return c.accountIDSS58
}
