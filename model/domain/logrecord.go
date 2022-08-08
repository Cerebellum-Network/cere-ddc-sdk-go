package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type LogRecord struct {
	Ack       []byte
	Timestamp *time.Time
	Address   string
}

var _ Protobufable = (*LogRecord)(nil)

func (l *LogRecord) ToProto() *pb.LogRecord {
	result := &pb.LogRecord{
		Timestamp: timestamppb.New(*l.Timestamp),
		Address:   l.Address,
		Ack:       l.Ack,
	}

	return result
}

func (l *LogRecord) ToDomain(pbLogRecord *pb.LogRecord) {
	timestamp := pbLogRecord.Timestamp.AsTime()
	l.Timestamp = &timestamp
	l.Address = pbLogRecord.Address
	l.Ack = pbLogRecord.Ack
}

func (l *LogRecord) MarshalProto() ([]byte, error) {
	return proto.Marshal(l.ToProto())
}

func (l *LogRecord) UnmarshalProto(logRecordAsBytes []byte) error {
	pbLogRecord := &pb.LogRecord{}
	if err := proto.Unmarshal(logRecordAsBytes, pbLogRecord); err != nil {
		return err
	}

	l.ToDomain(pbLogRecord)
	return nil
}
