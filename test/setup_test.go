package test

import (
	"context"
	"encoding/hex"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/sdktypes"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
	"testing"
)

type ApplicationTestSuite struct {
	ctx    context.Context
	cancel context.CancelFunc
	suite.Suite
	composeStack tc.ComposeStack
	useExisting  bool
}

func (a *ApplicationTestSuite) SetupSuite() {
	t := a.T()

	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancel = cancel

	_, a.useExisting = os.LookupEnv("CERE_USE_EXISTING_CHAIN")
	if !a.useExisting {
		compose, err := tc.NewDockerCompose("docker-compose.yml")
		a.Require().NoError(err)

		a.composeStack = compose

		err = compose.
			WaitForService("cere-sdk-test-chain",
				wait.
					ForLog("Running JSON-RPC WS server:"),
			).
			Up(ctx)
		assert.NoError(t, err, "compose.Up()")
		t.Cleanup(func() {
			assert.NoError(t, a.composeStack.Down(a.ctx, tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
		})
	}
}

func TestApplicationSuite(t *testing.T) {
	suite.Run(t, new(ApplicationTestSuite))
}

func (a *ApplicationTestSuite) deployBucketContract(client sdktypes.BlockchainClient) (*types.AccountID, error) {
	methodId, err := hex.DecodeString("9bae9d5e") // label: "new"
	if err != nil {
		logrus.WithError(err).Fatal("Can't decode method")
	}
	code, err := os.ReadFile("ddc_bucket.wasm")
	if err != nil {
		return nil, err
	}

	call := sdktypes.DeployCall{
		Code:     code,
		Salt:     []byte(uuid.New().String()),
		From:     signature.TestKeyringPairAlice,
		Value:    0,
		GasLimit: 10,
		Method:   methodId,
		Args:     []interface{}{},
	}

	contract, err := client.Deploy(a.ctx, call)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Contract deployed: %s", contract.ToHexString())

	return &contract, nil
}

func (a *ApplicationTestSuite) bucketSetAvailability(contractAddress string, client sdktypes.BlockchainClient, ctx context.Context, bucketId bucket.BucketId, avail bool) (string, error) {

	methodId, err := hex.DecodeString("053eb3ce")
	if err != nil {
		logrus.WithError(err).Fatal("Can't decode method")
	}
	c, err := utils.DecodeAccountIDFromSS58(contractAddress)
	if err != nil {
		return "", err
	}
	arg1 := types.U32(bucketId)
	arg2 := types.NewBool(avail)
	call := sdktypes.ContractCall{
		ContractAddress:     c,
		ContractAddressSS58: contractAddress,
		From:                signature.TestKeyringPairAlice,
		Value:               0,
		GasLimit:            9,
		Method:              methodId,
		Args: []interface{}{
			arg1,
			arg2,
		},
	}
	blockHash, err := client.CallToExec(ctx, call)
	if err != nil {
		return "", err
	}
	logrus.Infof("BucketSetAvailability tx hash: %s", blockHash.Hex())
	return blockHash.Hex(), nil
}

func (a *ApplicationTestSuite) bucketCreate(contractAddress string, client sdktypes.BlockchainClient, ctx context.Context) (string, error) {
	methodId, err := hex.DecodeString("0aeb2379")
	if err != nil {
		logrus.WithError(err).Fatal("Can't decode method")
	}
	c, err := utils.DecodeAccountIDFromSS58(contractAddress)
	if err != nil {
		return "", err
	}

	bucketParams := "ABC"
	//bucketParamsBytes := types.NewData([]byte{0x0c, 0x41, 0x42, 0x43, 0x00})
	// We didn't find how to pass this params in other way
	initialByte := []byte{0x0c}
	finalByte := []byte{0x00}
	bucketParamsBytes := types.NewData(append(append(initialByte, []byte(bucketParams)...), finalByte...))

	clusterId := types.U32(0)

	alice := signature.TestKeyringPairAlice

	call := sdktypes.ContractCall{
		ContractAddress:     c,
		ContractAddressSS58: contractAddress,
		From:                alice,
		Value:               0,
		GasLimit:            9,
		Method:              methodId,
		Args: []interface{}{
			bucketParamsBytes,
			clusterId,
		},
	}
	blockHash, err := client.CallToExec(ctx, call)
	if err != nil {
		return "", err
	}

	logrus.Infof("BucketCreate tx hash: %s", blockHash.Hex())
	return blockHash.Hex(), nil
}
