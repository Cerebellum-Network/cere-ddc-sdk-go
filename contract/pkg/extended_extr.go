package pkg

import (
	"context"
	"fmt"
	"github.com/centrifuge/go-substrate-rpc-client/v2/config"
	gethrpc "github.com/centrifuge/go-substrate-rpc-client/v2/gethrpc"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	gsrpctypes "github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/patractlabs/go-patract/api"
	"github.com/pkg/errors"
	"sync"
)

func (c *contract) callToExec(ctx api.Context, contractID types.AccountID, value types.UCompact, gasLimit types.UCompact, call []string, args ...interface{}) (types.Hash, error) {
	data, err := c.Contract.GetMessageData(call, args...)
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "getMessagesData")
	}

	return c.submitAndWaitExtrinsic(ctx, "Contracts.call", gsrpctypes.NewAddressFromAccountID(contractID[:]), value, gasLimit, data)
}

func (c *contract) submitAndWaitExtrinsic(ctx api.Context, call string, args ...interface{}) (types.Hash, error) {
	rpcApi := c.Contract.Native().Cli.API()
	meta, err := rpcApi.RPC.State.GetMetadataLatest()
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "get metadata lastest error")
	}

	cc, err := types.NewCall(meta, call, args...)
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "new call error")
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(cc)

	genesisHash, err := rpcApi.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "get block hash error")
	}

	rv, err := rpcApi.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "get runtime version lastest error")
	}

	authKey := ctx.From()

	key, err := types.CreateStorageKey(meta, "System", "Account", authKey.PublicKey, nil)
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "create storage key error")
	}

	var accountInfo types.AccountInfo
	ok, err := rpcApi.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return types.Hash{}, errors.Wrapf(err, "create storage key error by %s", authKey.Address)
	} else if !ok {
		return types.Hash{}, errors.Errorf("no accountInfo found by %s", authKey.Address)
	}

	nonce := uint32(accountInfo.Nonce)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	if err := c.sign(&ext, authKey, o); err != nil {
		return types.Hash{}, errors.Wrap(err, "sign extrinsic error")
	}

	// Send the extrinsic
	sub, err := c.submitAndWatchExtr(ext)
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "submit error")
	}
	defer func() {
		sub.sub.Unsubscribe()
		sub.quitOnce.Do(func() {
			close(sub.channel)
		})
	}()

	for {
		select {
		case status := <-sub.channel:
			if status.IsInBlock {
				return status.AsInBlock, nil
			}
		case err := <-sub.sub.Err():
			return types.Hash{}, errors.Wrap(err, "subscribe error")
		case <-ctx.Done():
			return types.Hash{}, ctx.Err()
		}
	}
}

func (c *contract) sign(e *types.Extrinsic, signer signature.KeyringPair, o types.SignatureOptions) error {
	if e.Type() != types.ExtrinsicVersion4 {
		return fmt.Errorf("unsupported extrinsic version: %v (isSigned: %v, type: %v)", e.Version, e.IsSigned(), e.Type())
	}

	mb, err := types.EncodeToBytes(e.Method)
	if err != nil {
		return err
	}

	era := o.Era
	if !o.Era.IsMortalEra {
		era = types.ExtrinsicEra{IsImmortalEra: true}
	}

	payload := types.ExtrinsicPayloadV4{
		ExtrinsicPayloadV3: types.ExtrinsicPayloadV3{
			Method:      mb,
			Era:         era,
			Nonce:       o.Nonce,
			Tip:         o.Tip,
			SpecVersion: o.SpecVersion,
			GenesisHash: o.GenesisHash,
			BlockHash:   o.BlockHash,
		},
		TransactionVersion: o.TransactionVersion,
	}

	signerPubKey := types.NewMultiAddressFromAccountID(signer.PublicKey)

	sig, err := prepareSignature(payload, signer)
	if err != nil {
		return err
	}

	extSig := types.ExtrinsicSignatureV4{
		Signer:    signerPubKey,
		Signature: types.MultiSignature{IsSr25519: true, AsSr25519: sig},
		Era:       era,
		Nonce:     o.Nonce,
		Tip:       o.Tip,
	}

	e.Signature = extSig

	// mark the extrinsic as signed
	e.Version |= types.ExtrinsicBitSigned

	return nil
}

func prepareSignature(e types.ExtrinsicPayloadV4, signer signature.KeyringPair) (types.Signature, error) {
	b, err := types.EncodeToBytes(e)
	if err != nil {
		return types.Signature{}, err
	}

	sig, err := Sign(b, signer.URI)
	return types.NewSignature(sig), err
}

type ExtrinsicStatusSubscription struct {
	sub      *gethrpc.ClientSubscription
	channel  chan types.ExtrinsicStatus
	quitOnce sync.Once
}

func (c *contract) submitAndWatchExtr(xt types.Extrinsic) (*ExtrinsicStatusSubscription, error) { //nolint:lll
	ctx, cancel := context.WithTimeout(context.Background(), config.Default().SubscribeTimeout)
	defer cancel()

	chanel := make(chan types.ExtrinsicStatus)

	enc, err := types.EncodeToHexString(xt)
	if err != nil {
		return nil, err
	}

	sub, err := c.Contract.Native().Cli.API().Client.Subscribe(ctx, "author", "submitAndWatchExtrinsic", "unwatchExtrinsic", "extrinsicUpdate",
		chanel, enc)
	if err != nil {
		return nil, err
	}

	return &ExtrinsicStatusSubscription{sub: sub, channel: chanel}, nil
}
