package domain

import (
	"errors"

	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/crypto"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

// ## Generation of a SignedPiece:
//
// 1. Prepare a `Piece` structure and its ProtoBuf serialization.
// 2. Prepare a `Signature` structure with its details except `Value`.
// 3. Pass those to `NewSignedPiece(…)`.
// 4. Use `SigneableCid()` to generate a signeable message, and a CID of the piece.
// 5. Generate a signature of the `signeable` message using `crypto.CreateScheme(…)`.
// 6. Store the signature with `SetSignature()`.
// 7. Serialize the SignedPiece with `MarshalProto()` for transmission or storage.
// 8. Use the CID above as a permanent identifier of the piece.
//
// ## Verification of a SignedPiece:
//
// 1. Deserialize using `UnmarshalProto()`.
// 2. Call `Verify()`.
// 3. If the piece is to be forwarded or stored, use the original serialization (do not re-serialize).
//
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

func (sp *SignedPiece) PieceCid() (string, error) {
	return cid.CreateBuilder(sp.Signature.MultiHashType).Build(sp.PieceSerial())
}

func (sp *SignedPiece) SigneableCid() (signeable []byte, pieceCid string, err error) {
	pieceCid, err = sp.PieceCid()
	if err != nil {
		return nil, "", err
	}

	return []byte(pieceCid), pieceCid, nil
}

func (sp *SignedPiece) SetSignature(sig []byte) {
	sp.Signature.Value = sig
}

var ErrInvalidSignature = errors.New("invalid signature")

func (sp *SignedPiece) Verify() (pieceCid string, err error) {
	signeable, pieceCid, err := sp.SigneableCid()
	if err != nil {
		return "", err
	}

	ok, err := crypto.Verify(
		crypto.SchemeName(sp.Signature.Scheme),
		sp.Signature.Signer,
		signeable,
		sp.Signature.Value)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidSignature
	}

	return pieceCid, nil
}
