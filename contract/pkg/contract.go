package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/patractlabs/go-patract/api"
	"github.com/patractlabs/go-patract/metadata"
	"github.com/patractlabs/go-patract/rpc"
	"github.com/patractlabs/go-patract/utils"
	logger "github.com/patractlabs/go-patract/utils/log"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const CERE = 10_000_000_000

type Contract interface {
	CallToRead(ctx api.Context, result interface{}, call []string, args ...interface{}) error
	CallToReadEncoded(ctx api.Context, call []string, args ...interface{}) (string, error)
	CallToExec(ctx api.Context, value float64, gasLimit float64, call []string, args ...interface{}) (types.Hash, error)
	GetAccountIDSS58() string
}

type contract struct {
	*rpc.Contract

	accountIDSS58 string
	account       types.AccountID
	metadata      *metadata.Data
}

type NopLogger struct{}

func (l *NopLogger) Flush()                       {}
func (l *NopLogger) Debug(string, ...interface{}) {}
func (l *NopLogger) Info(string, ...interface{})  {}
func (l *NopLogger) Warn(string, ...interface{})  {}
func (l *NopLogger) Error(string, ...interface{}) {}

func CreateContract(rpcContract *rpc.Contract, accountIDSS58 string, metadata *metadata.Data) Contract {
	account, err := utils.DecodeAccountIDFromSS58(accountIDSS58)
	if err != nil {
		log.WithError(err).WithField("accountIDSS58", accountIDSS58).Fatal("Can't decode accountIDSS58")
	}
	contract := &contract{
		Contract:      rpcContract,
		accountIDSS58: accountIDSS58,
		metadata:      metadata,
		account:       account,
	}
	contract.WithLogger(logger.NewLogger())

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

func (c *contract) CallToExec(ctx api.Context, value float64, gasLimit float64, call []string, args ...interface{}) (types.Hash, error) {
	valueRaw := types.NewUCompactFromUInt(uint64(value * CERE))

	var gasLimitRaw types.UCompact
	if gasLimit < 0 {
		gasLimitRaw = types.NewUCompactFromUInt(uint64(gasLimit * CERE))
	} else {
		gasLimitRaw = types.NewUCompactFromUInt(rpc.DefaultGasLimitForCall)
	}

	hash, err := c.callToExec(ctx, c.account, valueRaw, gasLimitRaw, call, args...)
	return hash, err
}

func (c *contract) GetAccountIDSS58() string {
	return c.accountIDSS58
}
