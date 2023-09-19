// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.3
// source: state.proto

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

type State int32

const (
	State_NA    State = 0
	State_GRAY  State = 1
	State_GREEN State = 2
	State_BLUE  State = 3
	State_RED   State = 4
)

// Enum value maps for State.
var (
	State_name = map[int32]string{
		0: "NA",
		1: "GRAY",
		2: "GREEN",
		3: "BLUE",
		4: "RED",
	}
	State_value = map[string]int32{
		"NA":    0,
		"GRAY":  1,
		"GREEN": 2,
		"BLUE":  3,
		"RED":   4,
	}
)

func (x State) Enum() *State {
	p := new(State)
	*p = x
	return p
}

func (x State) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (State) Descriptor() protoreflect.EnumDescriptor {
	return file_state_proto_enumTypes[0].Descriptor()
}

func (State) Type() protoreflect.EnumType {
	return &file_state_proto_enumTypes[0]
}

func (x State) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use State.Descriptor instead.
func (State) EnumDescriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{0}
}

// Full State object used for two goals
// Routing optimization:
// User use this object to get information about cluster state and select the prioritised node for request
// Cluster state monitoring:
// Node use this object to get information about cluster state and check the state of other nodes
type FullState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Short current node state
	State *ShortState `protobuf:"bytes,1,opt,name=state,proto3" json:"state,omitempty"`
	// Signature of the short state by current node
	Signature *StateSignature `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	// Node statistic
	Statistic *Statistic `protobuf:"bytes,3,opt,name=statistic,proto3" json:"statistic,omitempty"`
	// Cluster map
	ClusterMap map[uint32]*GossipState `protobuf:"bytes,4,rep,name=cluster_map,json=clusterMap,proto3" json:"cluster_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *FullState) Reset() {
	*x = FullState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FullState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FullState) ProtoMessage() {}

func (x *FullState) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FullState.ProtoReflect.Descriptor instead.
func (*FullState) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{0}
}

func (x *FullState) GetState() *ShortState {
	if x != nil {
		return x.State
	}
	return nil
}

func (x *FullState) GetSignature() *StateSignature {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *FullState) GetStatistic() *Statistic {
	if x != nil {
		return x.Statistic
	}
	return nil
}

func (x *FullState) GetClusterMap() map[uint32]*GossipState {
	if x != nil {
		return x.ClusterMap
	}
	return nil
}

type ShortState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// ID of the node from smart contract
	NodeID uint32 `protobuf:"varint,1,opt,name=nodeID,proto3" json:"nodeID,omitempty"`
	// ID of the cluster from smart contract
	ClusterID uint32 `protobuf:"varint,2,opt,name=clusterID,proto3" json:"clusterID,omitempty"`
	// URL of the node
	Url string `protobuf:"bytes,3,opt,name=url,proto3" json:"url,omitempty"`
	// State of the node
	State State `protobuf:"varint,4,opt,name=state,proto3,enum=pb.State" json:"state,omitempty"`
	// Location of the node (Alpha2 country code)
	Location string `protobuf:"bytes,5,opt,name=location,proto3" json:"location,omitempty"`
	// Size of the node
	Size uint32 `protobuf:"varint,6,opt,name=size,proto3" json:"size,omitempty"`
	// Updated_at time of the short state
	Updated uint64 `protobuf:"varint,7,opt,name=updated,proto3" json:"updated,omitempty"`
}

func (x *ShortState) Reset() {
	*x = ShortState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShortState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShortState) ProtoMessage() {}

func (x *ShortState) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShortState.ProtoReflect.Descriptor instead.
func (*ShortState) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{1}
}

func (x *ShortState) GetNodeID() uint32 {
	if x != nil {
		return x.NodeID
	}
	return 0
}

func (x *ShortState) GetClusterID() uint32 {
	if x != nil {
		return x.ClusterID
	}
	return 0
}

func (x *ShortState) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *ShortState) GetState() State {
	if x != nil {
		return x.State
	}
	return State_NA
}

