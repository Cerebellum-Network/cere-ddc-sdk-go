package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Tag struct {
	Key        []byte
	Value      []byte
	Searchable pb.SearchType
}

var _ Protobufable = (*Tag)(nil)

func (t *Tag) ToProto() *pb.Tag {
	return &pb.Tag{
		Key:        t.Key,
		Value:      t.Value,
		Searchable: t.Searchable,
	}
}

func (t *Tag) ToDomain(pbTag *pb.Tag) {
	t.Key = pbTag.Key
	t.Value = pbTag.Value
	t.Searchable = pbTag.Searchable
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
