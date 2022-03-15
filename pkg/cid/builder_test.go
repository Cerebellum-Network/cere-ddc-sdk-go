package cid

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embedded asd

func TestGetPieceCid(t *testing.T) {
	//given
	testSubject := DefaultBuilder()
	expectedCid := "bafkreigaknpexyvxt76zgkitavbwx6ejgfheup5oybpm77f3pxzrvwpfdi"

	//when
	c, err := testSubject.Build([]byte("Hello world!"))

	//then
	assert.NoError(t, err)
	assert.Equal(t, expectedCid, c)
}
