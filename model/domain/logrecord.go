package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type (
	LogRecord struct {
		Request   IsRequest
		Timestamp *time.Time
		Address   string
		Resources uint32
	}

	WriteRequest struct {
		Cid       string
		BucketId  uint32
		Size      uint32
		Signature *Signature
	}

	ReadRequest struct {
		Cid      string
		BucketId uint32
	}

	QueryRequest struct {
		Query *Query
	}

	IsRequest interface {
		isRequest()
	}
)

var _ Protobufable = (*LogRecord)(nil)

func (l *LogRecord) ToProto() *pb.LogRecord {
	result := &pb.LogRecord{
		Timestamp: timestamppb.New(*l.Timestamp),
		Address:   l.Address,
		Resources: l.Resources,
	}

	l.requestToProto(result)

	return result
}

func (l *LogRecord) ToDomain(pbLogRecord *pb.LogRecord) {
	timestamp := pbLogRecord.Timestamp.AsTime()
	l.Timestamp = &timestamp

	l.Address = pbLogRecord.Address
	l.Resources = pbLogRecord.Resources
	l.Request = requestToDomain(pbLogRecord)
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

func (l *LogRecord) requestToProto(pbLogRecord *pb.LogRecord) {
	switch record := l.Request.(type) {
	case *WriteRequest:
		signature := record.Signature.ToProto()
		pbLogRecord.Request = &pb.LogRecord_WriteRequest{WriteRequest: &pb.WriteRequest{
			BucketId:  record.BucketId,
			Size:      record.Size,
			Cid:       record.Cid,
			Signature: signature,
		}}
	case *ReadRequest:
		pbLogRecord.Request = &pb.LogRecord_ReadRequest{ReadRequest: &pb.ReadRequest{
			Cid:      record.Cid,
			BucketId: record.BucketId,
		}}
	case *QueryRequest:
		pbLogRecord.Request = &pb.LogRecord_QueryRequest{QueryRequest: &pb.QueryRequest{
			Query: record.Query.ToProto(),
		}}
	}
}

func requestToDomain(pbLogRecord *pb.LogRecord) IsRequest {
	switch record := pbLogRecord.Request.(type) {
	case *pb.LogRecord_WriteRequest:
		writeRecord := record.WriteRequest

		signature := &Signature{}
		signature.ToDomain(writeRecord.Signature)

		return &WriteRequest{
			BucketId:  writeRecord.BucketId,
			Size:      writeRecord.Size,
			Cid:       writeRecord.Cid,
			Signature: signature,
		}
	case *pb.LogRecord_ReadRequest:
		readRecord := record.ReadRequest
		return &ReadRequest{Cid: readRecord.Cid, BucketId: readRecord.BucketId}
	case *pb.LogRecord_QueryRequest:
		queryRecord := record.QueryRequest
		query := &Query{}
		query.ToDomain(queryRecord.Query)
		return &QueryRequest{Query: query}
	}

	return nil
}

func (w *WriteRequest) isRequest() {}
func (q *QueryRequest) isRequest() {}
func (r *ReadRequest) isRequest()  {}
