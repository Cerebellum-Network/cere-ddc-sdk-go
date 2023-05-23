// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.21.12
// source: ack.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

//
// Acknowledgment data
type Ack struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// user timestamp of request in UNIX milliseconds.
	Timestamp uint64 `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// node public key where gas was used
	PublicKey []byte `protobuf:"bytes,2,opt,name=publicKey,proto3" json:"publicKey,omitempty"`
	// confirmed used gas
	Gas uint64 `protobuf:"varint,3,opt,name=gas,proto3" json:"gas,omitempty"`
	// nonce for avoiding ack duplicates
	Nonce []byte `protobuf:"bytes,4,opt,name=nonce,proto3" json:"nonce,omitempty"`
	// request id for log record
	RequestId string `protobuf:"bytes,5,opt,name=requestId,proto3" json:"requestId,omitempty"`
	// session id
	SessionId []byte `protobuf:"bytes,7,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	// signature of acknowledgment data
	// sign(CID(requestId) + timestamp + gas + nonce + sessionID)
	Signature []byte `protobuf:"bytes,8,opt,name=signature,proto3" json:"signature,omitempty"`
	// The name of the signature scheme (sr25519, ed25519).
	// Default and recommended value: "" or "sr25519".
	Scheme string `protobuf:"bytes,9,opt,name=scheme,proto3" json:"scheme,omitempty"`
	// The ID of the hashing algorithm as per multiformats/multihash.
	// Default and recommended value: 0 or 0xb220, meaning blake2b-256.
	MultiHashType uint64 `protobuf:"varint,10,opt,name=multiHashType,proto3" json:"multiHashType,omitempty"`
	// list of chunk cIds, requested by client
	Chunks []string `protobuf:"bytes,11,rep,name=chunks,proto3" json:"chunks,omitempty"`
}

func (x *Ack) Reset() {
	*x = Ack{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ack_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Ack) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ack) ProtoMessage() {}

func (x *Ack) ProtoReflect() protoreflect.Message {
	mi := &file_ack_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ack.ProtoReflect.Descriptor instead.
func (*Ack) Descriptor() ([]byte, []int) {
	return file_ack_proto_rawDescGZIP(), []int{0}
}

func (x *Ack) GetTimestamp() uint64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *Ack) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *Ack) GetGas() uint64 {
	if x != nil {
		return x.Gas
	}
	return 0
}

func (x *Ack) GetNonce() []byte {
	if x != nil {
		return x.Nonce
	}
	return nil
}

func (x *Ack) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *Ack) GetSessionId() []byte {
	if x != nil {
		return x.SessionId
	}
	return nil
}

func (x *Ack) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *Ack) GetScheme() string {
	if x != nil {
		return x.Scheme
	}
	return ""
}

func (x *Ack) GetMultiHashType() uint64 {
	if x != nil {
		return x.MultiHashType
	}
	return 0
}

func (x *Ack) GetChunks() []string {
	if x != nil {
		return x.Chunks
	}
	return nil
}

type AckRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// acknowledgment data
	Ack *Ack `protobuf:"bytes,1,opt,name=ack,proto3" json:"ack,omitempty"`
	// user public key
	PublicKey []byte `protobuf:"bytes,2,opt,name=publicKey,proto3" json:"publicKey,omitempty"`
	// timestamp when record was saved in UNIX nanoseconds
	Timestamp uint64 `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *AckRecord) Reset() {
	*x = AckRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ack_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AckRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AckRecord) ProtoMessage() {}

func (x *AckRecord) ProtoReflect() protoreflect.Message {
	mi := &file_ack_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AckRecord.ProtoReflect.Descriptor instead.
func (*AckRecord) Descriptor() ([]byte, []int) {
	return file_ack_proto_rawDescGZIP(), []int{1}
}

func (x *AckRecord) GetAck() *Ack {
	if x != nil {
		return x.Ack
	}
	return nil
}

func (x *AckRecord) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *AckRecord) GetTimestamp() uint64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

//
// A ack records list
type AckRecordList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AckRecords []*AckRecord `protobuf:"bytes,1,rep,name=ackRecords,proto3" json:"ackRecords,omitempty"`
}

func (x *AckRecordList) Reset() {
	*x = AckRecordList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ack_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AckRecordList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AckRecordList) ProtoMessage() {}

func (x *AckRecordList) ProtoReflect() protoreflect.Message {
	mi := &file_ack_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AckRecordList.ProtoReflect.Descriptor instead.
func (*AckRecordList) Descriptor() ([]byte, []int) {
	return file_ack_proto_rawDescGZIP(), []int{2}
}

func (x *AckRecordList) GetAckRecords() []*AckRecord {
	if x != nil {
		return x.AckRecords
	}
	return nil
}

var File_ack_proto protoreflect.FileDescriptor

var file_ack_proto_rawDesc = []byte{
	0x0a, 0x09, 0x61, 0x63, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62, 0x22,
	0x99, 0x02, 0x0a, 0x03, 0x41, 0x63, 0x6b, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b,
	0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x4b, 0x65, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x67, 0x61, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x03, 0x67, 0x61, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x72,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x65, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x65,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e,
	0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x18,
	0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x12, 0x24, 0x0a,
	0x0d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x18, 0x0a,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x0d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x18, 0x0b, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x06, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x22, 0x66, 0x0a, 0x09, 0x41,
	0x63, 0x6b, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x19, 0x0a, 0x03, 0x61, 0x63, 0x6b, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x07, 0x2e, 0x70, 0x62, 0x2e, 0x41, 0x63, 0x6b, 0x52, 0x03,
	0x61, 0x63, 0x6b, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65,
	0x79, 0x12, 0x20, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x04, 0x42, 0x02, 0x30, 0x01, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x22, 0x3e, 0x0a, 0x0d, 0x41, 0x63, 0x6b, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64,
	0x4c, 0x69, 0x73, 0x74, 0x12, 0x2d, 0x0a, 0x0a, 0x61, 0x63, 0x6b, 0x52, 0x65, 0x63, 0x6f, 0x72,
	0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x41, 0x63,
	0x6b, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x52, 0x0a, 0x61, 0x63, 0x6b, 0x52, 0x65, 0x63, 0x6f,
	0x72, 0x64, 0x73, 0x42, 0x05, 0x5a, 0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_ack_proto_rawDescOnce sync.Once
	file_ack_proto_rawDescData = file_ack_proto_rawDesc
)

func file_ack_proto_rawDescGZIP() []byte {
	file_ack_proto_rawDescOnce.Do(func() {
		file_ack_proto_rawDescData = protoimpl.X.CompressGZIP(file_ack_proto_rawDescData)
	})
	return file_ack_proto_rawDescData
}

var file_ack_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_ack_proto_goTypes = []interface{}{
	(*Ack)(nil),           // 0: pb.Ack
	(*AckRecord)(nil),     // 1: pb.AckRecord
	(*AckRecordList)(nil), // 2: pb.AckRecordList
}
var file_ack_proto_depIdxs = []int32{
	0, // 0: pb.AckRecord.ack:type_name -> pb.Ack
	1, // 1: pb.AckRecordList.ackRecords:type_name -> pb.AckRecord
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_ack_proto_init() }
func file_ack_proto_init() {
	if File_ack_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ack_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Ack); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_ack_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AckRecord); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_ack_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AckRecordList); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_ack_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_ack_proto_goTypes,
		DependencyIndexes: file_ack_proto_depIdxs,
		MessageInfos:      file_ack_proto_msgTypes,
	}.Build()
	File_ack_proto = out.File
	file_ack_proto_rawDesc = nil
	file_ack_proto_goTypes = nil
	file_ack_proto_depIdxs = nil
}
