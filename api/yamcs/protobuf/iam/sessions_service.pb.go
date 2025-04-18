// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: yamcs/protobuf/iam/sessions_service.proto

package iam

import (
	_ "github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

type ListSessionsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Sessions      []*SessionInfo         `protobuf:"bytes,1,rep,name=sessions" json:"sessions,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListSessionsResponse) Reset() {
	*x = ListSessionsResponse{}
	mi := &file_yamcs_protobuf_iam_sessions_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListSessionsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListSessionsResponse) ProtoMessage() {}

func (x *ListSessionsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_iam_sessions_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListSessionsResponse.ProtoReflect.Descriptor instead.
func (*ListSessionsResponse) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_iam_sessions_service_proto_rawDescGZIP(), []int{0}
}

func (x *ListSessionsResponse) GetSessions() []*SessionInfo {
	if x != nil {
		return x.Sessions
	}
	return nil
}

type SessionInfo struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Session identifier
	Id             *string                `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Username       *string                `protobuf:"bytes,2,opt,name=username" json:"username,omitempty"`
	IpAddress      *string                `protobuf:"bytes,3,opt,name=ipAddress" json:"ipAddress,omitempty"`
	Hostname       *string                `protobuf:"bytes,4,opt,name=hostname" json:"hostname,omitempty"`
	StartTime      *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=startTime" json:"startTime,omitempty"`
	LastAccessTime *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=lastAccessTime" json:"lastAccessTime,omitempty"`
	ExpirationTime *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=expirationTime" json:"expirationTime,omitempty"`
	Clients        []string               `protobuf:"bytes,8,rep,name=clients" json:"clients,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *SessionInfo) Reset() {
	*x = SessionInfo{}
	mi := &file_yamcs_protobuf_iam_sessions_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SessionInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SessionInfo) ProtoMessage() {}

func (x *SessionInfo) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_iam_sessions_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SessionInfo.ProtoReflect.Descriptor instead.
func (*SessionInfo) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_iam_sessions_service_proto_rawDescGZIP(), []int{1}
}

func (x *SessionInfo) GetId() string {
	if x != nil && x.Id != nil {
		return *x.Id
	}
	return ""
}

func (x *SessionInfo) GetUsername() string {
	if x != nil && x.Username != nil {
		return *x.Username
	}
	return ""
}

func (x *SessionInfo) GetIpAddress() string {
	if x != nil && x.IpAddress != nil {
		return *x.IpAddress
	}
	return ""
}

func (x *SessionInfo) GetHostname() string {
	if x != nil && x.Hostname != nil {
		return *x.Hostname
	}
	return ""
}

func (x *SessionInfo) GetStartTime() *timestamppb.Timestamp {
	if x != nil {
		return x.StartTime
	}
	return nil
}

func (x *SessionInfo) GetLastAccessTime() *timestamppb.Timestamp {
	if x != nil {
		return x.LastAccessTime
	}
	return nil
}

func (x *SessionInfo) GetExpirationTime() *timestamppb.Timestamp {
	if x != nil {
		return x.ExpirationTime
	}
	return nil
}

func (x *SessionInfo) GetClients() []string {
	if x != nil {
		return x.Clients
	}
	return nil
}

type SessionEventInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	EndReason     *string                `protobuf:"bytes,1,opt,name=endReason" json:"endReason,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SessionEventInfo) Reset() {
	*x = SessionEventInfo{}
	mi := &file_yamcs_protobuf_iam_sessions_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SessionEventInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SessionEventInfo) ProtoMessage() {}

func (x *SessionEventInfo) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_iam_sessions_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SessionEventInfo.ProtoReflect.Descriptor instead.
func (*SessionEventInfo) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_iam_sessions_service_proto_rawDescGZIP(), []int{2}
}

func (x *SessionEventInfo) GetEndReason() string {
	if x != nil && x.EndReason != nil {
		return *x.EndReason
	}
	return ""
}

var File_yamcs_protobuf_iam_sessions_service_proto protoreflect.FileDescriptor

