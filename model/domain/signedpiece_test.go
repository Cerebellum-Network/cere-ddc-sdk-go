package domain

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"github.com/stretchr/testify/require"
)

type StoreRequest struct {
	Body  string
	Cid   string
	Piece struct {
		Data []byte
		Tags []struct {
			Searchable string
		}
	}
}

func TestStoreRequest(t *testing.T) {
	doTestStoreRequest(t, "../../ddc-schemas/test-vectors/store-request-sdk-js-1.1.0.json")
	doTestStoreRequest(t, "../../ddc-schemas/test-vectors/store-request-sdk-js-1.2.8.json")
	doTestStoreRequest(t, "../../ddc-schemas/test-vectors/store-request-sdk-js-1.2.9.json")
}

func doTestStoreRequest(t *testing.T, vectorFile string) {
	raw, err := os.ReadFile(vectorFile)
	require.NoError(t, err)

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
	require.Equal(t, request.Piece.Data, sp.Piece().Data)
	tagType := pb.SearchType_value[request.Piece.Tags[0].Searchable]
	require.Equal(t, tagType, int32(sp.Piece().Tags[0].Searchable))
}
