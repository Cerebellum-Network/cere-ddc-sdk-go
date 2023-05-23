package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

type Ack struct {
	Timestamp     *time.Time
	Gas           uint64
	PublicKey     []byte
	Nonce         []byte
	RequestId     string
	SessionId     []byte
	Signature     []byte
	Schema        string
	MultiHashType uint64
	Chunks        []string
}

var _ Protobufable = (*Ack)(nil)

func (a *Ack) MarshalProto() ([]byte, error) {
	return proto.Marshal(a.ToProto())
}

func (a *Ack) UnmarshalProto(bytes []byte) error {
	ack := &pb.Ack{}
	if err := proto.Unmarshal(bytes, ack); err != nil {
		return err
	}

	a.ToDomain(ack)
	return nil
}

func (a *Ack) ToDomain(ack *pb.Ack) {
	timestamp := time.UnixMilli(int64(ack.Timestamp))
	a.Timestamp = &timestamp

	a.PublicKey = ack.PublicKey
	a.Gas = ack.Gas
	a.Nonce = ack.Nonce
	a.RequestId = ack.RequestId
	a.SessionId = ack.SessionId
	a.Signature = ack.Signature
	a.Schema = ack.Scheme
	a.MultiHashType = ack.MultiHashType
	a.Chunks = make([]string, len(ack.Chunks))
	copy(a.Chunks, ack.Chunks)
}

func (a *Ack) ToProto() *pb.Ack {
	return func() *pb.Ack {
		ack := &pb.Ack{
			PublicKey:     a.PublicKey,
			Gas:           a.Gas,
			Nonce:         a.Nonce,
			RequestId:     a.RequestId,
			SessionId:     a.SessionId,
			Signature:     a.Signature,
			Scheme:        a.Schema,
			MultiHashType: a.MultiHashType,
			Timestamp:     uint64(a.Timestamp.UnixMilli()),
		}
		ack.Chunks = make([]string, len(a.Chunks))
		copy(ack.Chunks, a.Chunks)
		return ack
	}()
}
