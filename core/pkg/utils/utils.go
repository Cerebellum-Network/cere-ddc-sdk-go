package utils

import (
	"crypto/rand"
	"errors"
	"golang.org/x/crypto/blake2b"
	"math/big"
)

func RemoveSorted[T any](slice []T, i int) []T {
	if i < 0 || len(slice) <= i {
		return slice
	}

	return append(slice[:i], slice[i+1:]...)[: len(slice)-1 : len(slice)-1]
}

func IsSuccessHttpStatus(code int) bool {
	return code >= 200 && code < 300
}

func HashBlake2b256(data []byte) []byte {
	hash := blake2b.Sum256(data)
	return hash[:]
}

func RandomInt64(max int64) (int64, error) {
	if max <= 0 {
		return -1, errors.New("crypto/rand: argument to max int64 is <= 0")
	}

	random, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return -1, err
	}

	return random.Int64(), nil
}
