package cid

import (
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

const (
	defaultMhType   = multihash.BLAKE2B_MAX - 32
	defaultMhLength = 32
	defaultCodec    = cid.Raw
)

type Builder struct {
	cidBuilder cid.V1Builder
}

func CreateBuilder(codec uint64, mhType uint64) *Builder {
	return &Builder{cidBuilder: cid.V1Builder{Codec: codec, MhType: mhType}}
}

func DefaultBuilder() *Builder {
	return &Builder{cidBuilder: cid.V1Builder{Codec: defaultCodec, MhType: defaultMhType, MhLength: defaultMhLength}}
}

func (b *Builder) Build(data []byte) (string, error) {
	c, err := b.cidBuilder.Sum(data)
	if err != nil {
		return "", err
	}

	return c.String(), nil
}
