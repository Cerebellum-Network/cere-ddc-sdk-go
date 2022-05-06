package cid

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPieceCid_Sha2_256(t *testing.T) {
	//given
	testSubject := CreateBuilder(BLAKE2B_256)
	expectedCid := "bafk2bzacea73ycjnxe2qov7cvnhx52lzfp6nf5jcblnfus6gqreh6ygganbws"

	//when
	c, err := testSubject.Build([]byte("Hello world!"))

	//then
	assert.NoError(t, err)
	assert.Equal(t, expectedCid, c)
}
