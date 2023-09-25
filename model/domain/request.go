package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Request struct {
	Body          []byte
	PublicKey     []byte
	Signature     []byte
	Scheme        string
	MultiHashType uint64
	SessionId     []byte
	RequestId     string
}

var _ Protobufable = (*Request)(nil)

func (r *Request) MarshalProto() ([]byte, error) {
	return proto.Marshal(r.ToProto())
}

func (r *Request) UnmarshalProto(bytes []byte) error {
	pbRequest := &pb.Request{}
	if err := proto.Unmarshal(bytes, pbRequest); err != nil {
		return err
	}

	r.ToDomain(pbRequest)
	return nil
}

func (r *Request) ToProto() *pb.Request {
	return &pb.Request{
		Body:          r.Body,
		PublicKey:     r.PublicKey,
		Signature:     r.Signature,
		Scheme:        r.Scheme,
		MultiHashType: r.MultiHashType,
		SessionId:     r.SessionId,
		RequestId:     r.RequestId,
	}
}

func (r *Request) ToDomain(pbRequest *pb.Request) {
	r.Body = pbRequest.Body
	r.PublicKey = pbRequest.PublicKey
	r.Signature = pbRequest.Signature
	r.Scheme = pbRequest.Scheme
	r.MultiHashType = pbRequest.MultiHashType
	r.SessionId = pbRequest.SessionId
	r.RequestId = pbRequest.RequestId
}
