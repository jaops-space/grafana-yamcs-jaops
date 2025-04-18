// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: yamcs/api/websocket.proto

package api

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	anypb "google.golang.org/protobuf/types/known/anypb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ClientMessage struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Message type. Typically the name of a topic to subscribe to, or a built-in like "cancel".
	Type string `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	// Options specific to the type
	Options *anypb.Any `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	// Optional client-side message identifier, returned in reply messages.
	Id int32 `protobuf:"varint,3,opt,name=id,proto3" json:"id,omitempty"`
	// If applicable, the call associated with this message
	// This should be used when the client is streaming multiple messages
	// handled by the same call.
	Call int32 `protobuf:"varint,4,opt,name=call,proto3" json:"call,omitempty"`
	// If set, permit the server to keep a WebSocket connection despite frame writes
	// getting dropped (channel not open or not writable). If unset the default is 0,
	// meaning that if the server can't write a frame, it will close the connection
	// (impacting all calls on that connection).
	//
	// This attribute is only applied when it is set on the first message of a call.
	// Since Yamcs 5.7.6 this option is deprecated in favour of lowPriority below.
	//
	// Deprecated: Marked as deprecated in yamcs/api/websocket.proto.
	MaxDroppedWrites int32 `protobuf:"varint,5,opt,name=maxDroppedWrites,proto3" json:"maxDroppedWrites,omitempty"`
	// If set to true, permit the server to drop messages if writing the message would cause the
	// channel to exceed the highWaterMark
	// (see https://docs.yamcs.org/yamcs-server-manual/services/global/http-server/)
	// This attribute is only applied when it is set on the first message of a call.
	//
	// Note that if a message exceeds the highWaterMark, with this option set it will always be dropped.
	// A warning will be printed in the Yamcs logs in this case.
	LowPriority   bool `protobuf:"varint,6,opt,name=lowPriority,proto3" json:"lowPriority,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ClientMessage) Reset() {
	*x = ClientMessage{}
	mi := &file_yamcs_api_websocket_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientMessage) ProtoMessage() {}

func (x *ClientMessage) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_api_websocket_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClientMessage.ProtoReflect.Descriptor instead.
func (*ClientMessage) Descriptor() ([]byte, []int) {
	return file_yamcs_api_websocket_proto_rawDescGZIP(), []int{0}
}

func (x *ClientMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *ClientMessage) GetOptions() *anypb.Any {
	if x != nil {
		return x.Options
	}
	return nil
}

func (x *ClientMessage) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *ClientMessage) GetCall() int32 {
	if x != nil {
		return x.Call
	}
	return 0
}

// Deprecated: Marked as deprecated in yamcs/api/websocket.proto.
func (x *ClientMessage) GetMaxDroppedWrites() int32 {
	if x != nil {
		return x.MaxDroppedWrites
	}
	return 0
}

func (x *ClientMessage) GetLowPriority() bool {
	if x != nil {
		return x.LowPriority
	}
	return false
}

type ServerMessage struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Message type. Typically the name of the subscribed topic, or a built-in like "reply".
	Type string `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	// If applicable, the call associated with this message
	Call int32 `protobuf:"varint,2,opt,name=call,proto3" json:"call,omitempty"`
	// Sequence counter (scoped to the call)
	Seq           int32      `protobuf:"varint,3,opt,name=seq,proto3" json:"seq,omitempty"`
	Data          *anypb.Any `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerMessage) Reset() {
	*x = ServerMessage{}
	mi := &file_yamcs_api_websocket_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerMessage) ProtoMessage() {}

func (x *ServerMessage) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_api_websocket_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerMessage.ProtoReflect.Descriptor instead.
func (*ServerMessage) Descriptor() ([]byte, []int) {
	return file_yamcs_api_websocket_proto_rawDescGZIP(), []int{1}
}

func (x *ServerMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *ServerMessage) GetCall() int32 {
	if x != nil {
		return x.Call
	}
	return 0
}

func (x *ServerMessage) GetSeq() int32 {
	if x != nil {
		return x.Seq
	}
	return 0
}

func (x *ServerMessage) GetData() *anypb.Any {
	if x != nil {
		return x.Data
	}
	return nil
}

// Message to be provided in a ClientMessage if type is "cancel".
// This is a special message type that allows cancelling a call.
type CancelOptions struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Call          int32                  `protobuf:"varint,1,opt,name=call,proto3" json:"call,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CancelOptions) Reset() {
	*x = CancelOptions{}
	mi := &file_yamcs_api_websocket_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CancelOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CancelOptions) ProtoMessage() {}

