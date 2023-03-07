package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXorDouble(t *testing.T) {
	//given
	hashLeft := "bafkreigaknpexyvxt76zgkitavbwx6ejgfheup5oybpm77f3pxzrvwpfdi"
	hashRight := "avbwx6ejgfheup5oybpm77f3pxzrvwpfdibafkreigaknpexyvxt76zgkit"
	hashLeftAsBytes := []byte(hashLeft)
	hashRightAsBytes := []byte(hashRight)

	//when
	hash := Xor(hashLeftAsBytes, hashRightAsBytes)
	doubleHash := Xor(hash, hashRightAsBytes)

	//then
	assert.NotEqual(t, []byte(hashLeft), hash)
	assert.Equal(t, []byte(hashLeft), doubleHash)
}

func TestXorDifferentOrder(t *testing.T) {
	//given
	hash1 := []byte("bafkreigaknpexyvxt76zgkitavbwx6ejgfheup5oybpm77f3pxzrvwpfdi")
	hash2 := []byte("avbwx6ejgfheup5oybpm77f3pxzrvwpfdibafkreigaknpexyvxt76zgkit")
	hash3 := []byte("eup5oybpm77f3pxzrvwpfdibaavbwx6ejgfhfkreigaknpexyvxt76zgkit")

	//when
	hash := Xor(Xor(hash1, hash2), hash3)
	differentOrderHash := Xor(Xor(hash3, hash1), hash2)

	//then
	assert.Equal(t, hash, differentOrderHash)
}

func TestXorDifferentLength(t *testing.T) {
	//given
	hashLeft := "bafkreigaknpexyvxt76zgkitavbwx6ejgfheup5oybpm77f3pxzrvwpfdi"
	hashRight := "avbwx6ejgfheup5oybpm"
	hashLeftAsBytes := []byte(hashLeft)
	hashRightAsBytes := []byte(hashRight)

	//when
	hash := Xor(hashLeftAsBytes, hashRightAsBytes)

	//then
	assert.NotEqual(t, []byte(hashLeft), hash)
	assert.NotEqual(t, []byte(hashRight), hash)
	assert.Equal(t, len(hashLeft), len(hash))
}

func TestXorEmpty(t *testing.T) {
	//given
	hash1 := make([]byte, 20)
	hash2 := make([]byte, 20)

	//when
	hash := Xor(hash1, hash2)

	//then
	assert.Equal(t, hash1, hash)
	assert.Equal(t, hash2, hash)
}

func TestXorNil(t *testing.T) {
	//given
	hash := []byte("bafkreigaknpexyvxt76zgkitavbwx6ejgfheup5oybpm77f3pxzrvwpfdi")

	//when
	result := Xor(nil, hash)

	//then
	assert.Equal(t, hash, result)
}
