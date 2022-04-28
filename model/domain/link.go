package domain

import (
	domain "github.com/cerebellum-network/cere-ddc-sdk-go/model"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Link struct {
	Cid  string
	Size uint64
	Name *string
}

var _ domain.Protobufable = (*Link)(nil)

func (l *Link) ToProto() *pb.Link {
	return &pb.Link{
		Cid:  l.Cid,
		Size: l.Size,
		Name: l.Name,
	}
}

func (l *Link) ToDomain(pbLink *pb.Link) {
	l.Cid = pbLink.Cid
	l.Size = pbLink.Size
	l.Name = pbLink.Name
}

func (l *Link) MarshalProto() ([]byte, error) {
	return proto.Marshal(l.ToProto())
}

func (l *Link) UnmarshalProto(linkAsBytes []byte) error {
	link := &pb.Link{}
	if err := proto.Unmarshal(linkAsBytes, link); err != nil {
		return err
	}

	l.ToDomain(link)
	return nil
}
