package domain

type Protobufable interface {
	MarshalProto() ([]byte, error)
	UnmarshalProto(bytes []byte) error
}