func (x *ShortState) GetLocation() string {
	if x != nil {
		return x.Location
	}
	return ""
}

func (x *ShortState) GetSize() uint32 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *ShortState) GetUpdated() uint64 {
	if x != nil {
		return x.Updated
	}
	return 0
}

// Each CDN node provide a full cluster state to other nodes and any client
// State bring short state (last updated state) and list of checks (who checks the state and what is the result)
// each check is signed by CDN node, that make them
// statistic is a last statistic state of CDN node, that needed for diagnostic only
// sign(CID(NodeID+ClusterID+URL+State+Location+Size+Updated_at))
type GossipState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LastState *ShortState       `protobuf:"bytes,1,opt,name=last_state,json=lastState,proto3" json:"last_state,omitempty"`
	Signature *StateSignature   `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	Statistic *Statistic        `protobuf:"bytes,3,opt,name=statistic,proto3" json:"statistic,omitempty"`
	Checks    map[uint32]*Check `protobuf:"bytes,4,rep,name=checks,proto3" json:"checks,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *GossipState) Reset() {
	*x = GossipState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GossipState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GossipState) ProtoMessage() {}

func (x *GossipState) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GossipState.ProtoReflect.Descriptor instead.
func (*GossipState) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{2}
}

func (x *GossipState) GetLastState() *ShortState {
	if x != nil {
		return x.LastState
	}
	return nil
}

func (x *GossipState) GetSignature() *StateSignature {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *GossipState) GetStatistic() *Statistic {
	if x != nil {
		return x.Statistic
	}
	return nil
}

func (x *GossipState) GetChecks() map[uint32]*Check {
	if x != nil {
		return x.Checks
	}
	return nil
}

// Signed check, that was produced by another CDN node
// and can be used to verify that the node works
type Check struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	State     *ShortState     `protobuf:"bytes,1,opt,name=state,proto3" json:"state,omitempty"`
	Signature *StateSignature `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *Check) Reset() {
	*x = Check{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Check) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Check) ProtoMessage() {}

func (x *Check) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Check.ProtoReflect.Descriptor instead.
func (*Check) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{3}
}

func (x *Check) GetState() *ShortState {
	if x != nil {
		return x.State
	}
	return nil
}

func (x *Check) GetSignature() *StateSignature {
	if x != nil {
		return x.Signature
	}
	return nil
}

// A signature and details to help verify it.
//
// #### Generation
//
// - Take hash for the stat object
//   - The hash function should be blake2b-256 and `multiHashType` should be empty.
//   - Base encoded in Base32 with the prefix `b`
//   - Example:
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
// - The signed message to store a state is:
//   - `<Bytes>DDC CDN ${state_hash} at ${timeText}</Bytes>`
//   - Note: the `<Bytes>` added for same format as piece signature is.
//   - Example: `<Bytes>DDC CDN "Hash" at 2022-06-27T07:33:44.607Z</Bytes>`
//
// - It is normal, if time of signing is bigger from state timestamp.
// - The State hash is:
//   - Short state: `Hash = CID(NodeID+ClusterID+URL+State+Location+Size+Updated_at)`
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
type StateSignature struct {
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

func (x *StateSignature) Reset() {
	*x = StateSignature{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateSignature) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateSignature) ProtoMessage() {}

func (x *StateSignature) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateSignature.ProtoReflect.Descriptor instead.
func (*StateSignature) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{4}
}

func (x *StateSignature) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *StateSignature) GetSigner() []byte {
	if x != nil {
		return x.Signer
	}
	return nil
}

func (x *StateSignature) GetScheme() string {
	if x != nil {
		return x.Scheme
	}
	return ""
}

func (x *StateSignature) GetMultiHashType() uint64 {
	if x != nil {
		return x.MultiHashType
	}
	return 0
}

func (x *StateSignature) GetTimestamp() uint64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

// Short statistic about node heals status
// Sessions - count of active sessions
// Redirects - count of redirects in current hour
// Cache_size - size of cache in bytes
// Cpu - cpu load in percents
// Ram - ram load in percents (regarding quotes)
// Hd - hard drive load in percents (regarding quotes)
// Uptime - uptime in seconds from last lunch
type Statistic struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Sessions  uint32     `protobuf:"varint,1,opt,name=sessions,proto3" json:"sessions,omitempty"`
	Redirects *Redirects `protobuf:"bytes,2,opt,name=redirects,proto3" json:"redirects,omitempty"`
	CacheSize uint32     `protobuf:"varint,3,opt,name=cache_size,json=cacheSize,proto3" json:"cache_size,omitempty"`
	Cpu       uint32     `protobuf:"varint,4,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Ram       uint32     `protobuf:"varint,5,opt,name=ram,proto3" json:"ram,omitempty"`
	Hd        uint32     `protobuf:"varint,6,opt,name=hd,proto3" json:"hd,omitempty"`
	Uptime    int64      `protobuf:"varint,7,opt,name=uptime,proto3" json:"uptime,omitempty"`
}

