package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/pb"
)

type Tag struct {
	Key   string
	Value string
}

func (t *Tag) ToProto() *pb.Tag {
	return &pb.Tag{
		Key:   t.Key,
		Value: t.Value,
	}
}

func (t *Tag) FromProto(pbTag *pb.Tag) {
	t.Key = pbTag.Key
	t.Value = pbTag.Value
}
