package domain

import (
	"encoding/hex"
	"errors"
	"fmt"

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

	timeText := "" // TODO
	msg := fmt.Sprintf("<Bytes>DDC store %s at %s</Bytes>", pieceCid, timeText)

	return []byte(msg), pieceCid, nil
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

	sig, signer, isV013 := sp.getSigAndSigner()
	if isV013 {
		// Verify the message from v0.1.3. TODO: Remove this.
		signeable = []byte(pieceCid)
	}

	ok, err := crypto.Verify(
		crypto.SchemeName(sp.Signature.Scheme),
		signer,
		signeable,
		sig)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidSignature
	}

	return pieceCid, nil
}

func (sp *SignedPiece) getSigAndSigner() (sig []byte, signer []byte, isV013 bool) {

	// Try the deprecated hexadecimal format. TODO: Remove this.
	sigFromHex := maybeDecodeHex(sp.Signature.Value, 64)
	signerFromHex := maybeDecodeHex(sp.Signature.Signer, 32)
	if sigFromHex != nil && signerFromHex != nil {
		return sigFromHex, signerFromHex, true
	}

	return sp.Signature.Value, sp.Signature.Signer, false
}

func maybeDecodeHex(src []byte, bytesLen int) (maybeDecoded []byte) {
	hexLen := 2 + bytesLen*2
	if len(src) == hexLen && string(src[:2]) == "0x" {
		decoded := make([]byte, bytesLen)
		_, err := hex.Decode(decoded, src[2:])
		if err != nil {
			return nil
		}
		return decoded
	}
	return nil
}
