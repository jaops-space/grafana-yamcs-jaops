// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.3
// source: yamcs/protobuf/pvalue/pvalue_service.proto

package pvalue

import (
	_ "github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	protobuf "github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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

type LoadParameterValuesRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Yamcs instance name
	Instance *string `protobuf:"bytes,1,opt,name=instance" json:"instance,omitempty"`
	// Stream name
	Stream *string `protobuf:"bytes,2,opt,name=stream" json:"stream,omitempty"`
	// A group of values, and their properties
	Values        []*ParameterValueUpdate `protobuf:"bytes,3,rep,name=values" json:"values,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadParameterValuesRequest) Reset() {
	*x = LoadParameterValuesRequest{}
	mi := &file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadParameterValuesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadParameterValuesRequest) ProtoMessage() {}

func (x *LoadParameterValuesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadParameterValuesRequest.ProtoReflect.Descriptor instead.
func (*LoadParameterValuesRequest) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescGZIP(), []int{0}
}

func (x *LoadParameterValuesRequest) GetInstance() string {
	if x != nil && x.Instance != nil {
		return *x.Instance
	}
	return ""
}

func (x *LoadParameterValuesRequest) GetStream() string {
	if x != nil && x.Stream != nil {
		return *x.Stream
	}
	return ""
}

func (x *LoadParameterValuesRequest) GetValues() []*ParameterValueUpdate {
	if x != nil {
		return x.Values
	}
	return nil
}

type LoadParameterValuesResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The number of values that were loaded
	ValueCount *uint32 `protobuf:"varint,1,opt,name=valueCount" json:"valueCount,omitempty"`
	// The earliest generation time of all received values
	MinGenerationTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=minGenerationTime" json:"minGenerationTime,omitempty"`
	// The latest generation time of all received values
	MaxGenerationTime *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=maxGenerationTime" json:"maxGenerationTime,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *LoadParameterValuesResponse) Reset() {
	*x = LoadParameterValuesResponse{}
	mi := &file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadParameterValuesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadParameterValuesResponse) ProtoMessage() {}

func (x *LoadParameterValuesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadParameterValuesResponse.ProtoReflect.Descriptor instead.
func (*LoadParameterValuesResponse) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescGZIP(), []int{1}
}

func (x *LoadParameterValuesResponse) GetValueCount() uint32 {
	if x != nil && x.ValueCount != nil {
		return *x.ValueCount
	}
	return 0
}

func (x *LoadParameterValuesResponse) GetMinGenerationTime() *timestamppb.Timestamp {
	if x != nil {
		return x.MinGenerationTime
	}
	return nil
}

func (x *LoadParameterValuesResponse) GetMaxGenerationTime() *timestamppb.Timestamp {
	if x != nil {
		return x.MaxGenerationTime
	}
	return nil
}

