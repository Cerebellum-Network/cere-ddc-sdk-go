package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

const (
	Code_SUCCESS               Code = 0
	Code_INTERNAL_ERROR        Code = 1
	Code_BAD_REQUEST_BODY      Code = 2
	Code_DESERIALIZE_FAIL      Code = 3
	Code_BUCKET_READ_FAIL      Code = 4
	Code_BUCKET_RENT_EXPIRED   Code = 5
	Code_INVALID_PUBLIC_KEY    Code = 6
	Code_INVALID_SIGNATURE     Code = 7
	Code_NO_ACCESS             Code = 8
	Code_SERIALIZE_FAIL        Code = 9
	Code_NO_REQUIRED_PARAMETER Code = 10
)

type (
	Code int32

	Ack struct {
		Request      isRequest
		Nonce        []byte
		NodeId       uint32
		ResponseCode Code
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

	isRequest interface {
		isRequest()
	}
)

var _ Protobufable = (*Ack)(nil)

func (a *Ack) ToProto() *pb.Ack {
	result := &pb.Ack{
		Nonce:        a.Nonce,
		NodeId:       a.NodeId,
		ResponseCode: pb.Code(a.ResponseCode),
	}

	a.recordToProto(result)

	return result
}

func (a *Ack) ToDomain(pbAck *pb.Ack) {
	a.Nonce = pbAck.Nonce
	a.NodeId = pbAck.NodeId
	a.ResponseCode = Code(pbAck.ResponseCode)
	a.Request = requestToDomain(pbAck)
}

func (a *Ack) MarshalProto() ([]byte, error) {
	return proto.Marshal(a.ToProto())
}

func (a *Ack) UnmarshalProto(ackAsBytes []byte) error {
	pbAck := &pb.Ack{}
	if err := proto.Unmarshal(ackAsBytes, pbAck); err != nil {
		return err
	}

	a.ToDomain(pbAck)
	return nil
}

func (a *Ack) recordToProto(ack *pb.Ack) {
	switch record := a.Request.(type) {
	case *WriteRequest:
		signature := record.Signature.ToProto()
		ack.Request = &pb.Ack_WriteRequest{WriteRequest: &pb.WriteRequest{
			BucketId:  record.BucketId,
			Size:      record.Size,
			Cid:       record.Cid,
			Signature: signature,
		}}
	case *ReadRequest:
		ack.Request = &pb.Ack_ReadRequest{ReadRequest: &pb.ReadRequest{
			Cid:      record.Cid,
			BucketId: record.BucketId,
		}}
	case *QueryRequest:
		ack.Request = &pb.Ack_QueryRequest{QueryRequest: &pb.QueryRequest{
			Query: record.Query.ToProto(),
		}}
	}
}

func requestToDomain(pbAck *pb.Ack) isRequest {
	switch record := pbAck.Request.(type) {
	case *pb.Ack_WriteRequest:
		writeRecord := record.WriteRequest

		signature := &Signature{}
		signature.ToDomain(writeRecord.Signature)

		return &WriteRequest{
			BucketId:  writeRecord.BucketId,
			Size:      writeRecord.Size,
			Cid:       writeRecord.Cid,
			Signature: signature,
		}
	case *pb.Ack_ReadRequest:
		readRecord := record.ReadRequest
		return &ReadRequest{Cid: readRecord.Cid, BucketId: readRecord.BucketId}
	case *pb.Ack_QueryRequest:
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