func (x *CancelOptions) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_api_websocket_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CancelOptions.ProtoReflect.Descriptor instead.
func (*CancelOptions) Descriptor() ([]byte, []int) {
	return file_yamcs_api_websocket_proto_rawDescGZIP(), []int{2}
}

func (x *CancelOptions) GetCall() int32 {
	if x != nil {
		return x.Call
	}
	return 0
}

// Message to be provided in the data field of a ServerMessage if type is "reply".
type Reply struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The id of the original client message (if provided)
	ReplyTo int32 `protobuf:"varint,1,opt,name=reply_to,json=replyTo,proto3" json:"reply_to,omitempty"`
	// If set, the call was not successful.
	Exception     *ExceptionMessage `protobuf:"bytes,2,opt,name=exception,proto3" json:"exception,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Reply) Reset() {
	*x = Reply{}
	mi := &file_yamcs_api_websocket_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Reply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Reply) ProtoMessage() {}

func (x *Reply) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_api_websocket_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Reply.ProtoReflect.Descriptor instead.
func (*Reply) Descriptor() ([]byte, []int) {
	return file_yamcs_api_websocket_proto_rawDescGZIP(), []int{3}
}

func (x *Reply) GetReplyTo() int32 {
	if x != nil {
		return x.ReplyTo
	}
	return 0
}

func (x *Reply) GetException() *ExceptionMessage {
	if x != nil {
		return x.Exception
	}
	return nil
}

// Message to be provided in the data field of a ServerMessage if type is "state".
type State struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Calls         []*State_CallInfo      `protobuf:"bytes,1,rep,name=calls,proto3" json:"calls,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *State) Reset() {
	*x = State{}
	mi := &file_yamcs_api_websocket_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *State) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*State) ProtoMessage() {}

func (x *State) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_api_websocket_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use State.ProtoReflect.Descriptor instead.
func (*State) Descriptor() ([]byte, []int) {
	return file_yamcs_api_websocket_proto_rawDescGZIP(), []int{4}
}

func (x *State) GetCalls() []*State_CallInfo {
	if x != nil {
		return x.Calls
	}
	return nil
}

