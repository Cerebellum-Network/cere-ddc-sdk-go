// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.18.1
// source: response.proto

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
//Response codes
type Code int32

const (
	Code_SUCCESS               Code = 0
	Code_CREATED               Code = 1
	Code_NOT_FOUND             Code = 2
	Code_FAILED_READ_BODY      Code = 3
	Code_FAILED_UNMARSHAL_BODY Code = 4
	Code_FAILED_MARSHAL_BODY   Code = 5
	Code_FAILED_GET_BUCKET     Code = 6
	Code_BUCKET_RENT_EXPIRED   Code = 7
	Code_INVALID_PUBLIC_KEY    Code = 8
	Code_INVALID_SIGNATURE     Code = 9
	Code_INVALID_PARAMETER     Code = 10
	Code_BUCKET_NO_ACCESS      Code = 11
	Code_INTERNAL_ERROR        Code = 12
	Code_BAD_GATEWAY           Code = 13
	Code_INVALID_SESSION_ID    Code = 14
)

// Enum value maps for Code.
var (
	Code_name = map[int32]string{
		0:  "SUCCESS",
		1:  "CREATED",
		2:  "NOT_FOUND",
		3:  "FAILED_READ_BODY",
		4:  "FAILED_UNMARSHAL_BODY",
		5:  "FAILED_MARSHAL_BODY",
		6:  "FAILED_GET_BUCKET",
		7:  "BUCKET_RENT_EXPIRED",
		8:  "INVALID_PUBLIC_KEY",
		9:  "INVALID_SIGNATURE",
		10: "INVALID_PARAMETER",
		11: "BUCKET_NO_ACCESS",
		12: "INTERNAL_ERROR",
		13: "BAD_GATEWAY",
		14: "INVALID_SESSION_ID",
	}
	Code_value = map[string]int32{
		"SUCCESS":               0,
		"CREATED":               1,
		"NOT_FOUND":             2,
		"FAILED_READ_BODY":      3,
		"FAILED_UNMARSHAL_BODY": 4,
		"FAILED_MARSHAL_BODY":   5,
		"FAILED_GET_BUCKET":     6,
		"BUCKET_RENT_EXPIRED":   7,
		"INVALID_PUBLIC_KEY":    8,
		"INVALID_SIGNATURE":     9,
		"INVALID_PARAMETER":     10,
		"BUCKET_NO_ACCESS":      11,
		"INTERNAL_ERROR":        12,
		"BAD_GATEWAY":           13,
		"INVALID_SESSION_ID":    14,
	}
)

func (x Code) Enum() *Code {
	p := new(Code)
	*p = x
	return p
}

func (x Code) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Code) Descriptor() protoreflect.EnumDescriptor {
	return file_response_proto_enumTypes[0].Descriptor()
}

func (Code) Type() protoreflect.EnumType {
	return &file_response_proto_enumTypes[0]
}

func (x Code) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Code.Descriptor instead.
func (Code) EnumDescriptor() ([]byte, []int) {
	return file_response_proto_rawDescGZIP(), []int{0}
}

//
//Response structure for API v1
type Response struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Response body
	Body []byte `protobuf:"bytes,1,opt,name=body,proto3" json:"body,omitempty"`
	// Response code
	ResponseCode Code `protobuf:"varint,2,opt,name=responseCode,proto3,enum=pb.Code" json:"responseCode,omitempty"`
	// Used gas for executing request
	Gas uint32 `protobuf:"varint,3,opt,name=gas,proto3" json:"gas,omitempty"`
	// CDN public key
	PublicKey []byte `protobuf:"bytes,4,opt,name=publicKey,proto3" json:"publicKey,omitempty"`
	// CDN signature sign(CID(varint body + body + varint response code + varint resources))
	Signature []byte `protobuf:"bytes,5,opt,name=signature,proto3" json:"signature,omitempty"`
	// The name of the signature scheme (sr25519, ed25519).
	// Default and recommended value: "" or "sr25519".
	Scheme string `protobuf:"bytes,6,opt,name=scheme,proto3" json:"scheme,omitempty"`
	// The ID of the hashing algorithm as per multiformats/multihash.
	// Default and recommended value: 0 or 0xb220, meaning blake2b-256.
	MultiHashType uint64 `protobuf:"varint,7,opt,name=multiHashType,proto3" json:"multiHashType,omitempty"`
}

func (x *Response) Reset() {
	*x = Response{}
	if protoimpl.UnsafeEnabled {
		mi := &file_response_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_response_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_response_proto_rawDescGZIP(), []int{0}
}

func (x *Response) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

func (x *Response) GetResponseCode() Code {
	if x != nil {
		return x.ResponseCode
	}
	return Code_SUCCESS
}

func (x *Response) GetGas() uint32 {
	if x != nil {
		return x.Gas
	}
	return 0
}

func (x *Response) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *Response) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *Response) GetScheme() string {
	if x != nil {
		return x.Scheme
	}
	return ""
}

func (x *Response) GetMultiHashType() uint64 {
	if x != nil {
		return x.MultiHashType
	}
	return 0
}

var File_response_proto protoreflect.FileDescriptor

