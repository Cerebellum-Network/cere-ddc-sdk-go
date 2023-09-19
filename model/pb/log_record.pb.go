// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.3
// source: log_record.proto

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

// A log record is a record for logging activity of each request to node.
type LogRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// request's metadata
	//
	// Types that are assignable to Request:
	//
	//	*LogRecord_WriteRequest
	//	*LogRecord_ReadRequest
	//	*LogRecord_QueryRequest
	Request isLogRecord_Request `protobuf_oneof:"request"`
	// timestamp of request in UNIX nanoseconds.
	Timestamp uint64 `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// ip address of user who sent request
	Address string `protobuf:"bytes,5,opt,name=address,proto3" json:"address,omitempty"`
	// number of used gas for processing request
	Gas uint32 `protobuf:"varint,6,opt,name=gas,proto3" json:"gas,omitempty"`
	// users publicKey
	PublicKey []byte `protobuf:"bytes,7,opt,name=publicKey,proto3" json:"publicKey,omitempty"`
	// session id
	SessionId []byte `protobuf:"bytes,9,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	// request id
	RequestId string `protobuf:"bytes,10,opt,name=requestId,proto3" json:"requestId,omitempty"`
	// list of pieces returned by request
	ResponsePieces []*ResponsePiece `protobuf:"bytes,12,rep,name=responsePieces,proto3" json:"responsePieces,omitempty"`
}

func (x *LogRecord) Reset() {
	*x = LogRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_log_record_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogRecord) ProtoMessage() {}

func (x *LogRecord) ProtoReflect() protoreflect.Message {
	mi := &file_log_record_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogRecord.ProtoReflect.Descriptor instead.
func (*LogRecord) Descriptor() ([]byte, []int) {
	return file_log_record_proto_rawDescGZIP(), []int{0}
}

func (m *LogRecord) GetRequest() isLogRecord_Request {
	if m != nil {
		return m.Request
	}
	return nil
}

func (x *LogRecord) GetWriteRequest() *WriteRequest {
	if x, ok := x.GetRequest().(*LogRecord_WriteRequest); ok {
		return x.WriteRequest
	}
	return nil
}

func (x *LogRecord) GetReadRequest() *ReadRequest {
	if x, ok := x.GetRequest().(*LogRecord_ReadRequest); ok {
		return x.ReadRequest
	}
	return nil
}

func (x *LogRecord) GetQueryRequest() *QueryRequest {
	if x, ok := x.GetRequest().(*LogRecord_QueryRequest); ok {
		return x.QueryRequest
	}
	return nil
}

func (x *LogRecord) GetTimestamp() uint64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *LogRecord) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *LogRecord) GetGas() uint32 {
	if x != nil {
		return x.Gas
	}
	return 0
}

func (x *LogRecord) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

func (x *LogRecord) GetSessionId() []byte {
	if x != nil {
		return x.SessionId
	}
	return nil
}

func (x *LogRecord) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *LogRecord) GetResponsePieces() []*ResponsePiece {
	if x != nil {
		return x.ResponsePieces
	}
	return nil
}

type isLogRecord_Request interface {
	isLogRecord_Request()
}

type LogRecord_WriteRequest struct {
	// write request metadata
	WriteRequest *WriteRequest `protobuf:"bytes,1,opt,name=writeRequest,proto3,oneof"`
}

type LogRecord_ReadRequest struct {
	// read request metadata
	ReadRequest *ReadRequest `protobuf:"bytes,2,opt,name=readRequest,proto3,oneof"`
}

type LogRecord_QueryRequest struct {
	// search request metadata
	QueryRequest *QueryRequest `protobuf:"bytes,3,opt,name=queryRequest,proto3,oneof"`
}

func (*LogRecord_WriteRequest) isLogRecord_Request() {}

func (*LogRecord_ReadRequest) isLogRecord_Request() {}

func (*LogRecord_QueryRequest) isLogRecord_Request() {}

// A log records list
type LogRecordList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LogRecords []*LogRecord `protobuf:"bytes,1,rep,name=logRecords,proto3" json:"logRecords,omitempty"`
}

func (x *LogRecordList) Reset() {
	*x = LogRecordList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_log_record_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogRecordList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogRecordList) ProtoMessage() {}

