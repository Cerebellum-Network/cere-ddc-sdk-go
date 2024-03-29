// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.23.3
// source: signature.proto

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

// A signature and details to help verify it.
//
// #### Generation
//
// - Compute a CID from the `piece` bytes using details from `signature`:
//   - [CIDv1](https://github.com/multiformats/cid).
//   - The hash function should be blake2b-256 and `multiHashType` should be empty.
//   - Content type codec `0x55`
//   - Base encoded in Base32 with the prefix `b`
//   - Example: bafk2bzacea73ycjnxe2qov7cvnhx52lzfp6nf5jcblnfus6gqreh6ygganbws
//
// - Store the public key of the signer in `publicKey` in binary encoding.
//
// - Store the current time in `timestamp`
//   - In JavaScript: `timestamp = +new Date()`
//
// - Format the current time in ISO 8601 `YYYY-MM-DDTHH:mm:ss.sssZ`
// - In JavaScript: `timeText = new Date(timestamp).toISOString()`
// - In Go format: `2006-01-02T15:04:05.000Z`
//
// - The signed message to store a piece is:
//   - `<Bytes>DDC store ${CID} at ${timeText}</Bytes>`
//   - Note: the `<Bytes>` part is enforced by the Polkadot.js browser extension.
//   - Example: `<Bytes>DDC store bafk2bzacea73ycjnxe2qov7cvnhx52lzfp6nf5jcblnfus6gqreh6ygganbws at 2022-06-27T07:33:44.607Z</Bytes>`
//
// - The signing scheme should be sr25519, and `scheme` should be empty.
//   - If this not supported by a signer, then `scheme` should be "ed25519".
//
// - Sign and store the signature in `sig` in binary encoding.
//
// #### Verification
//
// - Recompute the signed message using the details in `signature`.
// - Verify `sig` given the scheme, the message, and the public key.
//
// #### Legacy signatures before v0.1.4
//
// If `timestamp == 0`, assume an older version:
// - Decode `value` and `signer` from hexadecimal with or without `0x`.
// - Then the signed message is either `${CID}` or `<Bytes>${CID}</Bytes>`.
type Signature struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The cryptographic signature in binary encoding as per the scheme.
	Value []byte `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	// The public key of the signer in binary encoding as per the scheme.
	Signer []byte `protobuf:"bytes,2,opt,name=signer,proto3" json:"signer,omitempty"`
	// The name of the signature scheme (sr25519, secp256k1, ed25519).
	// Default and recommended value: "" or "sr25519".
	Scheme string `protobuf:"bytes,3,opt,name=scheme,proto3" json:"scheme,omitempty"`
	// The ID of the hashing algorithm as per multiformats/multihash.
	// Default and recommended value: 0 or 0xb220, meaning blake2b-256.
	MultiHashType uint64 `protobuf:"varint,4,opt,name=multiHashType,proto3" json:"multiHashType,omitempty"`
	// The timestamp in UNIX milliseconds.
	Timestamp uint64 `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *Signature) Reset() {
	*x = Signature{}
	if protoimpl.UnsafeEnabled {
		mi := &file_signature_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Signature) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Signature) ProtoMessage() {}

func (x *Signature) ProtoReflect() protoreflect.Message {
	mi := &file_signature_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Signature.ProtoReflect.Descriptor instead.
func (*Signature) Descriptor() ([]byte, []int) {
	return file_signature_proto_rawDescGZIP(), []int{0}
}

func (x *Signature) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Signature) GetSigner() []byte {
	if x != nil {
		return x.Signer
	}
	return nil
}

func (x *Signature) GetScheme() string {
	if x != nil {
		return x.Scheme
	}
	return ""
}

func (x *Signature) GetMultiHashType() uint64 {
	if x != nil {
		return x.MultiHashType
	}
	return 0
}

func (x *Signature) GetTimestamp() uint64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

var File_signature_proto protoreflect.FileDescriptor

var file_signature_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x02, 0x70, 0x62, 0x22, 0x95, 0x01, 0x0a, 0x09, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x69, 0x67,
	0x6e, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x73, 0x69, 0x67, 0x6e, 0x65,
	0x72, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x6d, 0x75, 0x6c,
	0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x0d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0x05, 0x5a,
	0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_signature_proto_rawDescOnce sync.Once
	file_signature_proto_rawDescData = file_signature_proto_rawDesc
)

func file_signature_proto_rawDescGZIP() []byte {
	file_signature_proto_rawDescOnce.Do(func() {
		file_signature_proto_rawDescData = protoimpl.X.CompressGZIP(file_signature_proto_rawDescData)
	})
	return file_signature_proto_rawDescData
}

var file_signature_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_signature_proto_goTypes = []interface{}{
	(*Signature)(nil), // 0: pb.Signature
}
var file_signature_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_signature_proto_init() }
func file_signature_proto_init() {
	if File_signature_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_signature_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Signature); i {
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
			RawDescriptor: file_signature_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_signature_proto_goTypes,
		DependencyIndexes: file_signature_proto_depIdxs,
		MessageInfos:      file_signature_proto_msgTypes,
	}.Build()
	File_signature_proto = out.File
	file_signature_proto_rawDesc = nil
	file_signature_proto_goTypes = nil
	file_signature_proto_depIdxs = nil
}