var file_yamcs_protobuf_iam_sessions_service_proto_rawDesc = string([]byte{
	0x0a, 0x29, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x69, 0x61, 0x6d, 0x2f, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x5f, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x79, 0x61, 0x6d,
	0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x69, 0x61, 0x6d, 0x1a,
	0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x79,
	0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x53, 0x0a, 0x14, 0x4c, 0x69,
	0x73, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x3b, 0x0a, 0x08, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x69, 0x61, 0x6d, 0x2e, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f,
	0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x08, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x22,
	0xcf, 0x02, 0x0a, 0x0b, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x69,
	0x70, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x69, 0x70, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x6f, 0x73,
	0x74, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x68, 0x6f, 0x73,
	0x74, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x38, 0x0a, 0x09, 0x73, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69,
	0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x73, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x42, 0x0a, 0x0e, 0x6c, 0x61, 0x73, 0x74, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x69, 0x6d,
	0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x52, 0x0e, 0x6c, 0x61, 0x73, 0x74, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54,
	0x69, 0x6d, 0x65, 0x12, 0x42, 0x0a, 0x0e, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x54, 0x69, 0x6d, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0e, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6c, 0x69, 0x65, 0x6e,
	0x74, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x73, 0x22, 0x30, 0x0a, 0x10, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1c, 0x0a, 0x09, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x61, 0x73,
	0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x61,
	0x73, 0x6f, 0x6e, 0x32, 0xd7, 0x01, 0x0a, 0x0b, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73,
	0x41, 0x70, 0x69, 0x12, 0x65, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x73, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x28, 0x2e, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x69, 0x61, 0x6d,
	0x2e, 0x4c, 0x69, 0x73, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x13, 0x8a, 0x92, 0x03, 0x0f, 0x0a, 0x0d, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x61, 0x0a, 0x10, 0x53, 0x75,
	0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x24, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x69, 0x61, 0x6d, 0x2e, 0x53, 0x65, 0x73, 0x73,
	0x69, 0x6f, 0x6e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0x0d, 0xda, 0x92,
	0x03, 0x09, 0x0a, 0x07, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x30, 0x01, 0x42, 0x6f, 0x0a,
	0x12, 0x6f, 0x72, 0x67, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x42, 0x14, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x41, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2d, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x2f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2d, 0x79, 0x61, 0x6d, 0x63,
	0x73, 0x2d, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x79, 0x61, 0x6d, 0x63,
	0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x69, 0x61, 0x6d,
})

var (
	file_yamcs_protobuf_iam_sessions_service_proto_rawDescOnce sync.Once
	file_yamcs_protobuf_iam_sessions_service_proto_rawDescData []byte
)

func file_yamcs_protobuf_iam_sessions_service_proto_rawDescGZIP() []byte {
	file_yamcs_protobuf_iam_sessions_service_proto_rawDescOnce.Do(func() {
		file_yamcs_protobuf_iam_sessions_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_iam_sessions_service_proto_rawDesc), len(file_yamcs_protobuf_iam_sessions_service_proto_rawDesc)))
	})
	return file_yamcs_protobuf_iam_sessions_service_proto_rawDescData
}

var file_yamcs_protobuf_iam_sessions_service_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_yamcs_protobuf_iam_sessions_service_proto_goTypes = []any{
	(*ListSessionsResponse)(nil),  // 0: yamcs.protobuf.iam.ListSessionsResponse
	(*SessionInfo)(nil),           // 1: yamcs.protobuf.iam.SessionInfo
	(*SessionEventInfo)(nil),      // 2: yamcs.protobuf.iam.SessionEventInfo
	(*timestamppb.Timestamp)(nil), // 3: google.protobuf.Timestamp
	(*emptypb.Empty)(nil),         // 4: google.protobuf.Empty
}
var file_yamcs_protobuf_iam_sessions_service_proto_depIdxs = []int32{
	1, // 0: yamcs.protobuf.iam.ListSessionsResponse.sessions:type_name -> yamcs.protobuf.iam.SessionInfo
	3, // 1: yamcs.protobuf.iam.SessionInfo.startTime:type_name -> google.protobuf.Timestamp
	3, // 2: yamcs.protobuf.iam.SessionInfo.lastAccessTime:type_name -> google.protobuf.Timestamp
	3, // 3: yamcs.protobuf.iam.SessionInfo.expirationTime:type_name -> google.protobuf.Timestamp
	4, // 4: yamcs.protobuf.iam.SessionsApi.ListSessions:input_type -> google.protobuf.Empty
	4, // 5: yamcs.protobuf.iam.SessionsApi.SubscribeSession:input_type -> google.protobuf.Empty
	0, // 6: yamcs.protobuf.iam.SessionsApi.ListSessions:output_type -> yamcs.protobuf.iam.ListSessionsResponse
	2, // 7: yamcs.protobuf.iam.SessionsApi.SubscribeSession:output_type -> yamcs.protobuf.iam.SessionEventInfo
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_yamcs_protobuf_iam_sessions_service_proto_init() }
func file_yamcs_protobuf_iam_sessions_service_proto_init() {
	if File_yamcs_protobuf_iam_sessions_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_iam_sessions_service_proto_rawDesc), len(file_yamcs_protobuf_iam_sessions_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_yamcs_protobuf_iam_sessions_service_proto_goTypes,
		DependencyIndexes: file_yamcs_protobuf_iam_sessions_service_proto_depIdxs,
		MessageInfos:      file_yamcs_protobuf_iam_sessions_service_proto_msgTypes,
	}.Build()
	File_yamcs_protobuf_iam_sessions_service_proto = out.File
	file_yamcs_protobuf_iam_sessions_service_proto_goTypes = nil
	file_yamcs_protobuf_iam_sessions_service_proto_depIdxs = nil
}