func (x *Statistic) Reset() {
	*x = Statistic{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Statistic) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Statistic) ProtoMessage() {}

func (x *Statistic) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Statistic.ProtoReflect.Descriptor instead.
func (*Statistic) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{5}
}

func (x *Statistic) GetSessions() uint32 {
	if x != nil {
		return x.Sessions
	}
	return 0
}

func (x *Statistic) GetRedirects() *Redirects {
	if x != nil {
		return x.Redirects
	}
	return nil
}

func (x *Statistic) GetCacheSize() uint32 {
	if x != nil {
		return x.CacheSize
	}
	return 0
}

func (x *Statistic) GetCpu() uint32 {
	if x != nil {
		return x.Cpu
	}
	return 0
}

func (x *Statistic) GetRam() uint32 {
	if x != nil {
		return x.Ram
	}
	return 0
}

func (x *Statistic) GetHd() uint32 {
	if x != nil {
		return x.Hd
	}
	return 0
}

func (x *Statistic) GetUptime() int64 {
	if x != nil {
		return x.Uptime
	}
	return 0
}

// Soft redirects it is redirects thar recommended via header
// Hard redirects it is redirects that forced via http code
// Counted redirects on last hour
type Redirects struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Soft uint32 `protobuf:"varint,1,opt,name=soft,proto3" json:"soft,omitempty"`
	Hard uint32 `protobuf:"varint,2,opt,name=hard,proto3" json:"hard,omitempty"`
}

func (x *Redirects) Reset() {
	*x = Redirects{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Redirects) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Redirects) ProtoMessage() {}

