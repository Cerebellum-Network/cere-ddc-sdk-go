package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

const (
	Code_SUCCESS                    Code = 0
	Code_CREATED                    Code = 1
	Code_NOT_FOUND                  Code = 2
	Code_FAILED_READ_BODY           Code = 3
	Code_FAILED_UNMARSHAL_BODY      Code = 4
	Code_FAILED_MARSHAL_BODY        Code = 5
	Code_FAILED_GET_BLOCKCHAIN_DATA Code = 6
	Code_BUCKET_RENT_EXPIRED        Code = 7
	Code_INVALID_PUBLIC_KEY         Code = 8
	Code_INVALID_SIGNATURE          Code = 9
	Code_INVALID_PARAMETER          Code = 10
	Code_BUCKET_NO_ACCESS           Code = 11
	Code_INTERNAL_ERROR             Code = 12
	Code_BAD_GATEWAY                Code = 13
	Code_INVALID_SESSION_ID         Code = 14
	Code_ACCOUNT_DEPOSIT_REQUIRED   Code = 15
	Code_GAS_EXPIRED                Code = 16
)

type (
	Code = pb.Code

	Response struct {
		Body          []byte
		PublicKey     []byte
		Signature     []byte
		Scheme        string
		Gas           uint32
		ResponseCode  Code
		MultiHashType uint64
	}
)

var _ Protobufable = (*Response)(nil)

func (r *Response) MarshalProto() ([]byte, error) {
	return proto.Marshal(r.ToProto())
}

func (r *Response) UnmarshalProto(bytes []byte) error {
	pbRequest := &pb.Response{}
	if err := proto.Unmarshal(bytes, pbRequest); err != nil {
		return err
	}

	r.ToDomain(pbRequest)
	return nil
}

func (r *Response) ToProto() *pb.Response {
	return &pb.Response{
		Body:          r.Body,
		PublicKey:     r.PublicKey,
		Signature:     r.Signature,
		Scheme:        r.Scheme,
		MultiHashType: r.MultiHashType,
		ResponseCode:  r.ResponseCode,
		Gas:           r.Gas,
	}
}

func (r *Response) ToDomain(pbRequest *pb.Response) {
	r.Body = pbRequest.Body
	r.PublicKey = pbRequest.PublicKey
	r.Signature = pbRequest.Signature
	r.Scheme = pbRequest.Scheme
	r.MultiHashType = pbRequest.MultiHashType
	r.ResponseCode = pbRequest.ResponseCode
	r.Gas = pbRequest.Gas
}
