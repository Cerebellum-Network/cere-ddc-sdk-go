package utils

import "hash/crc64"

var crc64Table = crc64.MakeTable(crc64.ECMA)

func CidToToken(cid string) uint64 {
	return crc64.Checksum([]byte(cid), crc64Table)
}