func (x *Redirects) ProtoReflect() protoreflect.Message {
	mi := &file_state_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Redirects.ProtoReflect.Descriptor instead.
func (*Redirects) Descriptor() ([]byte, []int) {
	return file_state_proto_rawDescGZIP(), []int{6}
}

func (x *Redirects) GetSoft() uint32 {
	if x != nil {
		return x.Soft
	}
	return 0
}

func (x *Redirects) GetHard() uint32 {
	if x != nil {
		return x.Hard
	}
	return 0
}

var File_state_proto protoreflect.FileDescriptor

var file_state_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70,
	0x62, 0x22, 0xa0, 0x02, 0x0a, 0x09, 0x46, 0x75, 0x6c, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12,
	0x24, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e,
	0x2e, 0x70, 0x62, 0x2e, 0x53, 0x68, 0x6f, 0x72, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x30, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x65, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x09, 0x73, 0x69,
	0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x2b, 0x0a, 0x09, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x73, 0x74, 0x69, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e,
	0x53, 0x74, 0x61, 0x74, 0x69, 0x73, 0x74, 0x69, 0x63, 0x52, 0x09, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x73, 0x74, 0x69, 0x63, 0x12, 0x3e, 0x0a, 0x0b, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f,
	0x6d, 0x61, 0x70, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x70, 0x62, 0x2e, 0x46,
	0x75, 0x6c, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x4d, 0x61, 0x70, 0x1a, 0x4e, 0x0a, 0x0f, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x4d,
	0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x25, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70, 0x62, 0x2e, 0x47, 0x6f,
	0x73, 0x73, 0x69, 0x70, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x22, 0xbf, 0x01, 0x0a, 0x0a, 0x53, 0x68, 0x6f, 0x72, 0x74, 0x53, 0x74,
	0x61, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x44, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x44, 0x12, 0x1c, 0x0a, 0x09, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x44, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09,
	0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x44, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12, 0x1f, 0x0a, 0x05, 0x73,
	0x74, 0x61, 0x74, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x09, 0x2e, 0x70, 0x62, 0x2e,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1a, 0x0a, 0x08,
	0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x18, 0x0a, 0x07,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x75,
	0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x22, 0x96, 0x02, 0x0a, 0x0b, 0x47, 0x6f, 0x73, 0x73, 0x69,
	0x70, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x2d, 0x0a, 0x0a, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x73,
	0x74, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x62, 0x2e,
	0x53, 0x68, 0x6f, 0x72, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x09, 0x6c, 0x61, 0x73, 0x74,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x30, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x65, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x09, 0x73, 0x69,
	0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x2b, 0x0a, 0x09, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x73, 0x74, 0x69, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e,
	0x53, 0x74, 0x61, 0x74, 0x69, 0x73, 0x74, 0x69, 0x63, 0x52, 0x09, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x73, 0x74, 0x69, 0x63, 0x12, 0x33, 0x0a, 0x06, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x18, 0x04,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x70, 0x62, 0x2e, 0x47, 0x6f, 0x73, 0x73, 0x69, 0x70,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x06, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x1a, 0x44, 0x0a, 0x0b, 0x43, 0x68, 0x65,
	0x63, 0x6b, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x1f, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x70, 0x62, 0x2e, 0x43,
	0x68, 0x65, 0x63, 0x6b, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22,
	0x5f, 0x0a, 0x05, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x12, 0x24, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x68, 0x6f,
	0x72, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x30,
	0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x12, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x53, 0x69, 0x67, 0x6e,
	0x61, 0x74, 0x75, 0x72, 0x65, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x22, 0x9a, 0x01, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x69, 0x67,
	0x6e, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x73, 0x69, 0x67, 0x6e, 0x65,
	0x72, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x6d, 0x75, 0x6c,
	0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x0d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x48, 0x61, 0x73, 0x68, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0xbf, 0x01,
	0x0a, 0x09, 0x53, 0x74, 0x61, 0x74, 0x69, 0x73, 0x74, 0x69, 0x63, 0x12, 0x1a, 0x0a, 0x08, 0x73,
	0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x73,
	0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x2b, 0x0a, 0x09, 0x72, 0x65, 0x64, 0x69, 0x72,
	0x65, 0x63, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x62, 0x2e,
	0x52, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x73, 0x52, 0x09, 0x72, 0x65, 0x64, 0x69, 0x72,
	0x65, 0x63, 0x74, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x73, 0x69,
	0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x63, 0x61, 0x63, 0x68, 0x65, 0x53,
	0x69, 0x7a, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x70, 0x75, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x03, 0x63, 0x70, 0x75, 0x12, 0x10, 0x0a, 0x03, 0x72, 0x61, 0x6d, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x03, 0x72, 0x61, 0x6d, 0x12, 0x0e, 0x0a, 0x02, 0x68, 0x64, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x02, 0x68, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x75, 0x70, 0x74, 0x69, 0x6d,
	0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65, 0x22,
	0x33, 0x0a, 0x09, 0x52, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x73, 0x12, 0x12, 0x0a, 0x04,
	0x73, 0x6f, 0x66, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x73, 0x6f, 0x66, 0x74,
	0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04,
	0x68, 0x61, 0x72, 0x64, 0x2a, 0x37, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x06, 0x0a,
	0x02, 0x4e, 0x41, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x47, 0x52, 0x41, 0x59, 0x10, 0x01, 0x12,
	0x09, 0x0a, 0x05, 0x47, 0x52, 0x45, 0x45, 0x4e, 0x10, 0x02, 0x12, 0x08, 0x0a, 0x04, 0x42, 0x4c,
	0x55, 0x45, 0x10, 0x03, 0x12, 0x07, 0x0a, 0x03, 0x52, 0x45, 0x44, 0x10, 0x04, 0x42, 0x05, 0x5a,
	0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_state_proto_rawDescOnce sync.Once
	file_state_proto_rawDescData = file_state_proto_rawDesc
)

