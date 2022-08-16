package utils

import "golang.org/x/crypto/blake2b"

func HashBlake2b256(data []byte) []byte {
	hash := blake2b.Sum256(data)
	return hash[:]
}
