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
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
