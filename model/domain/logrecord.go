package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

type (
	LogRecordList []*LogRecord

	LogRecord struct {
		Request   IsRequest
		Timestamp *time.Time
		Address   string
		Gas       uint32
		PublicKey []byte
		SessionId []byte
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
var _ Protobufable = (*LogRecordList)(nil)

func (l *LogRecord) ToProto() *pb.LogRecord {
	result := &pb.LogRecord{
		Timestamp: uint64(l.Timestamp.UnixMilli()),
		Address:   l.Address,
		Gas:       l.Gas,
		SessionId: l.SessionId,
		PublicKey: l.PublicKey,
	}

	l.requestToProto(result)

	return result
}

func (l *LogRecord) ToDomain(pbLogRecord *pb.LogRecord) {
	timestamp := time.UnixMilli(int64(pbLogRecord.Timestamp))
	l.Timestamp = &timestamp

	l.Address = pbLogRecord.Address
	l.Gas = pbLogRecord.Gas
	l.SessionId = pbLogRecord.SessionId
	l.PublicKey = pbLogRecord.PublicKey
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

func (l *LogRecordList) MarshalProto() ([]byte, error) {
	return proto.Marshal(l.ToProto())
}

func (l *LogRecordList) UnmarshalProto(logRecordAsBytes []byte) error {
	pbLogRecords := &pb.LogRecordList{}
	if err := proto.Unmarshal(logRecordAsBytes, pbLogRecords); err != nil {
		return err
	}

	l.ToDomain(pbLogRecords)
	return nil
}

func (l *LogRecordList) ToProto() *pb.LogRecordList {
	list := *l
	result := make([]*pb.LogRecord, 0, len(list))
	for _, v := range list {
		result = append(result, v.ToProto())
	}

	return &pb.LogRecordList{LogRecords: result}
}

func (l *LogRecordList) ToDomain(pbLogRecordList *pb.LogRecordList) {
	result := LogRecordList(make([]*LogRecord, 0, len(pbLogRecordList.LogRecords)))
	for _, v := range pbLogRecordList.LogRecords {
		record := &LogRecord{}
		record.ToDomain(v)
		result = append(result, record)
	}

	*l = result
}

func (l *LogRecord) requestToProto(pbLogRecord *pb.LogRecord) {
	switch record := l.Request.(type) {
	case *WriteRequest:
		var signature *pb.Signature
		if record.Signature != nil {
			signature = record.Signature.ToProto()
		}

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
		var query *pb.Query
		if record.Query != nil {
			query = record.Query.ToProto()
		}

		pbLogRecord.Request = &pb.LogRecord_QueryRequest{QueryRequest: &pb.QueryRequest{
			Query: query,
		}}
	}
}

func requestToDomain(pbLogRecord *pb.LogRecord) IsRequest {
	switch record := pbLogRecord.Request.(type) {
	case *pb.LogRecord_WriteRequest:
		writeRecord := record.WriteRequest

		var signature *Signature
		if writeRecord.Signature != nil {
			signature = &Signature{}
			signature.ToDomain(writeRecord.Signature)
		}

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

		var query *Query
		if queryRecord.Query != nil {
			query = &Query{}
			query.ToDomain(queryRecord.Query)
		}

		return &QueryRequest{Query: query}
	}

	return nil
}

func (w *WriteRequest) isRequest() {}
func (q *QueryRequest) isRequest() {}
func (r *ReadRequest) isRequest()  {}
