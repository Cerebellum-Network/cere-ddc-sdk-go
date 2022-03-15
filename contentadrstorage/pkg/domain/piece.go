package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/pb"
)

type Piece struct {
	Data     []byte
	BucketId uint64
	Tags     []*Tag
}

func (p *Piece) ToProto() *pb.Piece {
	tags := make([]*pb.Tag, len(p.Tags))

	for i, t := range p.Tags {
		tags[i] = t.toProto()
	}

	return &pb.Piece{
		Data:     p.Data,
		BucketId: p.BucketId,
		Tags:     tags,
	}
}

func (p *Piece) FromProto(pbPiece *pb.Piece) {
	p.Data = pbPiece.Data
	p.BucketId = pbPiece.BucketId
	p.Tags = make([]*Tag, len(pbPiece.Tags))

	for i, t := range pbPiece.Tags {
		tag := &Tag{}
		tag.fromProto(t)

		p.Tags[i] = tag
	}
}
