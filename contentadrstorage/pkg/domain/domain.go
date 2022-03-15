package domain

import (
	"google.golang.org/protobuf/proto"
)

type PieceUri struct {
	BucketId uint64
	Cid      string
}

var protoMarshalOption = proto.MarshalOptions{Deterministic: true}
