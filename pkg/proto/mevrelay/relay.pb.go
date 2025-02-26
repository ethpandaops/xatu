// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: pkg/proto/mevrelay/relay.proto

package mevrelay

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Relay struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name *wrapperspb.StringValue `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Url  *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty"`
}

func (x *Relay) Reset() {
	*x = Relay{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_mevrelay_relay_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Relay) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Relay) ProtoMessage() {}

func (x *Relay) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_mevrelay_relay_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Relay.ProtoReflect.Descriptor instead.
func (*Relay) Descriptor() ([]byte, []int) {
	return file_pkg_proto_mevrelay_relay_proto_rawDescGZIP(), []int{0}
}

func (x *Relay) GetName() *wrapperspb.StringValue {
	if x != nil {
		return x.Name
	}
	return nil
}

func (x *Relay) GetUrl() *wrapperspb.StringValue {
	if x != nil {
		return x.Url
	}
	return nil
}

type ValidatorRegistrationMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FeeRecipient *wrapperspb.StringValue `protobuf:"bytes,1,opt,name=fee_recipient,proto3" json:"fee_recipient,omitempty"`
	GasLimit     *wrapperspb.UInt64Value `protobuf:"bytes,2,opt,name=gas_limit,proto3" json:"gas_limit,omitempty"`
	Timestamp    *wrapperspb.UInt64Value `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Pubkey       *wrapperspb.StringValue `protobuf:"bytes,4,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
}

func (x *ValidatorRegistrationMessage) Reset() {
	*x = ValidatorRegistrationMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_mevrelay_relay_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidatorRegistrationMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidatorRegistrationMessage) ProtoMessage() {}

func (x *ValidatorRegistrationMessage) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_mevrelay_relay_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidatorRegistrationMessage.ProtoReflect.Descriptor instead.
func (*ValidatorRegistrationMessage) Descriptor() ([]byte, []int) {
	return file_pkg_proto_mevrelay_relay_proto_rawDescGZIP(), []int{1}
}

func (x *ValidatorRegistrationMessage) GetFeeRecipient() *wrapperspb.StringValue {
	if x != nil {
		return x.FeeRecipient
	}
	return nil
}

func (x *ValidatorRegistrationMessage) GetGasLimit() *wrapperspb.UInt64Value {
	if x != nil {
		return x.GasLimit
	}
	return nil
}

func (x *ValidatorRegistrationMessage) GetTimestamp() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *ValidatorRegistrationMessage) GetPubkey() *wrapperspb.StringValue {
	if x != nil {
		return x.Pubkey
	}
	return nil
}

type ValidatorRegistration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message   *ValidatorRegistrationMessage `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Signature *wrapperspb.StringValue       `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *ValidatorRegistration) Reset() {
	*x = ValidatorRegistration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_mevrelay_relay_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidatorRegistration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidatorRegistration) ProtoMessage() {}

func (x *ValidatorRegistration) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_mevrelay_relay_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidatorRegistration.ProtoReflect.Descriptor instead.
func (*ValidatorRegistration) Descriptor() ([]byte, []int) {
	return file_pkg_proto_mevrelay_relay_proto_rawDescGZIP(), []int{2}
}

func (x *ValidatorRegistration) GetMessage() *ValidatorRegistrationMessage {
	if x != nil {
		return x.Message
	}
	return nil
}

func (x *ValidatorRegistration) GetSignature() *wrapperspb.StringValue {
	if x != nil {
		return x.Signature
	}
	return nil
}

var File_pkg_proto_mevrelay_relay_proto protoreflect.FileDescriptor

