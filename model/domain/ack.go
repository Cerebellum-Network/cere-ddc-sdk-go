package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

type Ack struct {
	Timestamp *time.Time
	Gas       uint64
	PublicKey []byte
	Nonce     []byte
	RequestId string
}

var _ Protobufable = (*Ack)(nil)

func (a *Ack) MarshalProto() ([]byte, error) {
	return proto.Marshal(a.ToProto())
}

func (a *Ack) UnmarshalProto(bytes []byte) error {
	ack := &pb.Ack{}
	if err := proto.Unmarshal(bytes, ack); err != nil {
		return err
	}

	a.ToDomain(ack)
	return nil
}

func (a *Ack) ToDomain(ack *pb.Ack) {
	timestamp := time.UnixMilli(int64(ack.Timestamp))
	a.Timestamp = &timestamp

	a.PublicKey = ack.PublicKey
	a.Gas = ack.Gas
	a.Nonce = ack.Nonce
	a.RequestId = ack.RequestId
}

func (a *Ack) ToProto() *pb.Ack {
	return &pb.Ack{
		PublicKey: a.PublicKey,
		Gas:       a.Gas,
		Nonce:     a.Nonce,
		RequestId: a.RequestId,
		Timestamp: uint64(a.Timestamp.UnixMilli()),
	}
}