type State_CallInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Call          int32                  `protobuf:"varint,1,opt,name=call,proto3" json:"call,omitempty"`
	Type          string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Options       *anypb.Any             `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *State_CallInfo) Reset() {
	*x = State_CallInfo{}
	mi := &file_yamcs_api_websocket_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *State_CallInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*State_CallInfo) ProtoMessage() {}

func (x *State_CallInfo) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_api_websocket_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use State_CallInfo.ProtoReflect.Descriptor instead.
func (*State_CallInfo) Descriptor() ([]byte, []int) {
	return file_yamcs_api_websocket_proto_rawDescGZIP(), []int{4, 0}
}

func (x *State_CallInfo) GetCall() int32 {
	if x != nil {
		return x.Call
	}
	return 0
}

func (x *State_CallInfo) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *State_CallInfo) GetOptions() *anypb.Any {
	if x != nil {
		return x.Options
	}
	return nil
}

var File_yamcs_api_websocket_proto protoreflect.FileDescriptor

var file_yamcs_api_websocket_proto_rawDesc = string([]byte{
	0x0a, 0x19, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x77, 0x65, 0x62, 0x73,
	0x6f, 0x63, 0x6b, 0x65, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x79, 0x61, 0x6d,
	0x63, 0x73, 0x2e, 0x61, 0x70, 0x69, 0x1a, 0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x19, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x65, 0x78, 0x63,
	0x65, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc9, 0x01, 0x0a,
	0x0d, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x2e, 0x0a, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x61, 0x6c, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x04, 0x63, 0x61, 0x6c, 0x6c, 0x12, 0x2e, 0x0a, 0x10, 0x6d, 0x61, 0x78, 0x44, 0x72, 0x6f,
	0x70, 0x70, 0x65, 0x64, 0x57, 0x72, 0x69, 0x74, 0x65, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05,
	0x42, 0x02, 0x18, 0x01, 0x52, 0x10, 0x6d, 0x61, 0x78, 0x44, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64,
	0x57, 0x72, 0x69, 0x74, 0x65, 0x73, 0x12, 0x20, 0x0a, 0x0b, 0x6c, 0x6f, 0x77, 0x50, 0x72, 0x69,
	0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x6c, 0x6f, 0x77,
	0x50, 0x72, 0x69, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x22, 0x73, 0x0a, 0x0d, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x63, 0x61, 0x6c, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x61, 0x6c,
	0x6c, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x65, 0x71, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03,
	0x73, 0x65, 0x71, 0x12, 0x28, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x23, 0x0a,
	0x0d, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x63, 0x61, 0x6c, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x61,
	0x6c, 0x6c, 0x22, 0x5d, 0x0a, 0x05, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x19, 0x0a, 0x08, 0x72,
	0x65, 0x70, 0x6c, 0x79, 0x5f, 0x74, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x72,
	0x65, 0x70, 0x6c, 0x79, 0x54, 0x6f, 0x12, 0x39, 0x0a, 0x09, 0x65, 0x78, 0x63, 0x65, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x79, 0x61, 0x6d, 0x63,
	0x73, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x45, 0x78, 0x63, 0x65, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x09, 0x65, 0x78, 0x63, 0x65, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x22, 0x9c, 0x01, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x2f, 0x0a, 0x05, 0x63,
	0x61, 0x6c, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x79, 0x61, 0x6d,
	0x63, 0x73, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x43, 0x61, 0x6c,
	0x6c, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x05, 0x63, 0x61, 0x6c, 0x6c, 0x73, 0x1a, 0x62, 0x0a, 0x08,
	0x43, 0x61, 0x6c, 0x6c, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x61, 0x6c, 0x6c,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x61, 0x6c, 0x6c, 0x12, 0x12, 0x0a, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x12, 0x2e, 0x0a, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x42, 0x60, 0x0a, 0x12, 0x6f, 0x72, 0x67, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x42, 0x0e, 0x57, 0x65, 0x62, 0x53, 0x6f, 0x63, 0x6b, 0x65,
	0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2d, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x2f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2d, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2d, 0x6a,
	0x61, 0x6f, 0x70, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61,
	0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_yamcs_api_websocket_proto_rawDescOnce sync.Once
	file_yamcs_api_websocket_proto_rawDescData []byte
)

func file_yamcs_api_websocket_proto_rawDescGZIP() []byte {
	file_yamcs_api_websocket_proto_rawDescOnce.Do(func() {
		file_yamcs_api_websocket_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_yamcs_api_websocket_proto_rawDesc), len(file_yamcs_api_websocket_proto_rawDesc)))
	})
	return file_yamcs_api_websocket_proto_rawDescData
}

var file_yamcs_api_websocket_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_yamcs_api_websocket_proto_goTypes = []any{
	(*ClientMessage)(nil),    // 0: yamcs.api.ClientMessage
	(*ServerMessage)(nil),    // 1: yamcs.api.ServerMessage
	(*CancelOptions)(nil),    // 2: yamcs.api.CancelOptions
	(*Reply)(nil),            // 3: yamcs.api.Reply
	(*State)(nil),            // 4: yamcs.api.State
	(*State_CallInfo)(nil),   // 5: yamcs.api.State.CallInfo
	(*anypb.Any)(nil),        // 6: google.protobuf.Any
	(*ExceptionMessage)(nil), // 7: yamcs.api.ExceptionMessage
}
var file_yamcs_api_websocket_proto_depIdxs = []int32{
	6, // 0: yamcs.api.ClientMessage.options:type_name -> google.protobuf.Any
	6, // 1: yamcs.api.ServerMessage.data:type_name -> google.protobuf.Any
	7, // 2: yamcs.api.Reply.exception:type_name -> yamcs.api.ExceptionMessage
	5, // 3: yamcs.api.State.calls:type_name -> yamcs.api.State.CallInfo
	6, // 4: yamcs.api.State.CallInfo.options:type_name -> google.protobuf.Any
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_yamcs_api_websocket_proto_init() }
func file_yamcs_api_websocket_proto_init() {
	if File_yamcs_api_websocket_proto != nil {
		return
	}
	file_yamcs_api_exception_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_yamcs_api_websocket_proto_rawDesc), len(file_yamcs_api_websocket_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_yamcs_api_websocket_proto_goTypes,
		DependencyIndexes: file_yamcs_api_websocket_proto_depIdxs,
		MessageInfos:      file_yamcs_api_websocket_proto_msgTypes,
	}.Build()
	File_yamcs_api_websocket_proto = out.File
	file_yamcs_api_websocket_proto_goTypes = nil
	file_yamcs_api_websocket_proto_depIdxs = nil
}