type ParameterValueUpdate struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Fully qualified parameter name
	Parameter *string `protobuf:"bytes,1,opt,name=parameter" json:"parameter,omitempty"`
	// The new value
	Value *protobuf.Value `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	// The generation time of the value. If specified, must be a date
	// string in ISO 8601 format.
	GenerationTime *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=generationTime" json:"generationTime,omitempty"`
	// How long before this value expires, in milliseconds
	ExpiresIn     *uint64 `protobuf:"varint,4,opt,name=expiresIn" json:"expiresIn,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ParameterValueUpdate) Reset() {
	*x = ParameterValueUpdate{}
	mi := &file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ParameterValueUpdate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ParameterValueUpdate) ProtoMessage() {}

func (x *ParameterValueUpdate) ProtoReflect() protoreflect.Message {
	mi := &file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ParameterValueUpdate.ProtoReflect.Descriptor instead.
func (*ParameterValueUpdate) Descriptor() ([]byte, []int) {
	return file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescGZIP(), []int{2}
}

func (x *ParameterValueUpdate) GetParameter() string {
	if x != nil && x.Parameter != nil {
		return *x.Parameter
	}
	return ""
}

func (x *ParameterValueUpdate) GetValue() *protobuf.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *ParameterValueUpdate) GetGenerationTime() *timestamppb.Timestamp {
	if x != nil {
		return x.GenerationTime
	}
	return nil
}

func (x *ParameterValueUpdate) GetExpiresIn() uint64 {
	if x != nil && x.ExpiresIn != nil {
		return *x.ExpiresIn
	}
	return 0
}

var File_yamcs_protobuf_pvalue_pvalue_service_proto protoreflect.FileDescriptor

var file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDesc = string([]byte{
	0x0a, 0x2a, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x70, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2f, 0x70, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1a, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x95, 0x01,
	0x0a, 0x1a, 0x4c, 0x6f, 0x61, 0x64, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x72, 0x65,
	0x61, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x12, 0x43, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x2b, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x70, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x06, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x22, 0xd1, 0x01, 0x0a, 0x1b, 0x4c, 0x6f, 0x61, 0x64, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x43, 0x6f,
	0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x48, 0x0a, 0x11, 0x6d, 0x69, 0x6e, 0x47, 0x65, 0x6e, 0x65,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x11, 0x6d, 0x69,
	0x6e, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x48, 0x0a, 0x11, 0x6d, 0x61, 0x78, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x54, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x11, 0x6d, 0x61, 0x78, 0x47, 0x65, 0x6e, 0x65, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x22, 0xc3, 0x01, 0x0a, 0x14, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x55, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72,
	0x12, 0x2b, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x15, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x42, 0x0a,
	0x0e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x0e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d,
	0x65, 0x12, 0x1c, 0x0a, 0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73, 0x49, 0x6e, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73, 0x49, 0x6e, 0x32,
	0xd6, 0x01, 0x0a, 0x12, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x73, 0x41, 0x70, 0x69, 0x12, 0xbf, 0x01, 0x0a, 0x13, 0x4c, 0x6f, 0x61, 0x64, 0x50,
	0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x31,
	0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x70, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2e, 0x4c, 0x6f, 0x61, 0x64, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x32, 0x2e, 0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x70, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2e, 0x4c, 0x6f, 0x61, 0x64, 0x50, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x3f, 0x8a, 0x92, 0x03, 0x3b, 0x3a, 0x01, 0x2a, 0x1a, 0x36,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x2d, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x2f, 0x7b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x7d,
	0x2f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x73, 0x2f, 0x7b, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x7d, 0x3a, 0x6c, 0x6f, 0x61, 0x64, 0x28, 0x01, 0x42, 0x79, 0x0a, 0x12, 0x6f, 0x72, 0x67, 0x2e,
	0x79, 0x61, 0x6d, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x42, 0x1b,
	0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x53,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x44, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2d,
	0x73, 0x70, 0x61, 0x63, 0x65, 0x2f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2d, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2d, 0x6a, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x79, 0x61,
	0x6d, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x76, 0x61,
	0x6c, 0x75, 0x65,
})

var (
	file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescOnce sync.Once
	file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescData []byte
)

func file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescGZIP() []byte {
	file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescOnce.Do(func() {
		file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDesc), len(file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDesc)))
	})
	return file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDescData
}

var file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_yamcs_protobuf_pvalue_pvalue_service_proto_goTypes = []any{
	(*LoadParameterValuesRequest)(nil),  // 0: yamcs.protobuf.pvalue.LoadParameterValuesRequest
	(*LoadParameterValuesResponse)(nil), // 1: yamcs.protobuf.pvalue.LoadParameterValuesResponse
	(*ParameterValueUpdate)(nil),        // 2: yamcs.protobuf.pvalue.ParameterValueUpdate
	(*timestamppb.Timestamp)(nil),       // 3: google.protobuf.Timestamp
	(*protobuf.Value)(nil),              // 4: yamcs.protobuf.Value
}
var file_yamcs_protobuf_pvalue_pvalue_service_proto_depIdxs = []int32{
	2, // 0: yamcs.protobuf.pvalue.LoadParameterValuesRequest.values:type_name -> yamcs.protobuf.pvalue.ParameterValueUpdate
	3, // 1: yamcs.protobuf.pvalue.LoadParameterValuesResponse.minGenerationTime:type_name -> google.protobuf.Timestamp
	3, // 2: yamcs.protobuf.pvalue.LoadParameterValuesResponse.maxGenerationTime:type_name -> google.protobuf.Timestamp
	4, // 3: yamcs.protobuf.pvalue.ParameterValueUpdate.value:type_name -> yamcs.protobuf.Value
	3, // 4: yamcs.protobuf.pvalue.ParameterValueUpdate.generationTime:type_name -> google.protobuf.Timestamp
	0, // 5: yamcs.protobuf.pvalue.ParameterValuesApi.LoadParameterValues:input_type -> yamcs.protobuf.pvalue.LoadParameterValuesRequest
	1, // 6: yamcs.protobuf.pvalue.ParameterValuesApi.LoadParameterValues:output_type -> yamcs.protobuf.pvalue.LoadParameterValuesResponse
	6, // [6:7] is the sub-list for method output_type
	5, // [5:6] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_yamcs_protobuf_pvalue_pvalue_service_proto_init() }
func file_yamcs_protobuf_pvalue_pvalue_service_proto_init() {
	if File_yamcs_protobuf_pvalue_pvalue_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDesc), len(file_yamcs_protobuf_pvalue_pvalue_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_yamcs_protobuf_pvalue_pvalue_service_proto_goTypes,
		DependencyIndexes: file_yamcs_protobuf_pvalue_pvalue_service_proto_depIdxs,
		MessageInfos:      file_yamcs_protobuf_pvalue_pvalue_service_proto_msgTypes,
	}.Build()
	File_yamcs_protobuf_pvalue_pvalue_service_proto = out.File
	file_yamcs_protobuf_pvalue_pvalue_service_proto_goTypes = nil
	file_yamcs_protobuf_pvalue_pvalue_service_proto_depIdxs = nil
}
