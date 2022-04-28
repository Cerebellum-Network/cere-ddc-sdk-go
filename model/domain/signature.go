package domain

import (
	domain "github.com/cerebellum-network/cere-ddc-sdk-go/model"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Signature struct {
	Value  string
	Signer string
	Scheme string
}

var _ domain.Protobufable = (*Signature)(nil)

func (s *Signature) ToProto() *pb.Signature {
	return &pb.Signature{
		Value:  s.Value,
		Signer: s.Signer,
		Scheme: s.Scheme,
	}
}

func (s *Signature) ToDomain(pbSignature *pb.Signature) {
	s.Value = pbSignature.Value
	s.Signer = pbSignature.Signer
	s.Scheme = pbSignature.Scheme
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