func (x *LogRecordList) ProtoReflect() protoreflect.Message {
	mi := &file_log_record_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogRecordList.ProtoReflect.Descriptor instead.
func (*LogRecordList) Descriptor() ([]byte, []int) {
	return file_log_record_proto_rawDescGZIP(), []int{1}
}

func (x *LogRecordList) GetLogRecords() []*LogRecord {
	if x != nil {
		return x.LogRecords
	}
	return nil
}

// A write request is a container of metadata for upload piece requests
type WriteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// CID of saved piece
	Cid string `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	// bucket identifier where was stored piece
	BucketId uint32 `protobuf:"varint,2,opt,name=bucketId,proto3" json:"bucketId,omitempty"`
	// size of stored piece
	Size uint32 `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	// piece signature
	Signature *Signature `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *WriteRequest) Reset() {
	*x = WriteRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_log_record_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WriteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteRequest) ProtoMessage() {}

func (x *WriteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_log_record_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WriteRequest.ProtoReflect.Descriptor instead.
func (*WriteRequest) Descriptor() ([]byte, []int) {
	return file_log_record_proto_rawDescGZIP(), []int{2}
}

func (x *WriteRequest) GetCid() string {
	if x != nil {
		return x.Cid
	}
	return ""
}

func (x *WriteRequest) GetBucketId() uint32 {
	if x != nil {
		return x.BucketId
	}
	return 0
}

func (x *WriteRequest) GetSize() uint32 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *WriteRequest) GetSignature() *Signature {
	if x != nil {
		return x.Signature
	}
	return nil
}

// A read request is a container of metadata for download piece requests
type ReadRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// CID of requested piece
	Cid string `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	// bucket identifier of requested piece
	BucketId uint32 `protobuf:"varint,2,opt,name=bucketId,proto3" json:"bucketId,omitempty"`
	// piece signature
	Signature *Signature `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *ReadRequest) Reset() {
	*x = ReadRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_log_record_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadRequest) ProtoMessage() {}

func (x *ReadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_log_record_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadRequest.ProtoReflect.Descriptor instead.
func (*ReadRequest) Descriptor() ([]byte, []int) {
	return file_log_record_proto_rawDescGZIP(), []int{3}
}

func (x *ReadRequest) GetCid() string {
	if x != nil {
		return x.Cid
	}
	return ""
}

func (x *ReadRequest) GetBucketId() uint32 {
	if x != nil {
		return x.BucketId
	}
	return 0
}

func (x *ReadRequest) GetSignature() *Signature {
	if x != nil {
		return x.Signature
	}
	return nil
}

// A query request is a container of metadata for search requests
type QueryRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// query for search
	Query *Query `protobuf:"bytes,1,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *QueryRequest) Reset() {
	*x = QueryRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_log_record_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryRequest) ProtoMessage() {}

func (x *QueryRequest) ProtoReflect() protoreflect.Message {
	mi := &file_log_record_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryRequest.ProtoReflect.Descriptor instead.
func (*QueryRequest) Descriptor() ([]byte, []int) {
	return file_log_record_proto_rawDescGZIP(), []int{4}
}

func (x *QueryRequest) GetQuery() *Query {
	if x != nil {
		return x.Query
	}
	return nil
}

type ResponsePiece struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// CID of piece
	Cid string `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	// size of piece
	Size uint32 `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
}

func (x *ResponsePiece) Reset() {
	*x = ResponsePiece{}
	if protoimpl.UnsafeEnabled {
		mi := &file_log_record_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResponsePiece) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResponsePiece) ProtoMessage() {}

