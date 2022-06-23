package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestSignedPiece(t *testing.T) {
	piece := Piece{BucketId: 7, Data: []byte{8, 9}}
	pieceSerial, err := piece.MarshalProto()
	require.NoError(t, err)
	sig := Signature{Value: "abc"}

	sp := NewSignedPiece(&piece, pieceSerial, &sig)
	spSerial, err := sp.MarshalProto()
	require.NoError(t, err)

	require.Equal(t,
		[]byte{
			// Fixed prefix for the DDC namespace.
			0xD0, 0x0C,
			// DDC message type 1.
			0x01,
			// Protobuf encoding.
			0xa, 0x6, 0xa, 0x2, 0x8, 0x9, 0x10, 0x7, 0x12, 0x5, 0xa, 0x3, 0x61, 0x62, 0x63},
		spSerial)

	// Deserialize ok with the right type code.
	sp2 := &SignedPiece{}
	err = sp2.UnmarshalProto(spSerial)
	require.NoError(t, err)
	require.Equal(t, sp.PieceSerial(), sp2.PieceSerial())

	// Deserialize ok without any type code.
	sp3 := &SignedPiece{}
	err = sp3.UnmarshalProto(spSerial[3:])
	require.NoError(t, err)
	require.Equal(t, sp.PieceSerial(), sp3.PieceSerial())

	// Check the format of the type code as a protobuf field.
	tagNum, tagType, tagSize := protowire.ConsumeTag(spSerial)
	require.Equal(t, protowire.VarintType, tagType)
	require.Equal(t, protowire.Number(202), tagNum) // Fixed field for message types.
	require.Equal(t, 2, tagSize)
	msgType, _ := protowire.ConsumeVarint(spSerial[tagSize:])
	require.Equal(t, DDTYPE_SIGNED_PIECE, msgType) // The message type.

	// Reject an unexpected type code.
	spSerial[2] = 9
	sp4 := &SignedPiece{}
	err = sp4.UnmarshalProto(spSerial)
	require.EqualError(t, err, "invalid message type (9)")
}
