package domain

import "github.com/cerebellum-network/cere-ddc-sdk-go/pb"

type Link struct {
	Cid  string
	Size uint64
	Name *string
}

func (p *Link) ToProto() *pb.Link {
	return &pb.Link{
		Cid:  p.Cid,
		Size: p.Size,
		Name: p.Name,
	}
}

func (p *Link) FromProto(pbLink *pb.Link) {
	p.Cid = pbLink.Cid
	p.Size = pbLink.Size
	p.Name = pbLink.Name
}
