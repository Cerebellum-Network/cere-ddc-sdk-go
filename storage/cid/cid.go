package cid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	"github.com/multiformats/go-varint"
)

var (
	ErrInvalidCid                = errors.New("invalid cid")
	ErrUnknownProtocolVersion    = errors.New("unknown protocol version")
	ErrInvalidContentType        = errors.New("invalid content type")
	ErrNotSupportedHashAlgorithm = errors.New("not supported hash algorithm")
)

type (
	ProtocolVersion uint8
	ContentType     uint64
	MhType          uint64
	MBase           mbase.Encoding
)

const (
	ProtocolV1 ProtocolVersion = 0x01

	MhTypeNone   MhType = mh.BLAKE3
	MhTypeBlake3 MhType = mh.BLAKE3
	MBase32      MBase  = mbase.Base32
	MBaseNone    MBase  = mbase.Identity

	ContentTypeNone      ContentType = 0x00
	ContentTypeDagPb     ContentType = 0x01
	ContentTypeRaw       ContentType = 0x02
	ContentTypeMultipart ContentType = 0x03
)

type Params struct {
	Version     ProtocolVersion
	MBase       MBase
	ContentType ContentType
}

type Cid struct {
	Params
	Mhash   mh.Multihash
	MhType  MhType
	RawHash []byte
	bytes   []byte
}

func DefaultParams(contentType ContentType) Params {
	return Params{
		Version:     ProtocolV1,
		MBase:       MBase32,
		ContentType: contentType,
	}
}

func FromMultiHash(params Params, mhash mh.Multihash) (*Cid, error) {
	decodedMhash, err := mh.Decode(mhash)
	if err != nil {
		return nil, err
	}
	cid := &Cid{
		Params:  params,
		Mhash:   mhash,
		MhType:  MhType(decodedMhash.Code),
		RawHash: decodedMhash.Digest,
	}
	cid.setBytes()
	return cid, nil
}

func FromRawHash(params Params, rawHash []byte, mhType MhType) *Cid {
	mhash, _ := mh.Encode(rawHash, uint64(mhType))
	cid := &Cid{
		Params:  params,
		Mhash:   mhash,
		MhType:  mhType,
		RawHash: rawHash,
	}
	cid.setBytes()
	return cid
}

func Sum(params Params, mhType MhType, data []byte) (*Cid, error) {
	mhash, err := mh.Sum(data, uint64(mhType), -1)
	if err != nil {
		return nil, err
	}
	decodedMhash, err := mh.Decode(mhash)
	if err != nil {
		return nil, err
	}
	cid := &Cid{
		Params:  params,
		Mhash:   mhash,
		MhType:  MhType(decodedMhash.Code),
		RawHash: decodedMhash.Digest,
	}
	cid.setBytes()
	return cid, err
}

// SumRawPiece is Sum for data DDC stores whole as one raw piece with no
// caller-supplied id: the BLAKE3 tree root over its blocks, which is the BLAKE3
// hash of data.
func SumRawPiece(data []byte) (*Cid, error) {
	return Sum(DefaultParams(ContentTypeRaw), MhTypeBlake3, data)
}

func DecodeString(cidStr string) (*Cid, error) {
	mbaseEncoding, rawCid, err := mbase.Decode(cidStr)
	if err != nil {
		return nil, err
	}

	cid, err := Decode(rawCid)
	if err == nil {
		cid.MBase = MBase(mbaseEncoding)
	}
	return cid, err
}

func Decode(rawCid []byte) (*Cid, error) {
	buf := bytes.NewBuffer(rawCid)
	version, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	contentType, err := varint.ReadUvarint(buf)
	if err != nil {
		return nil, err
	}

	mhash, err := mh.Cast(buf.Bytes())
	if err != nil {
		return nil, err
	}

	decodedMhash, err := mh.Decode(mhash)
	if err != nil {
		return nil, err
	}

	return &Cid{
		Params: Params{
			Version:     ProtocolVersion(version),
			ContentType: ContentType(contentType),
			MBase:       MBase32,
		},
		Mhash:   mhash,
		MhType:  MhType(decodedMhash.Code),
		RawHash: decodedMhash.Digest,
		bytes:   rawCid,
	}, nil
}

func Validate(id *Cid) error {
	if id.Version != ProtocolV1 {
		return ErrUnknownProtocolVersion
	}

	if id.MhType != MhTypeBlake3 {
		return ErrNotSupportedHashAlgorithm
	}

	return nil
}

func DecodeAndValidate(rawCid []byte, types ...ContentType) (*Cid, error) {
	id, err := Decode(rawCid)
	if err != nil {
		return nil, ErrInvalidCid
	}
	if err = Validate(id); err != nil {
		return nil, err
	}

	if !slices.Contains(types, id.ContentType) {
		return nil, ErrInvalidContentType
	}

	return id, nil
}

func (cid *Cid) Bytes() []byte {
	return cid.bytes
}

func (cid *Cid) setBytes() {
	hashlen := len(cid.Mhash)

	var buf bytes.Buffer
	buf.Grow(1 + varint.UvarintSize(uint64(cid.ContentType)) + hashlen)

	buf.WriteByte(byte(cid.Version))

	var rawCodec [binary.MaxVarintLen64]byte
	n := varint.PutUvarint(rawCodec[:], uint64(cid.ContentType))
	buf.Write(rawCodec[:n])
	buf.Write(cid.Mhash)

	cid.bytes = buf.Bytes()
}

func (cid *Cid) String() string {
	// error can be ignored assuming that MBase is always set to valid value
	id, _ := mbase.Encode(mbase.Encoding(cid.MBase), cid.Bytes())
	return id
}

func ToBase32(rawCid []byte) string {
	id, _ := mbase.Encode(mbase.Encoding(MBase32), rawCid)
	return id
}
