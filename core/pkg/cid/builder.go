package cid

import (
	"github.com/ipfs/go-cid"
)

const (
	Blake2b256   = 0xb220
	defaultCodec = cid.Raw
)

type Builder struct {
	cidBuilder cid.V1Builder
}

func CreateBuilder(mhType uint64) *Builder {
	if mhType == 0 {
		mhType = Blake2b256
	}
	return &Builder{cidBuilder: cid.V1Builder{Codec: defaultCodec, MhType: mhType}}
}

func (b *Builder) Build(data []byte) (string, error) {
	c, err := b.cidBuilder.Sum(data)
	if err != nil {
		return "", err
	}

	return c.String(), nil
}
