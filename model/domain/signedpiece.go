package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type SignedPiece struct {
	Piece     *Piece
	Signature *Signature
}

var _ Protobufable = (*SignedPiece)(nil)

func (sp *SignedPiece) ToProto() *pb.SignedPiece {
	return &pb.SignedPiece{
		Piece:     sp.Piece.ToProto(),
		Signature: sp.Signature.ToProto(),
	}
}

func (sp *SignedPiece) ToDomain(pbSignedPiece *pb.SignedPiece) {
	sp.Piece = &Piece{}
	sp.Piece.ToDomain(pbSignedPiece.Piece)

	sp.Signature = &Signature{}
	sp.Signature.ToDomain(pbSignedPiece.Signature)
}

func (sp *SignedPiece) MarshalProto() ([]byte, error) {
	return proto.Marshal(sp.ToProto())
}

func (sp *SignedPiece) UnmarshalProto(signedPieceAsBytes []byte) error {
	pbSignedPiece := &pb.SignedPiece{}
	if err := proto.Unmarshal(signedPieceAsBytes, pbSignedPiece); err != nil {
		return err
	}

	sp.ToDomain(pbSignedPiece)
	return nil
}
