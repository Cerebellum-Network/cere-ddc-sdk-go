// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.23.3
// source: link.proto

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

// A link is a pointer from one piece to another.
type Link struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The CID of the linked piece.
	Cid string `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	// The size of the payload of the linked piece.
	Size uint64 `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
	// The name of the linked piece, in the context of the linking piece.
	Name string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *Link) Reset() {
	*x = Link{}
	if protoimpl.UnsafeEnabled {
		mi := &file_link_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Link) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Link) ProtoMessage() {}

func (x *Link) ProtoReflect() protoreflect.Message {
	mi := &file_link_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Link.ProtoReflect.Descriptor instead.
func (*Link) Descriptor() ([]byte, []int) {
	return file_link_proto_rawDescGZIP(), []int{0}
}

func (x *Link) GetCid() string {
	if x != nil {
		return x.Cid
	}
	return ""
}

func (x *Link) GetSize() uint64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *Link) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

var File_link_proto protoreflect.FileDescriptor

var file_link_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x6c, 0x69, 0x6e, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62,
	0x22, 0x40, 0x0a, 0x04, 0x4c, 0x69, 0x6e, 0x6b, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69,
	0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x42, 0x05, 0x5a, 0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_link_proto_rawDescOnce sync.Once
	file_link_proto_rawDescData = file_link_proto_rawDesc
)

func file_link_proto_rawDescGZIP() []byte {
	file_link_proto_rawDescOnce.Do(func() {
		file_link_proto_rawDescData = protoimpl.X.CompressGZIP(file_link_proto_rawDescData)
	})
	return file_link_proto_rawDescData
}

var file_link_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_link_proto_goTypes = []interface{}{
	(*Link)(nil), // 0: pb.Link
}
var file_link_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_link_proto_init() }
func file_link_proto_init() {
	if File_link_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_link_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Link); i {
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
			RawDescriptor: file_link_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_link_proto_goTypes,
		DependencyIndexes: file_link_proto_depIdxs,
		MessageInfos:      file_link_proto_msgTypes,
	}.Build()
	File_link_proto = out.File
	file_link_proto_rawDesc = nil
	file_link_proto_goTypes = nil
	file_link_proto_depIdxs = nil
}
