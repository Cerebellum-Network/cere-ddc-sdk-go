package domain

import (
	"bytes"
	"encoding/hex"
	"time"
)

// decodeHex decodes a hex-encoded string and trim 0x prefix if needed
func decodeHex(src []byte) ([]byte, error) {
	src = bytes.TrimPrefix(src, []byte("0x"))
	decoded := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(decoded, src)
	return decoded, err
}

// encodeHex encodes a string to hex and add 0x prefix
func encodeHex(src []byte) []byte {
	res := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(res, src)
	return append([]byte("0x"), res...)
}

// Format the time the same way as JavaScript Date.toISOString()
func formatTimestamp(unixMilli uint64) string {
	return time.UnixMilli(int64(unixMilli)).UTC().Format("2006-01-02T15:04:05.000Z")
}
