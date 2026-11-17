package cid_test

import (
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cid"
)

const testCid = "baeab4ihbfh24fq4q2p4mzsmibrzwiww2efitqzerctyexuzt6zrbp6yfbm"

func TestEncodeCid(t *testing.T) {
	id, err := cid.Sum(cid.DefaultParams(0), cid.MhTypeBlake3, []byte("test-cid"))
	require.NoError(t, err)

	encodedCid := id.String()
	require.NoError(t, err)
	require.Equal(t, testCid, encodedCid)
}

func TestDecodeCid(t *testing.T) {
	id, err := cid.DecodeString(testCid)
	require.NoError(t, err)
	require.Equal(t, id.ContentType, cid.ContentType(0))
	require.Equal(t, id.MBase, cid.MBase32)
	require.Equal(t, id.Version, cid.ProtocolV1)
}

func TestEncodeDecodeCid(t *testing.T) {
	contentType := cid.ContentType(mrand.Int63())
	id, err := cid.Sum(cid.DefaultParams(contentType), cid.MhTypeBlake3, []byte("test-cid"))
	require.NoError(t, err)

	encodedCid := id.String()
	require.NoError(t, err)

	decodedCid, err := cid.DecodeString(encodedCid)
	require.NoError(t, err)
	require.Equal(t, decodedCid.MBase, cid.MBase32)
	require.Equal(t, decodedCid.ContentType, contentType)
	require.Equal(t, decodedCid.Version, cid.ProtocolV1)
}
