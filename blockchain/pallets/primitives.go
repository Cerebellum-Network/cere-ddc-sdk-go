// Primitive types definition of the Cere Network blockchain.
//
// Find more about SCALE encoding/decoding for enums here: https://github.com/centrifuge/go-substrate-rpc-client/blob/0a28e8b/types/example_enum_test.go#L27.
//
// Based on [ddc-primitives@0.1.0].
//
// [ddc-primitives@0.1.0]: https://github.com/Cerebellum-Network/blockchain-node/tree/d970a49/primitives.
package pallets

import (
	"errors"
	"reflect"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	ErrUnknownVariant = errors.New("unknown variant")
)

const (
	NodeTypeStorage = 1
)

type (
	BucketId          = types.U64
	ClusterId         = types.H160
	DdcEra            = types.U32
	StorageNodePubKey = types.AccountID
)

type NodePubKey struct {
	IsStoragePubKey bool
	AsStoragePubKey StorageNodePubKey
}

func (m *NodePubKey) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	if b == 0 {
		m.IsStoragePubKey = true
		err = decoder.Decode(&m.AsStoragePubKey)
	} else {
		return ErrUnknownVariant
	}

	if err != nil {
		return err
	}

	return nil
}

func (m NodePubKey) Encode(encoder scale.Encoder) error {
	var err1, err2 error
	if m.IsStoragePubKey {
		err1 = encoder.PushByte(0)
		err2 = encoder.Encode(m.AsStoragePubKey)
	} else {
		return ErrUnknownVariant
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

type StorageNodeMode struct {
	IsFull    bool
	IsStorage bool
	IsCache   bool
}

func (m *StorageNodeMode) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	if b == 1 {
		m.IsFull = true
	} else if b == 2 {
		m.IsStorage = true
	} else if b == 3 {
		m.IsCache = true
	} else {
		return ErrUnknownVariant
	}

	return nil
}

func (m StorageNodeMode) Encode(encoder scale.Encoder) error {
	var err error
	if m.IsFull {
		err = encoder.PushByte(1)
	} else if m.IsStorage {
		err = encoder.PushByte(2)
	} else if m.IsCache {
		err = encoder.PushByte(3)
	} else {
		return ErrUnknownVariant
	}

	if err != nil {
		return err
	}

	return nil
}

type ClusterStatus struct {
	IsUnbonded  bool
	IsBonded    bool
	IsActivated bool
	IsUnbonding bool
}

func (m *ClusterStatus) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	i := int(b)

	v := reflect.ValueOf(m)
	if i > v.NumField() {
		return ErrUnknownVariant
	}

	v.Field(i).SetBool(true)

	return nil
}

func (m ClusterStatus) Encode(encoder scale.Encoder) error {
	var err1, err2 error
	v := reflect.ValueOf(m)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Bool() {
			err1 = encoder.PushByte(byte(i))
			err2 = encoder.Encode(i + 1) // values are defined from 1
			break
		}
		if i == v.NumField()-1 {
			return ErrUnknownVariant
		}
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
