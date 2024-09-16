// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.2
// source: pkg/proto/mevrelay/payloads.proto

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

type ProposerPayloadDelivered struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Slot                 *wrapperspb.UInt64Value `protobuf:"bytes,1,opt,name=slot,proto3" json:"slot,omitempty"`
	ParentHash           *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=parent_hash,proto3" json:"parent_hash,omitempty"`
	BlockHash            *wrapperspb.StringValue `protobuf:"bytes,3,opt,name=block_hash,proto3" json:"block_hash,omitempty"`
	BuilderPubkey        *wrapperspb.StringValue `protobuf:"bytes,4,opt,name=builder_pubkey,proto3" json:"builder_pubkey,omitempty"`
	ProposerPubkey       *wrapperspb.StringValue `protobuf:"bytes,5,opt,name=proposer_pubkey,proto3" json:"proposer_pubkey,omitempty"`
	ProposerFeeRecipient *wrapperspb.StringValue `protobuf:"bytes,6,opt,name=proposer_fee_recipient,proto3" json:"proposer_fee_recipient,omitempty"`
	GasLimit             *wrapperspb.UInt64Value `protobuf:"bytes,7,opt,name=gas_limit,proto3" json:"gas_limit,omitempty"`
	GasUsed              *wrapperspb.UInt64Value `protobuf:"bytes,8,opt,name=gas_used,proto3" json:"gas_used,omitempty"`
	Value                *wrapperspb.StringValue `protobuf:"bytes,9,opt,name=value,proto3" json:"value,omitempty"`
	BlockNumber          *wrapperspb.UInt64Value `protobuf:"bytes,10,opt,name=block_number,proto3" json:"block_number,omitempty"`
	NumTx                *wrapperspb.UInt64Value `protobuf:"bytes,11,opt,name=num_tx,proto3" json:"num_tx,omitempty"`
}

func (x *ProposerPayloadDelivered) Reset() {
	*x = ProposerPayloadDelivered{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_mevrelay_payloads_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProposerPayloadDelivered) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProposerPayloadDelivered) ProtoMessage() {}

func (x *ProposerPayloadDelivered) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_mevrelay_payloads_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProposerPayloadDelivered.ProtoReflect.Descriptor instead.
func (*ProposerPayloadDelivered) Descriptor() ([]byte, []int) {
	return file_pkg_proto_mevrelay_payloads_proto_rawDescGZIP(), []int{0}
}

