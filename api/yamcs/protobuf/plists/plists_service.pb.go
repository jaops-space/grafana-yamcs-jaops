// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: yamcs/protobuf/plists/plists_service.proto

package plists

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

type ListParameterListsRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Yamcs instance name
	Instance      *string `protobuf:"bytes,1,opt,name=instance" json:"instance,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListParameterListsRequest) Reset() {
	*x = ListParameterListsRequest{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListParameterListsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListParameterListsRequest) ProtoMessage() {}

func (x *ListParameterListsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListParameterListsRequest.ProtoReflect.Descriptor instead.
func (*ListParameterListsRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{0}
}

func (x *ListParameterListsRequest) GetInstance() string {
	if x != nil && x.Instance != nil {
		return *x.Instance
	}
	return ""
}

type GetParameterListRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Yamcs instance name
	Instance *string `protobuf:"bytes,1,opt,name=instance" json:"instance,omitempty"`
	// List identifier
	List          *string `protobuf:"bytes,2,opt,name=list" json:"list,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetParameterListRequest) Reset() {
	*x = GetParameterListRequest{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetParameterListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetParameterListRequest) ProtoMessage() {}

func (x *GetParameterListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetParameterListRequest.ProtoReflect.Descriptor instead.
func (*GetParameterListRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{1}
}

func (x *GetParameterListRequest) GetInstance() string {
	if x != nil && x.Instance != nil {
		return *x.Instance
	}
	return ""
}

func (x *GetParameterListRequest) GetList() string {
	if x != nil && x.List != nil {
		return *x.List
	}
	return ""
}

type ListParameterListsResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// List of lists, sorted by name
	//
	// The returned items include the patterns, however does
	// not resolve them. Use a specific parameter list request
	// to get that level of detail instead.
	Lists         []*ParameterListInfo `protobuf:"bytes,1,rep,name=lists" json:"lists,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListParameterListsResponse) Reset() {
	*x = ListParameterListsResponse{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListParameterListsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListParameterListsResponse) ProtoMessage() {}

func (x *ListParameterListsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListParameterListsResponse.ProtoReflect.Descriptor instead.
func (*ListParameterListsResponse) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{2}
}

func (x *ListParameterListsResponse) GetLists() []*ParameterListInfo {
	if x != nil {
		return x.Lists
	}
	return nil
}

type CreateParameterListRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Yamcs instance name
	Instance *string `protobuf:"bytes,1,opt,name=instance" json:"instance,omitempty"`
	// List name
	Name *string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	// Optional description
	Description *string `protobuf:"bytes,3,opt,name=description" json:"description,omitempty"`
	// Parameter names (either exact match or glob pattern)
	Patterns      []string `protobuf:"bytes,4,rep,name=patterns" json:"patterns,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateParameterListRequest) Reset() {
	*x = CreateParameterListRequest{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateParameterListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateParameterListRequest) ProtoMessage() {}

func (x *CreateParameterListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateParameterListRequest.ProtoReflect.Descriptor instead.
func (*CreateParameterListRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{3}
}

func (x *CreateParameterListRequest) GetInstance() string {
	if x != nil && x.Instance != nil {
		return *x.Instance
	}
	return ""
}

func (x *CreateParameterListRequest) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *CreateParameterListRequest) GetDescription() string {
	if x != nil && x.Description != nil {
		return *x.Description
	}
	return ""
}

func (x *CreateParameterListRequest) GetPatterns() []string {
	if x != nil {
		return x.Patterns
	}
	return nil
}

type UpdateParameterListRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Yamcs instance name
	Instance *string `protobuf:"bytes,1,opt,name=instance" json:"instance,omitempty"`
	// List identifier
	List *string `protobuf:"bytes,2,opt,name=list" json:"list,omitempty"`
	// List name
	Name *string `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
	// Optional description
	Description *string `protobuf:"bytes,4,opt,name=description" json:"description,omitempty"`
	// List of parameter patterns
	PatternDefinition *PatternDefinition `protobuf:"bytes,5,opt,name=patternDefinition" json:"patternDefinition,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *UpdateParameterListRequest) Reset() {
	*x = UpdateParameterListRequest{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateParameterListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateParameterListRequest) ProtoMessage() {}

func (x *UpdateParameterListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateParameterListRequest.ProtoReflect.Descriptor instead.
func (*UpdateParameterListRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{4}
}

func (x *UpdateParameterListRequest) GetInstance() string {
	if x != nil && x.Instance != nil {
		return *x.Instance
	}
	return ""
}

func (x *UpdateParameterListRequest) GetList() string {
	if x != nil && x.List != nil {
		return *x.List
	}
	return ""
}

func (x *UpdateParameterListRequest) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *UpdateParameterListRequest) GetDescription() string {
	if x != nil && x.Description != nil {
		return *x.Description
	}
	return ""
}

func (x *UpdateParameterListRequest) GetPatternDefinition() *PatternDefinition {
	if x != nil {
		return x.PatternDefinition
	}
	return nil
}

type PatternDefinition struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Parameter names (either exact match or glob pattern)
	Patterns      []string `protobuf:"bytes,1,rep,name=patterns" json:"patterns,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PatternDefinition) Reset() {
	*x = PatternDefinition{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PatternDefinition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PatternDefinition) ProtoMessage() {}

func (x *PatternDefinition) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PatternDefinition.ProtoReflect.Descriptor instead.
func (*PatternDefinition) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{5}
}

func (x *PatternDefinition) GetPatterns() []string {
	if x != nil {
		return x.Patterns
	}
	return nil
}

type DeleteParameterListRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Yamcs instance name
	Instance *string `protobuf:"bytes,1,opt,name=instance" json:"instance,omitempty"`
	// List identifier
	List          *string `protobuf:"bytes,2,opt,name=list" json:"list,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteParameterListRequest) Reset() {
	*x = DeleteParameterListRequest{}
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteParameterListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteParameterListRequest) ProtoMessage() {}

