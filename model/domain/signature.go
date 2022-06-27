package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Signature struct {
	Value         []byte
	Signer        []byte
	Scheme        string
	MultiHashType uint64
}

var _ Protobufable = (*Signature)(nil)

func (s *Signature) ToProto() *pb.Signature {
	return &pb.Signature{
		Value:         s.Value,
		Signer:        s.Signer,
		Scheme:        s.Scheme,
		MultiHashType: s.MultiHashType,
	}
}

func (s *Signature) ToDomain(pbSignature *pb.Signature) {
	s.Value = pbSignature.Value
	s.Signer = pbSignature.Signer
	s.Scheme = pbSignature.Scheme
	s.MultiHashType = pbSignature.MultiHashType
}

func (s *Signature) MarshalProto() ([]byte, error) {
	return proto.Marshal(s.ToProto())
}

func (s *Signature) UnmarshalProto(signatureAsBytes []byte) error {
	signature := &pb.Signature{}
	if err := proto.Unmarshal(signatureAsBytes, signature); err != nil {
		return err
	}

	s.ToDomain(signature)
	return nil
}
