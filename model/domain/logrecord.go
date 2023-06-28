package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

type (
	LogRecordList []*LogRecord

	LogRecord struct {
		Request        IsRequest
		Timestamp      *time.Time
		Address        string
		Gas            uint32
		PublicKey      []byte
		SessionId      []byte
		RequestId      string
		Signature      *Signature
		ResponsePieces []*ResponsePiece
	}

	WriteRequest struct {
		Cid      string
		BucketId uint32
		Size     uint32
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

	ResponsePiece struct {
		Cid  string
		Size uint32
	}
)

var _ Protobufable = (*LogRecord)(nil)
var _ Protobufable = (*LogRecordList)(nil)

func (l *LogRecord) ToProto() *pb.LogRecord {
	responsePieces := make([]*pb.ResponsePiece, 0, len(l.ResponsePieces))
	for _, piece := range l.ResponsePieces {
		responsePieces = append(responsePieces, responsePieceToProto(piece))
	}

	result := &pb.LogRecord{
		Timestamp:      uint64(l.Timestamp.UnixNano()),
		Address:        l.Address,
		Gas:            l.Gas,
		SessionId:      l.SessionId,
		PublicKey:      l.PublicKey,
		RequestId:      l.RequestId,
		ResponsePieces: responsePieces,
	}

	l.requestToProto(result)

	return result
}

func (l *LogRecord) ToDomain(pbLogRecord *pb.LogRecord) {
	timestamp := time.Unix(0, int64(pbLogRecord.Timestamp))
	l.Timestamp = &timestamp

	var signature *Signature

	l.Address = pbLogRecord.Address
	l.Gas = pbLogRecord.Gas
	l.SessionId = pbLogRecord.SessionId
	l.PublicKey = pbLogRecord.PublicKey
	l.RequestId = pbLogRecord.RequestId
	l.Request = requestToDomain(pbLogRecord)
	l.Signature = signature
	l.ResponsePieces = make([]*ResponsePiece, 0, len(pbLogRecord.ResponsePieces))
	for _, item := range pbLogRecord.ResponsePieces {
		l.ResponsePieces = append(l.ResponsePieces, responsePieceToDomain(item))
	}
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
		pbLogRecord.Request = &pb.LogRecord_WriteRequest{WriteRequest: &pb.WriteRequest{
			BucketId: record.BucketId,
			Size:     record.Size,
			Cid:      record.Cid,
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

		return &WriteRequest{
			BucketId: writeRecord.BucketId,
			Size:     writeRecord.Size,
			Cid:      writeRecord.Cid,
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

func responsePieceToDomain(pbResponsePiece *pb.ResponsePiece) *ResponsePiece {
	return &ResponsePiece{
		Cid:  pbResponsePiece.Cid,
		Size: pbResponsePiece.Size,
	}
}

func responsePieceToProto(responsePiece *ResponsePiece) *pb.ResponsePiece {
	return &pb.ResponsePiece{
		Cid:  responsePiece.Cid,
		Size: responsePiece.Size,
	}
}