func (x *DeleteParameterListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_plists_plists_service_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteParameterListRequest.ProtoReflect.Descriptor instead.
func (*DeleteParameterListRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP(), []int{6}
}

func (x *DeleteParameterListRequest) GetInstance() string {
	if x != nil && x.Instance != nil {
		return *x.Instance
	}
	return ""
}

func (x *DeleteParameterListRequest) GetList() string {
	if x != nil && x.List != nil {
		return *x.List
	}
	return ""
}

var File_yamcs_protobuf_plists_plists_service_proto protoreflect.FileDescriptor

var file_yamcs_protobuf_plists_plists_service_proto_rawDesc = string([]byte{
	0x0a, 0x2a, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69,
	0x73, 0x74, 0x73, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1b, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f,
	0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x22, 0x79,
	0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x6c,
	0x69, 0x73, 0x74, 0x73, 0x2f, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x37, 0x0a, 0x19, 0x4c, 0x69, 0x73, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a,
	0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x22, 0x49, 0x0a, 0x17, 0x47, 0x65,
	0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6c, 0x69, 0x73, 0x74, 0x22, 0x5c, 0x0a, 0x1a, 0x4c, 0x69, 0x73, 0x74, 0x50, 0x61, 0x72,
	0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x3e, 0x0a, 0x05, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x28, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x05, 0x6c, 0x69,
	0x73, 0x74, 0x73, 0x22, 0x8a, 0x01, 0x0a, 0x1a, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x73,
	0x22, 0xda, 0x01, 0x0a, 0x1a, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x1a, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6c,
	0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x56, 0x0a, 0x11, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e,
	0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x28, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e,
	0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x11, 0x70, 0x61, 0x74, 0x74,
	0x65, 0x72, 0x6e, 0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x2f, 0x0a,
	0x11, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x73, 0x22, 0x4c,
	0x0a, 0x1a, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65,
	0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6c, 0x69, 0x73, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x32, 0xc7, 0x06, 0x0a,
	0x11, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x73, 0x41,
	0x70, 0x69, 0x12, 0xa6, 0x01, 0x0a, 0x12, 0x4c, 0x69, 0x73, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x73, 0x12, 0x30, 0x2e, 0x79, 0x61, 0x6d, 0x63,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74,
	0x73, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c,
	0x69, 0x73, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x31, 0x2e, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69,
	0x73, 0x74, 0x73, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65,
	0x72, 0x4c, 0x69, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x2b,
	0x8a, 0x92, 0x03, 0x27, 0x0a, 0x25, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x2d, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x7b, 0x69, 0x6e, 0x73, 0x74,
	0x61, 0x6e, 0x63, 0x65, 0x7d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x12, 0xa0, 0x01, 0x0a, 0x10,
	0x47, 0x65, 0x74, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74,
	0x12, 0x2e, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x47, 0x65, 0x74, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x28, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0x32, 0x8a, 0x92, 0x03, 0x2e,
	0x0a, 0x2c, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72,
	0x2d, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x7b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65,
	0x7d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x7b, 0x6c, 0x69, 0x73, 0x74, 0x7d, 0x12, 0xa2,
	0x01, 0x0a, 0x13, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x31, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69,
	0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28, 0x2e, 0x79, 0x61, 0x6d, 0x63,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74,
	0x73, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x49,
	0x6e, 0x66, 0x6f, 0x22, 0x2e, 0x8a, 0x92, 0x03, 0x2a, 0x3a, 0x01, 0x2a, 0x1a, 0x25, 0x2f, 0x61,
	0x70, 0x69, 0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x2d, 0x6c, 0x69, 0x73,
	0x74, 0x73, 0x2f, 0x7b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x7d, 0x2f, 0x6c, 0x69,
	0x73, 0x74, 0x73, 0x12, 0xa9, 0x01, 0x0a, 0x13, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x31, 0x2e, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69,
	0x73, 0x74, 0x73, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65,
	0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28,
	0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72,
	0x4c, 0x69, 0x73, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0x35, 0x8a, 0x92, 0x03, 0x31, 0x3a, 0x01,
	0x2a, 0x2a, 0x2c, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65,
	0x72, 0x2d, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x7b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x7d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x7b, 0x6c, 0x69, 0x73, 0x74, 0x7d, 0x12,
	0x94, 0x01, 0x0a, 0x13, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65,
	0x74, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x31, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2e,
	0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c,
	0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x32, 0x8a, 0x92, 0x03, 0x2e, 0x22, 0x2c, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70,
	0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x2d, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f, 0x7b,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x7d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x73, 0x2f,
	0x7b, 0x6c, 0x69, 0x73, 0x74, 0x7d, 0x42, 0x7f, 0x0a, 0x19, 0x6f, 0x72, 0x67, 0x2e, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x6c, 0x69,
	0x73, 0x74, 0x73, 0x42, 0x1a, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x4c, 0x69,
	0x73, 0x74, 0x73, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50,
	0x01, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61,
	0x6f, 0x70, 0x73, 0x2d, 0x73, 0x70, 0x61, 0x63, 0x65, 0x2f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e,
	0x61, 0x2d, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2d, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x73,
})

var (
	file_yamcs_protobuf_plists_plists_service_proto_rawDescOnce sync.Once
	file_yamcs_protobuf_plists_plists_service_proto_rawDescData []byte
)

func file_yamcs_protobuf_plists_plists_service_proto_rawDescGZIP() []byte {
	file_yamcs_protobuf_plists_plists_service_proto_rawDescOnce.Do(func() {
		file_yamcs_protobuf_plists_plists_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_plists_plists_service_proto_rawDesc), len(file_yamcs_protobuf_plists_plists_service_proto_rawDesc)))
	})
	return file_yamcs_protobuf_plists_plists_service_proto_rawDescData
}

