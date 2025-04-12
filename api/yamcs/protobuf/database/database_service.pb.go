// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: yamcs/protobuf/database/database_service.proto

package database

import (
	_ "github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
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

type ListDatabasesResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Databases     []*DatabaseInfo        `protobuf:"bytes,1,rep,name=databases" json:"databases,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListDatabasesResponse) Reset() {
	*x = ListDatabasesResponse{}
	mi := &file_yamcs_protobuf_database_database_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListDatabasesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDatabasesResponse) ProtoMessage() {}

func (x *ListDatabasesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_database_database_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDatabasesResponse.ProtoReflect.Descriptor instead.
func (*ListDatabasesResponse) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_database_database_service_proto_rawDescGZIP(), []int{0}
}

func (x *ListDatabasesResponse) GetDatabases() []*DatabaseInfo {
	if x != nil {
		return x.Databases
	}
	return nil
}

type GetDatabaseRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Database name
	Name          *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetDatabaseRequest) Reset() {
	*x = GetDatabaseRequest{}
	mi := &file_yamcs_protobuf_database_database_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetDatabaseRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetDatabaseRequest) ProtoMessage() {}

func (x *GetDatabaseRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_database_database_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetDatabaseRequest.ProtoReflect.Descriptor instead.
func (*GetDatabaseRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_database_database_service_proto_rawDescGZIP(), []int{1}
}

func (x *GetDatabaseRequest) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

type DatabaseInfo struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Database name
	Name *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Path on server
	Path       *string `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
	Tablespace *string `protobuf:"bytes,3,opt,name=tablespace" json:"tablespace,omitempty"`
	// Names of the tables in this database
	Tables []string `protobuf:"bytes,4,rep,name=tables" json:"tables,omitempty"`
	// Names of the streams in this database
	Streams       []string `protobuf:"bytes,5,rep,name=streams" json:"streams,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DatabaseInfo) Reset() {
	*x = DatabaseInfo{}
	mi := &file_yamcs_protobuf_database_database_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DatabaseInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DatabaseInfo) ProtoMessage() {}

func (x *DatabaseInfo) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_database_database_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DatabaseInfo.ProtoReflect.Descriptor instead.
func (*DatabaseInfo) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_database_database_service_proto_rawDescGZIP(), []int{2}
}

func (x *DatabaseInfo) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *DatabaseInfo) GetPath() string {
	if x != nil && x.Path != nil {
		return *x.Path
	}
	return ""
}

func (x *DatabaseInfo) GetTablespace() string {
	if x != nil && x.Tablespace != nil {
		return *x.Tablespace
	}
	return ""
}

func (x *DatabaseInfo) GetTables() []string {
	if x != nil {
		return x.Tables
	}
	return nil
}

func (x *DatabaseInfo) GetStreams() []string {
	if x != nil {
		return x.Streams
	}
	return nil
}

var File_yamcs_protobuf_database_database_service_proto protoreflect.FileDescriptor

var file_yamcs_protobuf_database_database_service_proto_rawDesc = string([]byte{
	0x0a, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x64, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x62, 0x61,
	0x73, 0x65, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x16, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x5b, 0x0a, 0x15, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61,
	0x73, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x42, 0x0a, 0x09, 0x64,
	0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x24,
	0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x09, 0x64, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x73, 0x22,
	0x28, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x88, 0x01, 0x0a, 0x0c, 0x44, 0x61,
	0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61,
	0x74, 0x68, 0x12, 0x1e, 0x0a, 0x0a, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x18, 0x04, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x06, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x73, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x73, 0x32, 0xf9, 0x01, 0x0a, 0x0b, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73,
	0x65, 0x41, 0x70, 0x69, 0x12, 0x6c, 0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x61, 0x74, 0x61,
	0x62, 0x61, 0x73, 0x65, 0x73, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x2d, 0x2e,
	0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x61,
	0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x61, 0x74, 0x61, 0x62,
	0x61, 0x73, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x14, 0x8a, 0x92,
	0x03, 0x10, 0x0a, 0x0e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73,
	0x65, 0x73, 0x12, 0x7c, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73,
	0x65, 0x12, 0x2a, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x44, 0x61,
	0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x24, 0x2e,
	0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x61,
	0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x49,
	0x6e, 0x66, 0x6f, 0x22, 0x1b, 0x8a, 0x92, 0x03, 0x17, 0x0a, 0x15, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x64, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x73, 0x2f, 0x7b, 0x6e, 0x61, 0x6d, 0x65, 0x7d,
	0x42, 0x6e, 0x0a, 0x12, 0x6f, 0x72, 0x67, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x42, 0x0e, 0x44, 0x62, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2d, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x2f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2d, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2d, 0x6a,
	0x61, 0x6f, 0x70, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65,
})

var (
	file_yamcs_protobuf_database_database_service_proto_rawDescOnce sync.Once
	file_yamcs_protobuf_database_database_service_proto_rawDescData []byte
)

func file_yamcs_protobuf_database_database_service_proto_rawDescGZIP() []byte {
	file_yamcs_protobuf_database_database_service_proto_rawDescOnce.Do(func() {
		file_yamcs_protobuf_database_database_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_database_database_service_proto_rawDesc), len(file_yamcs_protobuf_database_database_service_proto_rawDesc)))
	})
	return file_yamcs_protobuf_database_database_service_proto_rawDescData
}

var file_yamcs_protobuf_database_database_service_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_yamcs_protobuf_database_database_service_proto_goTypes = []any{
	(*ListDatabasesResponse)(nil), // 0: yamcs.protobuf.archive.ListDatabasesResponse
	(*GetDatabaseRequest)(nil),    // 1: yamcs.protobuf.archive.GetDatabaseRequest
	(*DatabaseInfo)(nil),          // 2: yamcs.protobuf.archive.DatabaseInfo
	(*emptypb.Empty)(nil),         // 3: google.protobuf.Empty
}
var file_yamcs_protobuf_database_database_service_proto_depIdxs = []int32{
	2, // 0: yamcs.protobuf.archive.ListDatabasesResponse.databases:type_name -> yamcs.protobuf.archive.DatabaseInfo
	3, // 1: yamcs.protobuf.archive.DatabaseApi.ListDatabases:input_type -> google.protobuf.Empty
	1, // 2: yamcs.protobuf.archive.DatabaseApi.GetDatabase:input_type -> yamcs.protobuf.archive.GetDatabaseRequest
	0, // 3: yamcs.protobuf.archive.DatabaseApi.ListDatabases:output_type -> yamcs.protobuf.archive.ListDatabasesResponse
	2, // 4: yamcs.protobuf.archive.DatabaseApi.GetDatabase:output_type -> yamcs.protobuf.archive.DatabaseInfo
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_yamcs_protobuf_database_database_service_proto_init() }
func file_yamcs_protobuf_database_database_service_proto_init() {
	if File_yamcs_protobuf_database_database_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_database_database_service_proto_rawDesc), len(file_yamcs_protobuf_database_database_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_yamcs_protobuf_database_database_service_proto_goTypes,
		DependencyIndexes: file_yamcs_protobuf_database_database_service_proto_depIdxs,
		MessageInfos:      file_yamcs_protobuf_database_database_service_proto_msgTypes,
	}.Build()
	File_yamcs_protobuf_database_database_service_proto = out.File
	file_yamcs_protobuf_database_database_service_proto_goTypes = nil
	file_yamcs_protobuf_database_database_service_proto_depIdxs = nil
}
