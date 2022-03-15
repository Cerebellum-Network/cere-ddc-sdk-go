package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/pb"
)

type Query struct {
	BucketId uint64
	Tags     []*Tag
}

func (q *Query) ToProto() *pb.Query {
	tags := make([]*pb.Tag, len(q.Tags))

	for i, t := range q.Tags {
		tags[i] = t.ToProto()
	}

	return &pb.Query{
		BucketId: q.BucketId,
		Tags:     tags,
	}
}

func (q *Query) FromProto(pbQuery *pb.Query) {
	q.BucketId = pbQuery.BucketId
	q.Tags = make([]*Tag, len(q.Tags))

	for i, t := range pbQuery.Tags {
		tag := &Tag{}
		tag.FromProto(t)

		q.Tags[i] = tag
	}
}
