// Primitive types definition of the Cere Network blockchain.
//
// Find more about SCALE encoding/decoding for enums here: https://github.com/centrifuge/go-substrate-rpc-client/blob/0a28e8b/types/example_enum_test.go#L27.
//
// Based on [ddc-primitives@0.1.0].
//
// [ddc-primitives@0.1.0]: https://github.com/Cerebellum-Network/blockchain-node/tree/896f5d3/primitives.
package pallets

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

const (
	NodeTypeStorage = 1
	NodeTypeCdn     = 2
)

type ClusterId = types.H160
type DdcEra = types.U32
type BucketId = types.U64
type StorageNodePubKey = types.AccountID
type CdnNodePubKey = types.AccountID

type NodePubKey struct {
	AsCdnPubKey     CdnNodePubKey
	AsStoragePubKey StorageNodePubKey
	IsCdnPubKey     bool
	IsStoragePubKey bool
}

func (m *NodePubKey) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	if b == 0 {
		m.IsStoragePubKey = true
		err = decoder.Decode(&m.AsStoragePubKey)
	} else if b == 1 {
		m.IsCdnPubKey = true
		err = decoder.Decode(&m.AsCdnPubKey)
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
	} else if m.IsCdnPubKey {
		err1 = encoder.PushByte(1)
		err2 = encoder.Encode(m.AsCdnPubKey)
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