var file_pkg_proto_mevrelay_relay_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6d, 0x65, 0x76, 0x72,
	0x65, 0x6c, 0x61, 0x79, 0x2f, 0x72, 0x65, 0x6c, 0x61, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0d, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x6d, 0x65, 0x76, 0x72, 0x65, 0x6c, 0x61, 0x79, 0x1a,
	0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x69, 0x0a, 0x05, 0x52, 0x65, 0x6c, 0x61, 0x79, 0x12, 0x30, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2e, 0x0a, 0x03, 0x75, 0x72,
	0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x22, 0x90, 0x02, 0x0a, 0x1c, 0x56,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x42, 0x0a, 0x0d, 0x66,
	0x65, 0x65, 0x5f, 0x72, 0x65, 0x63, 0x69, 0x70, 0x69, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x52, 0x0d, 0x66, 0x65, 0x65, 0x5f, 0x72, 0x65, 0x63, 0x69, 0x70, 0x69, 0x65, 0x6e, 0x74, 0x12,
	0x3a, 0x0a, 0x09, 0x67, 0x61, 0x73, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x52, 0x09, 0x67, 0x61, 0x73, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x12, 0x3a, 0x0a, 0x09, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x34, 0x0a, 0x06, 0x70, 0x75, 0x62, 0x6b, 0x65,
	0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x06, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x22, 0x9a, 0x01,
	0x0a, 0x15, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x45, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x78, 0x61, 0x74, 0x75, 0x2e,
	0x6d, 0x65, 0x76, 0x72, 0x65, 0x6c, 0x61, 0x79, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x6f, 0x72, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x3a,
	0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x42, 0x30, 0x5a, 0x2e, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64,
	0x61, 0x6f, 0x70, 0x73, 0x2f, 0x78, 0x61, 0x74, 0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x6d, 0x65, 0x76, 0x72, 0x65, 0x6c, 0x61, 0x79, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_mevrelay_relay_proto_rawDescOnce sync.Once
	file_pkg_proto_mevrelay_relay_proto_rawDescData = file_pkg_proto_mevrelay_relay_proto_rawDesc
)

func file_pkg_proto_mevrelay_relay_proto_rawDescGZIP() []byte {
	file_pkg_proto_mevrelay_relay_proto_rawDescOnce.Do(func() {
		file_pkg_proto_mevrelay_relay_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_mevrelay_relay_proto_rawDescData)
	})
	return file_pkg_proto_mevrelay_relay_proto_rawDescData
}

var file_pkg_proto_mevrelay_relay_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_pkg_proto_mevrelay_relay_proto_goTypes = []any{
	(*Relay)(nil),                        // 0: xatu.mevrelay.Relay
	(*ValidatorRegistrationMessage)(nil), // 1: xatu.mevrelay.ValidatorRegistrationMessage
	(*ValidatorRegistration)(nil),        // 2: xatu.mevrelay.ValidatorRegistration
	(*wrapperspb.StringValue)(nil),       // 3: google.protobuf.StringValue
	(*wrapperspb.UInt64Value)(nil),       // 4: google.protobuf.UInt64Value
}
var file_pkg_proto_mevrelay_relay_proto_depIdxs = []int32{
	3, // 0: xatu.mevrelay.Relay.name:type_name -> google.protobuf.StringValue
	3, // 1: xatu.mevrelay.Relay.url:type_name -> google.protobuf.StringValue
	3, // 2: xatu.mevrelay.ValidatorRegistrationMessage.fee_recipient:type_name -> google.protobuf.StringValue
	4, // 3: xatu.mevrelay.ValidatorRegistrationMessage.gas_limit:type_name -> google.protobuf.UInt64Value
	4, // 4: xatu.mevrelay.ValidatorRegistrationMessage.timestamp:type_name -> google.protobuf.UInt64Value
	3, // 5: xatu.mevrelay.ValidatorRegistrationMessage.pubkey:type_name -> google.protobuf.StringValue
	1, // 6: xatu.mevrelay.ValidatorRegistration.message:type_name -> xatu.mevrelay.ValidatorRegistrationMessage
	3, // 7: xatu.mevrelay.ValidatorRegistration.signature:type_name -> google.protobuf.StringValue
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_pkg_proto_mevrelay_relay_proto_init() }
func file_pkg_proto_mevrelay_relay_proto_init() {
	if File_pkg_proto_mevrelay_relay_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_mevrelay_relay_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Relay); i {
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
		file_pkg_proto_mevrelay_relay_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*ValidatorRegistrationMessage); i {
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
		file_pkg_proto_mevrelay_relay_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*ValidatorRegistration); i {
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
			RawDescriptor: file_pkg_proto_mevrelay_relay_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_mevrelay_relay_proto_goTypes,
		DependencyIndexes: file_pkg_proto_mevrelay_relay_proto_depIdxs,
		MessageInfos:      file_pkg_proto_mevrelay_relay_proto_msgTypes,
	}.Build()
	File_pkg_proto_mevrelay_relay_proto = out.File
	file_pkg_proto_mevrelay_relay_proto_rawDesc = nil
	file_pkg_proto_mevrelay_relay_proto_goTypes = nil
	file_pkg_proto_mevrelay_relay_proto_depIdxs = nil
}
