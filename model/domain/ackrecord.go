package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type AckRecord struct {
	Ack       *Ack
	PublicKey []byte
}

var _ Protobufable = (*AckRecord)(nil)

func (a *AckRecord) MarshalProto() ([]byte, error) {
	return proto.Marshal(a.ToProto())
}

func (a *AckRecord) UnmarshalProto(bytes []byte) error {
	ackRecord := &pb.AckRecord{}
	if err := proto.Unmarshal(bytes, ackRecord); err != nil {
		return err
	}

	a.ToDomain(ackRecord)
	return nil
}

func (a *AckRecord) ToProto() *pb.AckRecord {
	return &pb.AckRecord{
		Ack:       a.Ack.ToProto(),
		PublicKey: a.PublicKey,
	}
}

func (a *AckRecord) ToDomain(record *pb.AckRecord) {
	ack := &Ack{}
	ack.ToDomain(record.Ack)

	a.Ack = ack
	a.PublicKey = record.PublicKey
}
