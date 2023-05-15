// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.21.12
// source: tag.proto

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

// How can this tag be searched.
type SearchType int32

const (
	// RANGE tags should be indexed by nodes into their database to allow efficient queries on a key
	// for an exact value or a range of values. This is the default.
	SearchType_RANGE SearchType = 0
	// NOT_SEARCHABLE tags should not be indexed. This option should be set when possible to save
	// node space and speed.
	SearchType_NOT_SEARCHABLE SearchType = 1
)

// Enum value maps for SearchType.
var (
	SearchType_name = map[int32]string{
		0: "RANGE",
		1: "NOT_SEARCHABLE",
	}
	SearchType_value = map[string]int32{
		"RANGE":          0,
		"NOT_SEARCHABLE": 1,
	}
)

func (x SearchType) Enum() *SearchType {
	p := new(SearchType)
	*p = x
	return p
}

func (x SearchType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SearchType) Descriptor() protoreflect.EnumDescriptor {
	return file_tag_proto_enumTypes[0].Descriptor()
}

func (SearchType) Type() protoreflect.EnumType {
	return &file_tag_proto_enumTypes[0]
}

func (x SearchType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SearchType.Descriptor instead.
func (SearchType) EnumDescriptor() ([]byte, []int) {
	return file_tag_proto_rawDescGZIP(), []int{0}
}

//
// A tag is a `key/value` attribute attached to a piece.
//
// Tags can be used to search or filter pieces in storage. If search is not needed, disable it
// using the `searchable` field.
//
// Specific tags are used to implement different higher protocols, such as a file system.
// Each key should start with a prefix indicating which protocol or application relies on it. Below is a non-exhaustive table of known tag keys:
//
// Tag Key      | Description
// ------------ | -----------
// content-type | The MIME type of the payload of a piece or file. This is returned by the CDN web interface as HTTP Content-Type.
// file-*       | Tags to implement a filesystem over object storage. Example: `file-path`
// kv-*         | Tags to implement a key/value store over object storage.
// enc-*        | Tags to organize data encryption and data sharing keys.
//
// Below is the structure of a tag:
type Tag struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The key of the tag. It is usually a UTF-8 string, but it may be any data.
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	// The value of the tag for this key. The value should be interpreted based on the key.
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	// Whether this tag is searchable or not. Yes by default.
	Searchable SearchType `protobuf:"varint,3,opt,name=searchable,proto3,enum=pb.SearchType" json:"searchable,omitempty"`
}

func (x *Tag) Reset() {
	*x = Tag{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Tag) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Tag) ProtoMessage() {}

func (x *Tag) ProtoReflect() protoreflect.Message {
	mi := &file_tag_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Tag.ProtoReflect.Descriptor instead.
func (*Tag) Descriptor() ([]byte, []int) {
	return file_tag_proto_rawDescGZIP(), []int{0}
}

func (x *Tag) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

func (x *Tag) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Tag) GetSearchable() SearchType {
	if x != nil {
		return x.Searchable
	}
	return SearchType_RANGE
}

var File_tag_proto protoreflect.FileDescriptor

var file_tag_proto_rawDesc = []byte{
	0x0a, 0x09, 0x74, 0x61, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62, 0x22,
	0x5d, 0x0a, 0x03, 0x54, 0x61, 0x67, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x2e,
	0x0a, 0x0a, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x61, 0x62, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x54, 0x79,
	0x70, 0x65, 0x52, 0x0a, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x61, 0x62, 0x6c, 0x65, 0x2a, 0x2b,
	0x0a, 0x0a, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x54, 0x79, 0x70, 0x65, 0x12, 0x09, 0x0a, 0x05,
	0x52, 0x41, 0x4e, 0x47, 0x45, 0x10, 0x00, 0x12, 0x12, 0x0a, 0x0e, 0x4e, 0x4f, 0x54, 0x5f, 0x53,
	0x45, 0x41, 0x52, 0x43, 0x48, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x01, 0x42, 0x05, 0x5a, 0x03, 0x2f,
	0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_tag_proto_rawDescOnce sync.Once
	file_tag_proto_rawDescData = file_tag_proto_rawDesc
)

func file_tag_proto_rawDescGZIP() []byte {
	file_tag_proto_rawDescOnce.Do(func() {
		file_tag_proto_rawDescData = protoimpl.X.CompressGZIP(file_tag_proto_rawDescData)
	})
	return file_tag_proto_rawDescData
}

var file_tag_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_tag_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_tag_proto_goTypes = []interface{}{
	(SearchType)(0), // 0: pb.SearchType
	(*Tag)(nil),     // 1: pb.Tag
}
var file_tag_proto_depIdxs = []int32{
	0, // 0: pb.Tag.searchable:type_name -> pb.SearchType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_tag_proto_init() }
func file_tag_proto_init() {
	if File_tag_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_tag_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Tag); i {
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
			RawDescriptor: file_tag_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_tag_proto_goTypes,
		DependencyIndexes: file_tag_proto_depIdxs,
		EnumInfos:         file_tag_proto_enumTypes,
		MessageInfos:      file_tag_proto_msgTypes,
	}.Build()
	File_tag_proto = out.File
	file_tag_proto_rawDesc = nil
	file_tag_proto_goTypes = nil
	file_tag_proto_depIdxs = nil
}