var file_yamcs_protobuf_plists_plists_service_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_yamcs_protobuf_plists_plists_service_proto_goTypes = []any{
	(*ListParameterListsRequest)(nil),  // 0: yamcs.protobuf.plists.ListParameterListsRequest
	(*GetParameterListRequest)(nil),    // 1: yamcs.protobuf.plists.GetParameterListRequest
	(*ListParameterListsResponse)(nil), // 2: yamcs.protobuf.plists.ListParameterListsResponse
	(*CreateParameterListRequest)(nil), // 3: yamcs.protobuf.plists.CreateParameterListRequest
	(*UpdateParameterListRequest)(nil), // 4: yamcs.protobuf.plists.UpdateParameterListRequest
	(*PatternDefinition)(nil),          // 5: yamcs.protobuf.plists.PatternDefinition
	(*DeleteParameterListRequest)(nil), // 6: yamcs.protobuf.plists.DeleteParameterListRequest
	(*ParameterListInfo)(nil),          // 7: yamcs.protobuf.plists.ParameterListInfo
	(*emptypb.Empty)(nil),              // 8: google.protobuf.Empty
}
var file_yamcs_protobuf_plists_plists_service_proto_depIdxs = []int32{
	7, // 0: yamcs.protobuf.plists.ListParameterListsResponse.lists:type_name -> yamcs.protobuf.plists.ParameterListInfo
	5, // 1: yamcs.protobuf.plists.UpdateParameterListRequest.patternDefinition:type_name -> yamcs.protobuf.plists.PatternDefinition
	0, // 2: yamcs.protobuf.plists.ParameterListsApi.ListParameterLists:input_type -> yamcs.protobuf.plists.ListParameterListsRequest
	1, // 3: yamcs.protobuf.plists.ParameterListsApi.GetParameterList:input_type -> yamcs.protobuf.plists.GetParameterListRequest
	3, // 4: yamcs.protobuf.plists.ParameterListsApi.CreateParameterList:input_type -> yamcs.protobuf.plists.CreateParameterListRequest
	4, // 5: yamcs.protobuf.plists.ParameterListsApi.UpdateParameterList:input_type -> yamcs.protobuf.plists.UpdateParameterListRequest
	6, // 6: yamcs.protobuf.plists.ParameterListsApi.DeleteParameterList:input_type -> yamcs.protobuf.plists.DeleteParameterListRequest
	2, // 7: yamcs.protobuf.plists.ParameterListsApi.ListParameterLists:output_type -> yamcs.protobuf.plists.ListParameterListsResponse
	7, // 8: yamcs.protobuf.plists.ParameterListsApi.GetParameterList:output_type -> yamcs.protobuf.plists.ParameterListInfo
	7, // 9: yamcs.protobuf.plists.ParameterListsApi.CreateParameterList:output_type -> yamcs.protobuf.plists.ParameterListInfo
	7, // 10: yamcs.protobuf.plists.ParameterListsApi.UpdateParameterList:output_type -> yamcs.protobuf.plists.ParameterListInfo
	8, // 11: yamcs.protobuf.plists.ParameterListsApi.DeleteParameterList:output_type -> google.protobuf.Empty
	7, // [7:12] is the sub-list for method output_type
	2, // [2:7] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_yamcs_protobuf_plists_plists_service_proto_init() }
func file_yamcs_protobuf_plists_plists_service_proto_init() {
	if File_yamcs_protobuf_plists_plists_service_proto != nil {
		return
	}
	file_yamcs_protobuf_plists_plists_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_plists_plists_service_proto_rawDesc), len(file_yamcs_protobuf_plists_plists_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_yamcs_protobuf_plists_plists_service_proto_goTypes,
		DependencyIndexes: file_yamcs_protobuf_plists_plists_service_proto_depIdxs,
		MessageInfos:      file_yamcs_protobuf_plists_plists_service_proto_msgTypes,
	}.Build()
	File_yamcs_protobuf_plists_plists_service_proto = out.File
	file_yamcs_protobuf_plists_plists_service_proto_goTypes = nil
	file_yamcs_protobuf_plists_plists_service_proto_depIdxs = nil
}
