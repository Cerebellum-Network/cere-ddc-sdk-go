package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type SessionStatus struct {
	PublicKey  []byte
	Gas        uint32
	SessionId  []byte
	EndOfEpoch uint64
}

var _ Protobufable = (*SessionStatus)(nil)

func (s *SessionStatus) MarshalProto() ([]byte, error) {
	return proto.Marshal(s.ToProto())
}

func (s *SessionStatus) UnmarshalProto(bytes []byte) error {
	pbSessionStatus := &pb.SessionStatus{}
	if err := proto.Unmarshal(bytes, pbSessionStatus); err != nil {
		return err
	}

	s.ToDomain(pbSessionStatus)
	return nil
}

func (s *SessionStatus) ToProto() *pb.SessionStatus {
	return &pb.SessionStatus{
		PublicKey:  s.PublicKey,
		Gas:        s.Gas,
		SessionId:  s.SessionId,
		EndOfEpoch: s.EndOfEpoch,
	}
}

func (s *SessionStatus) ToDomain(pbSessionStatus *pb.SessionStatus) {
	s.PublicKey = pbSessionStatus.PublicKey
	s.Gas = pbSessionStatus.Gas
	s.SessionId = pbSessionStatus.SessionId
	s.EndOfEpoch = pbSessionStatus.EndOfEpoch
}
