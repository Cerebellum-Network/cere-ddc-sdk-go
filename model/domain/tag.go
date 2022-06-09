package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Tag struct {
	Key   string
	Value []byte
}

var _ Protobufable = (*Tag)(nil)

func (t *Tag) ToProto() *pb.Tag {
	return &pb.Tag{
		Key:   t.Key,
		Value: t.Value,
	}
}

func (t *Tag) ToDomain(pbTag *pb.Tag) {
	t.Key = pbTag.Key
	t.Value = pbTag.Value
}

func (t *Tag) MarshalProto() ([]byte, error) {
	return proto.Marshal(t.ToProto())
}

func (t *Tag) UnmarshalProto(tagAsBytes []byte) error {
	tag := &pb.Tag{}
	if err := proto.Unmarshal(tagAsBytes, tag); err != nil {
		return err
	}

	t.ToDomain(tag)
	return nil
}