func (x *ResponsePiece) ProtoReflect() protoreflect.Message {
	mi := &file_log_record_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResponsePiece.ProtoReflect.Descriptor instead.
func (*ResponsePiece) Descriptor() ([]byte, []int) {
	return file_log_record_proto_rawDescGZIP(), []int{5}
}

func (x *ResponsePiece) GetCid() string {
	if x != nil {
		return x.Cid
	}
	return ""
}

func (x *ResponsePiece) GetSize() uint32 {
	if x != nil {
		return x.Size
	}
	return 0
}

var File_log_record_proto protoreflect.FileDescriptor

var file_log_record_proto_rawDesc = []byte{
	0x0a, 0x10, 0x6c, 0x6f, 0x67, 0x5f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x02, 0x70, 0x62, 0x1a, 0x0f, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0b, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x9e, 0x03, 0x0a, 0x09, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x63, 0x6f,
	0x72, 0x64, 0x12, 0x36, 0x0a, 0x0c, 0x77, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x62, 0x2e, 0x57, 0x72,
	0x69, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x0c, 0x77, 0x72,
	0x69, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x33, 0x0a, 0x0b, 0x72, 0x65,
	0x61, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0f, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x48, 0x00, 0x52, 0x0b, 0x72, 0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x36, 0x0a, 0x0c, 0x71, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x62, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x0c, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x42, 0x02, 0x30, 0x01, 0x52, 0x09,
	0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x67, 0x61, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x03, 0x67, 0x61, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b,
	0x65, 0x79, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x4b, 0x65, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64,
	0x18, 0x09, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49,
	0x64, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x18, 0x0a,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x12,
	0x39, 0x0a, 0x0e, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x69, 0x65, 0x63, 0x65,
	0x73, 0x18, 0x0c, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x69, 0x65, 0x63, 0x65, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x50, 0x69, 0x65, 0x63, 0x65, 0x73, 0x42, 0x09, 0x0a, 0x07, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x3e, 0x0a, 0x0d, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x63, 0x6f,
	0x72, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x2d, 0x0a, 0x0a, 0x6c, 0x6f, 0x67, 0x52, 0x65, 0x63,
	0x6f, 0x72, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e,
	0x4c, 0x6f, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x52, 0x0a, 0x6c, 0x6f, 0x67, 0x52, 0x65,
	0x63, 0x6f, 0x72, 0x64, 0x73, 0x22, 0x7d, 0x0a, 0x0c, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x63, 0x69, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x62, 0x75, 0x63, 0x6b, 0x65,
	0x74, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x62, 0x75, 0x63, 0x6b, 0x65,
	0x74, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x2b, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e,
	0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x22, 0x68, 0x0a, 0x0b, 0x52, 0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x63, 0x69, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x49,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x49,
	0x64, 0x12, 0x2b, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0x2f,
	0x0a, 0x0c, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f,
	0x0a, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e,
	0x70, 0x62, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x22,
	0x35, 0x0a, 0x0d, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x69, 0x65, 0x63, 0x65,
	0x12, 0x10, 0x0a, 0x03, 0x63, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63,
	0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x42, 0x05, 0x5a, 0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_log_record_proto_rawDescOnce sync.Once
	file_log_record_proto_rawDescData = file_log_record_proto_rawDesc
)

func file_log_record_proto_rawDescGZIP() []byte {
	file_log_record_proto_rawDescOnce.Do(func() {
		file_log_record_proto_rawDescData = protoimpl.X.CompressGZIP(file_log_record_proto_rawDescData)
	})
	return file_log_record_proto_rawDescData
}

var file_log_record_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_log_record_proto_goTypes = []interface{}{
	(*LogRecord)(nil),     // 0: pb.LogRecord
	(*LogRecordList)(nil), // 1: pb.LogRecordList
	(*WriteRequest)(nil),  // 2: pb.WriteRequest
	(*ReadRequest)(nil),   // 3: pb.ReadRequest
	(*QueryRequest)(nil),  // 4: pb.QueryRequest
	(*ResponsePiece)(nil), // 5: pb.ResponsePiece
	(*Signature)(nil),     // 6: pb.Signature
	(*Query)(nil),         // 7: pb.Query
}
var file_log_record_proto_depIdxs = []int32{
	2, // 0: pb.LogRecord.writeRequest:type_name -> pb.WriteRequest
	3, // 1: pb.LogRecord.readRequest:type_name -> pb.ReadRequest
	4, // 2: pb.LogRecord.queryRequest:type_name -> pb.QueryRequest
	5, // 3: pb.LogRecord.responsePieces:type_name -> pb.ResponsePiece
	0, // 4: pb.LogRecordList.logRecords:type_name -> pb.LogRecord
	6, // 5: pb.WriteRequest.signature:type_name -> pb.Signature
	6, // 6: pb.ReadRequest.signature:type_name -> pb.Signature
	7, // 7: pb.QueryRequest.query:type_name -> pb.Query
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_log_record_proto_init() }
func file_log_record_proto_init() {
	if File_log_record_proto != nil {
		return
	}
	file_signature_proto_init()
	file_query_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_log_record_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogRecord); i {
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
		file_log_record_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogRecordList); i {
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
		file_log_record_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WriteRequest); i {
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
		file_log_record_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReadRequest); i {
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
		file_log_record_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryRequest); i {
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
		file_log_record_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResponsePiece); i {
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
	file_log_record_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*LogRecord_WriteRequest)(nil),
		(*LogRecord_ReadRequest)(nil),
		(*LogRecord_QueryRequest)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_log_record_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_log_record_proto_goTypes,
		DependencyIndexes: file_log_record_proto_depIdxs,
		MessageInfos:      file_log_record_proto_msgTypes,
	}.Build()
	File_log_record_proto = out.File
	file_log_record_proto_rawDesc = nil
	file_log_record_proto_goTypes = nil
	file_log_record_proto_depIdxs = nil
}