var file_response_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x02, 0x70, 0x62, 0x22, 0xd8, 0x01, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x04, 0x62, 0x6f, 0x64, 0x79, 0x12, 0x2c, 0x0a, 0x0c, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x43, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x08, 0x2e, 0x70, 0x62,
	0x2e, 0x43, 0x6f, 0x64, 0x65, 0x52, 0x0c, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x43,
	0x6f, 0x64, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x67, 0x61, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x03, 0x67, 0x61, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b,
	0x65, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x4b, 0x65, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x6d, 0x75, 0x6c,
	0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x0d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x2a,
	0xc2, 0x02, 0x0a, 0x04, 0x43, 0x6f, 0x64, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43,
	0x45, 0x53, 0x53, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x44,
	0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x4e, 0x4f, 0x54, 0x5f, 0x46, 0x4f, 0x55, 0x4e, 0x44, 0x10,
	0x02, 0x12, 0x14, 0x0a, 0x10, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44, 0x5f, 0x52, 0x45, 0x41, 0x44,
	0x5f, 0x42, 0x4f, 0x44, 0x59, 0x10, 0x03, 0x12, 0x19, 0x0a, 0x15, 0x46, 0x41, 0x49, 0x4c, 0x45,
	0x44, 0x5f, 0x55, 0x4e, 0x4d, 0x41, 0x52, 0x53, 0x48, 0x41, 0x4c, 0x5f, 0x42, 0x4f, 0x44, 0x59,
	0x10, 0x04, 0x12, 0x17, 0x0a, 0x13, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44, 0x5f, 0x4d, 0x41, 0x52,
	0x53, 0x48, 0x41, 0x4c, 0x5f, 0x42, 0x4f, 0x44, 0x59, 0x10, 0x05, 0x12, 0x15, 0x0a, 0x11, 0x46,
	0x41, 0x49, 0x4c, 0x45, 0x44, 0x5f, 0x47, 0x45, 0x54, 0x5f, 0x42, 0x55, 0x43, 0x4b, 0x45, 0x54,
	0x10, 0x06, 0x12, 0x17, 0x0a, 0x13, 0x42, 0x55, 0x43, 0x4b, 0x45, 0x54, 0x5f, 0x52, 0x45, 0x4e,
	0x54, 0x5f, 0x45, 0x58, 0x50, 0x49, 0x52, 0x45, 0x44, 0x10, 0x07, 0x12, 0x16, 0x0a, 0x12, 0x49,
	0x4e, 0x56, 0x41, 0x4c, 0x49, 0x44, 0x5f, 0x50, 0x55, 0x42, 0x4c, 0x49, 0x43, 0x5f, 0x4b, 0x45,
	0x59, 0x10, 0x08, 0x12, 0x15, 0x0a, 0x11, 0x49, 0x4e, 0x56, 0x41, 0x4c, 0x49, 0x44, 0x5f, 0x53,
	0x49, 0x47, 0x4e, 0x41, 0x54, 0x55, 0x52, 0x45, 0x10, 0x09, 0x12, 0x15, 0x0a, 0x11, 0x49, 0x4e,
	0x56, 0x41, 0x4c, 0x49, 0x44, 0x5f, 0x50, 0x41, 0x52, 0x41, 0x4d, 0x45, 0x54, 0x45, 0x52, 0x10,
	0x0a, 0x12, 0x14, 0x0a, 0x10, 0x42, 0x55, 0x43, 0x4b, 0x45, 0x54, 0x5f, 0x4e, 0x4f, 0x5f, 0x41,
	0x43, 0x43, 0x45, 0x53, 0x53, 0x10, 0x0b, 0x12, 0x12, 0x0a, 0x0e, 0x49, 0x4e, 0x54, 0x45, 0x52,
	0x4e, 0x41, 0x4c, 0x5f, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x0c, 0x12, 0x0f, 0x0a, 0x0b, 0x42,
	0x41, 0x44, 0x5f, 0x47, 0x41, 0x54, 0x45, 0x57, 0x41, 0x59, 0x10, 0x0d, 0x12, 0x16, 0x0a, 0x12,
	0x49, 0x4e, 0x56, 0x41, 0x4c, 0x49, 0x44, 0x5f, 0x53, 0x45, 0x53, 0x53, 0x49, 0x4f, 0x4e, 0x5f,
	0x49, 0x44, 0x10, 0x0e, 0x42, 0x05, 0x5a, 0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_response_proto_rawDescOnce sync.Once
	file_response_proto_rawDescData = file_response_proto_rawDesc
)

func file_response_proto_rawDescGZIP() []byte {
	file_response_proto_rawDescOnce.Do(func() {
		file_response_proto_rawDescData = protoimpl.X.CompressGZIP(file_response_proto_rawDescData)
	})
	return file_response_proto_rawDescData
}

var file_response_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_response_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_response_proto_goTypes = []interface{}{
	(Code)(0),        // 0: pb.Code
	(*Response)(nil), // 1: pb.Response
}
var file_response_proto_depIdxs = []int32{
	0, // 0: pb.Response.responseCode:type_name -> pb.Code
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_response_proto_init() }
func file_response_proto_init() {
	if File_response_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_response_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response); i {
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
			RawDescriptor: file_response_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_response_proto_goTypes,
		DependencyIndexes: file_response_proto_depIdxs,
		EnumInfos:         file_response_proto_enumTypes,
		MessageInfos:      file_response_proto_msgTypes,
	}.Build()
	File_response_proto = out.File
	file_response_proto_rawDesc = nil
	file_response_proto_goTypes = nil
	file_response_proto_depIdxs = nil
}
