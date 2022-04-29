package cid

import (
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPieceCid_Sha2_256(t *testing.T) {
	//given
	testSubject := CreateBuilder(cid.Raw, multihash.SHA2_256)
	expectedCid := "bafkreigaknpexyvxt76zgkitavbwx6ejgfheup5oybpm77f3pxzrvwpfdi"

	//when
	c, err := testSubject.Build([]byte("Hello world!"))

	//then
	assert.NoError(t, err)
	assert.Equal(t, expectedCid, c)
}

func TestGetPieceCid_Blake2b_256(t *testing.T) {
	//given
	testSubject := DefaultBuilder()
	expectedCid := "bafk2bzacea73ycjnxe2qov7cvnhx52lzfp6nf5jcblnfus6gqreh6ygganbws"

	//when
	c, err := testSubject.Build([]byte("Hello world!"))

	//then
	assert.NoError(t, err)
	assert.Equal(t, expectedCid, c)
}
