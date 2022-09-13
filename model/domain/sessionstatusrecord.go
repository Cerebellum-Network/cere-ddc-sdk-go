package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type SessionStatusRecord struct {
	SessionStatus *SessionStatus
	PublicKey     []byte
	Signature     []byte
	Timestamp     uint64
}

var _ Protobufable = (*SessionStatusRecord)(nil)

func (s *SessionStatusRecord) MarshalProto() ([]byte, error) {
	return proto.Marshal(s.ToProto())
}

func (s *SessionStatusRecord) UnmarshalProto(bytes []byte) error {
	pbSessionStatusRecord := &pb.SessionStatusRecord{}
	if err := proto.Unmarshal(bytes, pbSessionStatusRecord); err != nil {
		return err
	}

	s.ToDomain(pbSessionStatusRecord)
	return nil
}

func (s *SessionStatusRecord) ToProto() *pb.SessionStatusRecord {
	return &pb.SessionStatusRecord{
		SessionStatus: s.SessionStatus.ToProto(),
		PublicKey:     s.PublicKey,
		Signature:     s.Signature,
		Timestamp:     s.Timestamp,
	}
}

func (s *SessionStatusRecord) ToDomain(pbSessionStatusRecord *pb.SessionStatusRecord) {
	sessionStatus := &SessionStatus{}
	sessionStatus.ToDomain(pbSessionStatusRecord.SessionStatus)

	s.SessionStatus = sessionStatus
	s.PublicKey = pbSessionStatusRecord.PublicKey
	s.Signature = pbSessionStatusRecord.Signature
	s.Timestamp = pbSessionStatusRecord.Timestamp
}
