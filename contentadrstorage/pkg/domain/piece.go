package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/pb"
)

type Piece struct {
	Data     []byte
	BucketId uint32
	Tags     []*Tag
	Links    []*Link
}

func (p *Piece) ToProto() *pb.Piece {
	tags := make([]*pb.Tag, len(p.Tags))

	for i, t := range p.Tags {
		tags[i] = t.ToProto()
	}

	links := make([]*pb.Link, len(p.Links))
	for i, l := range p.Links {
		links[i] = l.ToProto()
	}

	return &pb.Piece{
		Data:     p.Data,
		BucketId: p.BucketId,
		Tags:     tags,
		Links:    links,
	}
}

func (p *Piece) FromProto(pbPiece *pb.Piece) {
	p.Data = pbPiece.Data
	p.BucketId = pbPiece.BucketId

	p.Tags = make([]*Tag, len(pbPiece.Tags))
	for i, t := range pbPiece.Tags {
		tag := &Tag{}
		tag.FromProto(t)

		p.Tags[i] = tag
	}

	p.Links = make([]*Link, len(pbPiece.Links))
	for i, l := range pbPiece.Links {
		link := &Link{}
		link.FromProto(l)

		p.Links[i] = link
	}
}
