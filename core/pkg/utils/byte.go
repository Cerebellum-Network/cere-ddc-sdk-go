package utils

import "encoding/binary"

func Uint16ToBytes(number uint16) []byte {
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, number)
	return bytes
}

func Uint32ToBytes(number uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, number)
	return bytes
}

func BytesToUint16(bytes []byte) uint16 {
	return binary.BigEndian.Uint16(bytes)
}

func BytesToUint32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

func BytesToUint64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}

func Uint64ToBytes(number uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, number)
	return bytes
}

func Xor(left []byte, right []byte) []byte {
	var big, small []byte
	if len(left) >= len(right) {
		big, small = left, right
	} else {
		big, small = right, left
	}

	result := make([]byte, len(big))
	lastIndex := 0
	for i, v := range small {
		result[i] = v ^ big[i]
		lastIndex++
	}

	for i := lastIndex; i < len(big); i++ {
		result[i] ^= big[i]
	}

	return result
}
