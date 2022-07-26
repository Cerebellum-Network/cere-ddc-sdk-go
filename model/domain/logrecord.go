package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type (
	LogRecord struct {
		Record    isLogRecord
		Timestamp time.Time
		Id        string
	}

	WriteRecord struct {
		Cid       string
		BucketId  uint32
		Size      uint32
		Signature *Signature
	}

	ReadRecord struct {
		Cid      string
		BucketId uint32
	}

	QueryRecord struct {
		Query *Query
	}

	isLogRecord interface {
		isLogRecord()
	}
)

func (l *LogRecord) ToProto() *pb.LogRecord {
	result := &pb.LogRecord{
		Timestamp: timestamppb.New(l.Timestamp),
		Id:        l.Id,
	}
	l.recordToProto(result)

	return result
}

func (l *LogRecord) ToDomain(pbLogRecord *pb.LogRecord) {
	l.Id = pbLogRecord.Id
	l.Timestamp = pbLogRecord.Timestamp.AsTime()
	l.Record = recordToDomain(pbLogRecord)
}

func (l *LogRecord) recordToProto(logRecord *pb.LogRecord) {
	switch record := l.Record.(type) {
	case *WriteRecord:
		signature := record.Signature.ToProto()
		logRecord.Record = &pb.LogRecord_WriteRecord{WriteRecord: &pb.WriteRecord{
			BucketId:  record.BucketId,
			Size:      record.Size,
			Cid:       record.Cid,
			Signature: signature,
		}}
	case *ReadRecord:
		logRecord.Record = &pb.LogRecord_ReadRecord{ReadRecord: &pb.ReadRecord{
			Cid:      record.Cid,
			BucketId: record.BucketId,
		}}
	case *QueryRecord:
		logRecord.Record = &pb.LogRecord_QueryRecord{QueryRecord: &pb.QueryRecord{
			Query: record.Query.ToProto(),
		}}
	}
}

func recordToDomain(pbLogRecord *pb.LogRecord) isLogRecord {
	switch record := pbLogRecord.Record.(type) {
	case *pb.LogRecord_WriteRecord:
		writeRecord := record.WriteRecord

		signature := &Signature{}
		signature.ToDomain(writeRecord.Signature)

		return &WriteRecord{
			BucketId:  writeRecord.BucketId,
			Size:      writeRecord.Size,
			Cid:       writeRecord.Cid,
			Signature: signature,
		}
	case *pb.LogRecord_ReadRecord:
		readRecord := record.ReadRecord
		return &ReadRecord{Cid: readRecord.Cid, BucketId: readRecord.BucketId}
	case *pb.LogRecord_QueryRecord:
		queryRecord := record.QueryRecord
		query := &Query{}
		query.ToDomain(queryRecord.Query)
		return &QueryRecord{Query: query}
	}

	return nil
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

func (w *WriteRecord) isLogRecord() {}
func (q *QueryRecord) isLogRecord() {}
func (r *ReadRecord) isLogRecord()  {}
