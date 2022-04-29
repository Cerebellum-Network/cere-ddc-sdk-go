package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Query struct {
	BucketId uint32
	Tags     []*Tag
}

var _ Protobufable = (*Query)(nil)

func (q *Query) ToProto() *pb.Query {
	pbQuery := &pb.Query{
		BucketId: q.BucketId,
		Tags:     make([]*pb.Tag, len(q.Tags)),
	}

	for i, tag := range q.Tags {
		pbQuery.Tags[i] = tag.ToProto()
	}

	return pbQuery
}

func (q *Query) ToDomain(pbQuery *pb.Query) {
	q.BucketId = pbQuery.BucketId
	q.Tags = make([]*Tag, len(pbQuery.Tags))

	for i, pbTag := range pbQuery.Tags {
		tag := &Tag{}
		tag.ToDomain(pbTag)
		q.Tags[i] = tag
	}
}

func (q *Query) MarshalProto() ([]byte, error) {
	return proto.Marshal(q.ToProto())
}

func (q *Query) UnmarshalProto(queryAsBytes []byte) error {
	pbQuery := &pb.Query{}
	if err := proto.Unmarshal(queryAsBytes, pbQuery); err != nil {
		return err
	}

	q.ToDomain(pbQuery)
	return nil
}
