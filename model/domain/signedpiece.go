package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type SignedPiece struct {
	pieceSerial []byte
	Signature   *Signature

	piece *Piece
}

var _ Protobufable = (*SignedPiece)(nil)

func NewSignedPiece(piece *Piece, pieceSerial []byte, sig *Signature) *SignedPiece {
	return &SignedPiece{
		pieceSerial: pieceSerial,
		Signature:   sig,
		piece:       piece,
	}
}

func (sp *SignedPiece) ToProto() *pb.SignedPiece {
	return &pb.SignedPiece{
		Piece:     sp.pieceSerial,
		Signature: sp.Signature.ToProto(),
	}
}

func (sp *SignedPiece) ToDomain(pbSignedPiece *pb.SignedPiece) error {
	sp.pieceSerial = pbSignedPiece.Piece

	sp.Signature = &Signature{}
	sp.Signature.ToDomain(pbSignedPiece.Signature)

	sp.piece = &Piece{}
	return sp.piece.UnmarshalProto(sp.pieceSerial)
}

func (sp *SignedPiece) MarshalProto() ([]byte, error) {
	return proto.Marshal(sp.ToProto())
}

func (sp *SignedPiece) UnmarshalProto(signedPieceAsBytes []byte) error {
	pbSignedPiece := &pb.SignedPiece{}
	if err := proto.Unmarshal(signedPieceAsBytes, pbSignedPiece); err != nil {
		return err
	}

	return sp.ToDomain(pbSignedPiece)
}

func (sp *SignedPiece) PieceSerial() []byte {
	return sp.pieceSerial
}

func (sp *SignedPiece) Piece() *Piece {
	return sp.piece
}