func (x *ProposerPayloadDelivered) GetSlot() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Slot
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetParentHash() *wrapperspb.StringValue {
	if x != nil {
		return x.ParentHash
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetBlockHash() *wrapperspb.StringValue {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetBuilderPubkey() *wrapperspb.StringValue {
	if x != nil {
		return x.BuilderPubkey
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetProposerPubkey() *wrapperspb.StringValue {
	if x != nil {
		return x.ProposerPubkey
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetProposerFeeRecipient() *wrapperspb.StringValue {
	if x != nil {
		return x.ProposerFeeRecipient
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetGasLimit() *wrapperspb.UInt64Value {
	if x != nil {
		return x.GasLimit
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetGasUsed() *wrapperspb.UInt64Value {
	if x != nil {
		return x.GasUsed
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetValue() *wrapperspb.StringValue {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetBlockNumber() *wrapperspb.UInt64Value {
	if x != nil {
		return x.BlockNumber
	}
	return nil
}

func (x *ProposerPayloadDelivered) GetNumTx() *wrapperspb.UInt64Value {
	if x != nil {
		return x.NumTx
	}
	return nil
}

var File_pkg_proto_mevrelay_payloads_proto protoreflect.FileDescriptor

var file_pkg_proto_mevrelay_payloads_proto_rawDesc = []byte{
	0x0a, 0x21, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6d, 0x65, 0x76, 0x72,
	0x65, 0x6c, 0x61, 0x79, 0x2f, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0d, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x6d, 0x65, 0x76, 0x72, 0x65, 0x6c,
	0x61, 0x79, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0xd0, 0x05, 0x0a, 0x18, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x50,
	0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x44, 0x65, 0x6c, 0x69, 0x76, 0x65, 0x72, 0x65, 0x64, 0x12,
	0x30, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x04, 0x73, 0x6c, 0x6f,
	0x74, 0x12, 0x3e, 0x0a, 0x0b, 0x70, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x0b, 0x70, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x68, 0x61, 0x73,
	0x68, 0x12, 0x3c, 0x0a, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x12,
	0x44, 0x0a, 0x0e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x65, 0x72, 0x5f, 0x70, 0x75, 0x62, 0x6b, 0x65,
	0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x65, 0x72, 0x5f, 0x70,
	0x75, 0x62, 0x6b, 0x65, 0x79, 0x12, 0x46, 0x0a, 0x0f, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65,
	0x72, 0x5f, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0f, 0x70, 0x72,
	0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x5f, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x12, 0x54, 0x0a,
	0x16, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x5f, 0x66, 0x65, 0x65, 0x5f, 0x72, 0x65,
	0x63, 0x69, 0x70, 0x69, 0x65, 0x6e, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x16, 0x70, 0x72, 0x6f,
	0x70, 0x6f, 0x73, 0x65, 0x72, 0x5f, 0x66, 0x65, 0x65, 0x5f, 0x72, 0x65, 0x63, 0x69, 0x70, 0x69,
	0x65, 0x6e, 0x74, 0x12, 0x3a, 0x0a, 0x09, 0x67, 0x61, 0x73, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x09, 0x67, 0x61, 0x73, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x12,
	0x38, 0x0a, 0x08, 0x67, 0x61, 0x73, 0x5f, 0x75, 0x73, 0x65, 0x64, 0x18, 0x08, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x08, 0x67, 0x61, 0x73, 0x5f, 0x75, 0x73, 0x65, 0x64, 0x12, 0x32, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x40, 0x0a,
	0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x0a, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x52, 0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12,
	0x34, 0x0a, 0x06, 0x6e, 0x75, 0x6d, 0x5f, 0x74, 0x78, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x06, 0x6e,
	0x75, 0x6d, 0x5f, 0x74, 0x78, 0x42, 0x30, 0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x6f, 0x70, 0x73, 0x2f,
	0x78, 0x61, 0x74, 0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6d,
	0x65, 0x76, 0x72, 0x65, 0x6c, 0x61, 0x79, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_mevrelay_payloads_proto_rawDescOnce sync.Once
	file_pkg_proto_mevrelay_payloads_proto_rawDescData = file_pkg_proto_mevrelay_payloads_proto_rawDesc
)

func file_pkg_proto_mevrelay_payloads_proto_rawDescGZIP() []byte {
	file_pkg_proto_mevrelay_payloads_proto_rawDescOnce.Do(func() {
		file_pkg_proto_mevrelay_payloads_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_mevrelay_payloads_proto_rawDescData)
	})
	return file_pkg_proto_mevrelay_payloads_proto_rawDescData
}

var file_pkg_proto_mevrelay_payloads_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_pkg_proto_mevrelay_payloads_proto_goTypes = []interface{}{
	(*ProposerPayloadDelivered)(nil), // 0: xatu.mevrelay.ProposerPayloadDelivered
	(*wrapperspb.UInt64Value)(nil),   // 1: google.protobuf.UInt64Value
	(*wrapperspb.StringValue)(nil),   // 2: google.protobuf.StringValue
}
var file_pkg_proto_mevrelay_payloads_proto_depIdxs = []int32{
	1,  // 0: xatu.mevrelay.ProposerPayloadDelivered.slot:type_name -> google.protobuf.UInt64Value
	2,  // 1: xatu.mevrelay.ProposerPayloadDelivered.parent_hash:type_name -> google.protobuf.StringValue
	2,  // 2: xatu.mevrelay.ProposerPayloadDelivered.block_hash:type_name -> google.protobuf.StringValue
	2,  // 3: xatu.mevrelay.ProposerPayloadDelivered.builder_pubkey:type_name -> google.protobuf.StringValue
	2,  // 4: xatu.mevrelay.ProposerPayloadDelivered.proposer_pubkey:type_name -> google.protobuf.StringValue
	2,  // 5: xatu.mevrelay.ProposerPayloadDelivered.proposer_fee_recipient:type_name -> google.protobuf.StringValue
	1,  // 6: xatu.mevrelay.ProposerPayloadDelivered.gas_limit:type_name -> google.protobuf.UInt64Value
	1,  // 7: xatu.mevrelay.ProposerPayloadDelivered.gas_used:type_name -> google.protobuf.UInt64Value
	2,  // 8: xatu.mevrelay.ProposerPayloadDelivered.value:type_name -> google.protobuf.StringValue
	1,  // 9: xatu.mevrelay.ProposerPayloadDelivered.block_number:type_name -> google.protobuf.UInt64Value
	1,  // 10: xatu.mevrelay.ProposerPayloadDelivered.num_tx:type_name -> google.protobuf.UInt64Value
	11, // [11:11] is the sub-list for method output_type
	11, // [11:11] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_pkg_proto_mevrelay_payloads_proto_init() }
func file_pkg_proto_mevrelay_payloads_proto_init() {
	if File_pkg_proto_mevrelay_payloads_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_mevrelay_payloads_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProposerPayloadDelivered); i {
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
			RawDescriptor: file_pkg_proto_mevrelay_payloads_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_mevrelay_payloads_proto_goTypes,
		DependencyIndexes: file_pkg_proto_mevrelay_payloads_proto_depIdxs,
		MessageInfos:      file_pkg_proto_mevrelay_payloads_proto_msgTypes,
	}.Build()
	File_pkg_proto_mevrelay_payloads_proto = out.File
	file_pkg_proto_mevrelay_payloads_proto_rawDesc = nil
	file_pkg_proto_mevrelay_payloads_proto_goTypes = nil
	file_pkg_proto_mevrelay_payloads_proto_depIdxs = nil
}
