package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

type (
	AckRecordList []*AckRecord

	AckRecord struct {
		Ack       *Ack
		PublicKey []byte
		Timestamp *time.Time
	}
)

var _ Protobufable = (*AckRecord)(nil)
var _ Protobufable = (*AckRecordList)(nil)

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
		Timestamp: uint64(a.Timestamp.UnixNano()),
	}
}

func (a *AckRecord) ToDomain(record *pb.AckRecord) {
	ack := &Ack{}
	ack.ToDomain(record.Ack)
	timestamp := time.Unix(0, int64(record.Timestamp))

	a.Ack = ack
	a.PublicKey = record.PublicKey
	a.Timestamp = &timestamp
}

func (a *AckRecordList) MarshalProto() ([]byte, error) {
	return proto.Marshal(a.ToProto())
}

func (a *AckRecordList) UnmarshalProto(bytes []byte) error {
	pbAckRecords := &pb.AckRecordList{}
	if err := proto.Unmarshal(bytes, pbAckRecords); err != nil {
		return err
	}

	a.ToDomain(pbAckRecords)
	return nil
}

func (a *AckRecordList) ToProto() *pb.AckRecordList {
	list := *a
	result := make([]*pb.AckRecord, 0, len(list))
	for _, v := range list {
		result = append(result, v.ToProto())
	}

	return &pb.AckRecordList{AckRecords: result}
}

func (a *AckRecordList) ToDomain(records *pb.AckRecordList) {
	result := AckRecordList(make([]*AckRecord, 0, len(records.AckRecords)))
	for _, v := range records.AckRecords {
		record := &AckRecord{}
		record.ToDomain(v)
		result = append(result, record)
	}

	*a = result
}
