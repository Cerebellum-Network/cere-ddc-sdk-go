package domain

import (
	domain "github.com/cerebellum-network/cere-ddc-sdk-go/model"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Piece struct {
	Data     []byte
	BucketId uint32
	Tags     []*Tag
	Links    []*Link
}

var _ domain.Protobufable = (*Piece)(nil)

func (p *Piece) ToProto() *pb.Piece {
	pbPiece := &pb.Piece{
		Data:     p.Data,
		BucketId: p.BucketId,
		Tags:     make([]*pb.Tag, len(p.Tags)),
		Links:    make([]*pb.Link, len(p.Links)),
	}

	for i, tag := range p.Tags {
		pbPiece.Tags[i] = tag.ToProto()
	}

	for i, link := range p.Links {
		pbPiece.Links[i] = link.ToProto()
	}

	return pbPiece
}

func (p *Piece) ToDomain(pbPiece *pb.Piece) {
	p.Data = pbPiece.Data
	p.BucketId = pbPiece.BucketId
	p.Tags = make([]*Tag, len(pbPiece.Tags))
	p.Links = make([]*Link, len(pbPiece.Links))

	for i, pbTag := range pbPiece.Tags {
		tag := &Tag{}
		tag.ToDomain(pbTag)
		p.Tags[i] = tag
	}

	for i, pbLink := range pbPiece.Links {
		link := &Link{}
		link.ToDomain(pbLink)
		p.Links[i] = link
	}
}

func (p *Piece) MarshalProto() ([]byte, error) {
	return proto.Marshal(p.ToProto())
}

func (p *Piece) UnmarshalProto(pieceAsBytes []byte) error {
	pbPiece := &pb.Piece{}
	if err := proto.Unmarshal(pieceAsBytes, pbPiece); err != nil {
		return err
	}

	p.ToDomain(pbPiece)
	return nil
}
