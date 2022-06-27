package cid

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPieceCid(t *testing.T) {
	doTestGetPieceCid(t, Blake2b256, "Hello world!", "bafk2bzacea73ycjnxe2qov7cvnhx52lzfp6nf5jcblnfus6gqreh6ygganbws")
	doTestGetPieceCid(t, 0, "Hello world!", "bafk2bzacea73ycjnxe2qov7cvnhx52lzfp6nf5jcblnfus6gqreh6ygganbws")
}

func doTestGetPieceCid(t *testing.T, mhType uint64, data string, expectedCid string) {
	//given
	testSubject := CreateBuilder(mhType)

	//when
	c, err := testSubject.Build([]byte(data))

	//then
	assert.NoError(t, err)
	assert.Equal(t, expectedCid, c)
}
