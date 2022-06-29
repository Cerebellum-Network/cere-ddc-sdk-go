package domain

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const TEST_VECTOR = "../../ddc-schemas/test-vectors/store-request-sdk-js-1.2.8.json"

type StoreRequest struct {
	Body string
	Cid  string
}

func TestStoreRequest(t *testing.T) {
	raw, err := os.ReadFile(TEST_VECTOR)
	if err != nil {
		t.Skip("WARNING: no test vectors. Run: git submodule update --init")
	}

	request := StoreRequest{}
	err = json.Unmarshal(raw, &request)
	require.NoError(t, err)

	spSerial, err := hex.DecodeString(strings.TrimPrefix(request.Body, "0x"))
	require.NoError(t, err)

	sp := SignedPiece{}
	err = sp.UnmarshalProto(spSerial)
	require.NoError(t, err)

	cid, err := sp.Verify()
	require.NoError(t, err)

	require.Equal(t, request.Cid, cid)
}