func file_state_proto_rawDescGZIP() []byte {
	file_state_proto_rawDescOnce.Do(func() {
		file_state_proto_rawDescData = protoimpl.X.CompressGZIP(file_state_proto_rawDescData)
	})
	return file_state_proto_rawDescData
}

var file_state_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_state_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_state_proto_goTypes = []interface{}{
	(State)(0),             // 0: pb.State
	(*FullState)(nil),      // 1: pb.FullState
	(*ShortState)(nil),     // 2: pb.ShortState
	(*GossipState)(nil),    // 3: pb.GossipState
	(*Check)(nil),          // 4: pb.Check
	(*StateSignature)(nil), // 5: pb.StateSignature
	(*Statistic)(nil),      // 6: pb.Statistic
	(*Redirects)(nil),      // 7: pb.Redirects
	nil,                    // 8: pb.FullState.ClusterMapEntry
	nil,                    // 9: pb.GossipState.ChecksEntry
}
var file_state_proto_depIdxs = []int32{
	2,  // 0: pb.FullState.state:type_name -> pb.ShortState
	5,  // 1: pb.FullState.signature:type_name -> pb.StateSignature
	6,  // 2: pb.FullState.statistic:type_name -> pb.Statistic
	8,  // 3: pb.FullState.cluster_map:type_name -> pb.FullState.ClusterMapEntry
	0,  // 4: pb.ShortState.state:type_name -> pb.State
	2,  // 5: pb.GossipState.last_state:type_name -> pb.ShortState
	5,  // 6: pb.GossipState.signature:type_name -> pb.StateSignature
	6,  // 7: pb.GossipState.statistic:type_name -> pb.Statistic
	9,  // 8: pb.GossipState.checks:type_name -> pb.GossipState.ChecksEntry
	2,  // 9: pb.Check.state:type_name -> pb.ShortState
	5,  // 10: pb.Check.signature:type_name -> pb.StateSignature
	7,  // 11: pb.Statistic.redirects:type_name -> pb.Redirects
	3,  // 12: pb.FullState.ClusterMapEntry.value:type_name -> pb.GossipState
	4,  // 13: pb.GossipState.ChecksEntry.value:type_name -> pb.Check
	14, // [14:14] is the sub-list for method output_type
	14, // [14:14] is the sub-list for method input_type
	14, // [14:14] is the sub-list for extension type_name
	14, // [14:14] is the sub-list for extension extendee
	0,  // [0:14] is the sub-list for field type_name
}

func init() { file_state_proto_init() }
func file_state_proto_init() {
	if File_state_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_state_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FullState); i {
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
		file_state_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShortState); i {
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
		file_state_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GossipState); i {
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
		file_state_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Check); i {
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
		file_state_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateSignature); i {
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
		file_state_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Statistic); i {
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
		file_state_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Redirects); i {
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
			RawDescriptor: file_state_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_state_proto_goTypes,
		DependencyIndexes: file_state_proto_depIdxs,
		EnumInfos:         file_state_proto_enumTypes,
		MessageInfos:      file_state_proto_msgTypes,
	}.Build()
	File_state_proto = out.File
	file_state_proto_rawDesc = nil
	file_state_proto_goTypes = nil
	file_state_proto_depIdxs = nil
}
