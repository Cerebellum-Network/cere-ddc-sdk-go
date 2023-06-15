package domain

import (
	"errors"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type Signature struct {
	Value         []byte
	Signer        []byte
	Scheme        string
	MultiHashType uint64
	Timestamp     uint64
}

var _ Protobufable = (*Signature)(nil)

func (s *Signature) ToProto() *pb.Signature {
	return &pb.Signature{
		Value:         s.Value,
		Signer:        s.Signer,
		Scheme:        s.Scheme,
		MultiHashType: s.MultiHashType,
		Timestamp:     s.Timestamp,
	}
}

func (s *Signature) ToDomain(pbSignature *pb.Signature) {
	s.Value = pbSignature.Value
	s.Signer = pbSignature.Signer
	s.Scheme = pbSignature.Scheme
	s.MultiHashType = pbSignature.MultiHashType
	s.Timestamp = pbSignature.Timestamp
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

func (s *Signature) DecodedValue() ([]byte, error) {
	if len(s.Value) > 64 {
		signature, err := decodeHex(s.Value)
		if err != nil {
			return nil, errors.New("unable to decode hex signature")
		}

		return signature, nil
	}

	return s.Value, nil
}

func (s *Signature) DecodedSigner() ([]byte, error) {
	if len(s.Signer) > 32 {
		signer, err := decodeHex(s.Signer)
		if err != nil {
			return nil, errors.New("unable to decode hex signer")
		}

		return signer, nil
	}

	return s.Signer, nil
}
