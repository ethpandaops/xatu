// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v5.27.2
// source: pkg/proto/eth/v1/validator.proto

package v1

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

type ValidatorData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pubkey                     *wrapperspb.StringValue `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	WithdrawalCredentials      *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=withdrawal_credentials,proto3" json:"withdrawal_credentials,omitempty"`
	EffectiveBalance           *wrapperspb.UInt64Value `protobuf:"bytes,3,opt,name=effective_balance,proto3" json:"effective_balance,omitempty"`
	Slashed                    *wrapperspb.BoolValue   `protobuf:"bytes,4,opt,name=slashed,proto3" json:"slashed,omitempty"`
	ActivationEligibilityEpoch *wrapperspb.UInt64Value `protobuf:"bytes,5,opt,name=activation_eligibility_epoch,proto3" json:"activation_eligibility_epoch,omitempty"`
	ActivationEpoch            *wrapperspb.UInt64Value `protobuf:"bytes,6,opt,name=activation_epoch,proto3" json:"activation_epoch,omitempty"`
	ExitEpoch                  *wrapperspb.UInt64Value `protobuf:"bytes,7,opt,name=exit_epoch,proto3" json:"exit_epoch,omitempty"`
	WithdrawableEpoch          *wrapperspb.UInt64Value `protobuf:"bytes,8,opt,name=withdrawable_epoch,proto3" json:"withdrawable_epoch,omitempty"`
}

func (x *ValidatorData) Reset() {
	*x = ValidatorData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_validator_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidatorData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidatorData) ProtoMessage() {}

func (x *ValidatorData) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_validator_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidatorData.ProtoReflect.Descriptor instead.
func (*ValidatorData) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_validator_proto_rawDescGZIP(), []int{0}
}

func (x *ValidatorData) GetPubkey() *wrapperspb.StringValue {
	if x != nil {
		return x.Pubkey
	}
	return nil
}

func (x *ValidatorData) GetWithdrawalCredentials() *wrapperspb.StringValue {
	if x != nil {
		return x.WithdrawalCredentials
	}
	return nil
}

func (x *ValidatorData) GetEffectiveBalance() *wrapperspb.UInt64Value {
	if x != nil {
		return x.EffectiveBalance
	}
	return nil
}

func (x *ValidatorData) GetSlashed() *wrapperspb.BoolValue {
	if x != nil {
		return x.Slashed
	}
	return nil
}

func (x *ValidatorData) GetActivationEligibilityEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.ActivationEligibilityEpoch
	}
	return nil
}

func (x *ValidatorData) GetActivationEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.ActivationEpoch
	}
	return nil
}

func (x *ValidatorData) GetExitEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.ExitEpoch
	}
	return nil
}

func (x *ValidatorData) GetWithdrawableEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.WithdrawableEpoch
	}
	return nil
}

type Validator struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data    *ValidatorData          `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Index   *wrapperspb.UInt64Value `protobuf:"bytes,2,opt,name=index,proto3" json:"index,omitempty"`
	Balance *wrapperspb.UInt64Value `protobuf:"bytes,3,opt,name=balance,proto3" json:"balance,omitempty"`
	Status  *wrapperspb.StringValue `protobuf:"bytes,4,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *Validator) Reset() {
	*x = Validator{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_validator_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Validator) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Validator) ProtoMessage() {}

func (x *Validator) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_validator_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Validator.ProtoReflect.Descriptor instead.
func (*Validator) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_validator_proto_rawDescGZIP(), []int{1}
}

func (x *Validator) GetData() *ValidatorData {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Validator) GetIndex() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Index
	}
	return nil
}

func (x *Validator) GetBalance() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Balance
	}
	return nil
}

func (x *Validator) GetStatus() *wrapperspb.StringValue {
	if x != nil {
		return x.Status
	}
	return nil
}

var File_pkg_proto_eth_v1_validator_proto protoreflect.FileDescriptor

var file_pkg_proto_eth_v1_validator_proto_rawDesc = []byte{
	0x0a, 0x20, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f,
	0x76, 0x31, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0b, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x1a,
	0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xd5, 0x04, 0x0a, 0x0d, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x44, 0x61, 0x74,
	0x61, 0x12, 0x34, 0x0a, 0x06, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x06, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x12, 0x54, 0x0a, 0x16, 0x77, 0x69, 0x74, 0x68, 0x64,
	0x72, 0x61, 0x77, 0x61, 0x6c, 0x5f, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c,
	0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x16, 0x77, 0x69, 0x74, 0x68, 0x64, 0x72, 0x61, 0x77, 0x61,
	0x6c, 0x5f, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x12, 0x4a, 0x0a,
	0x11, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x69, 0x76, 0x65, 0x5f, 0x62, 0x61, 0x6c, 0x61, 0x6e,
	0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36,
	0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x11, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x69, 0x76,
	0x65, 0x5f, 0x62, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x34, 0x0a, 0x07, 0x73, 0x6c, 0x61,
	0x73, 0x68, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42, 0x6f, 0x6f,
	0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07, 0x73, 0x6c, 0x61, 0x73, 0x68, 0x65, 0x64, 0x12,
	0x60, 0x0a, 0x1c, 0x61, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x65, 0x6c,
	0x69, 0x67, 0x69, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x52, 0x1c, 0x61, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x65, 0x6c, 0x69, 0x67, 0x69, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x65, 0x70, 0x6f, 0x63,
	0x68, 0x12, 0x48, 0x0a, 0x10, 0x61, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49,
	0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x10, 0x61, 0x63, 0x74, 0x69, 0x76,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x3c, 0x0a, 0x0a, 0x65,
	0x78, 0x69, 0x74, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0a, 0x65,
	0x78, 0x69, 0x74, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x4c, 0x0a, 0x12, 0x77, 0x69, 0x74,
	0x68, 0x64, 0x72, 0x61, 0x77, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18,
	0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x52, 0x12, 0x77, 0x69, 0x74, 0x68, 0x64, 0x72, 0x61, 0x77, 0x61, 0x62, 0x6c,
	0x65, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x22, 0xdd, 0x01, 0x0a, 0x09, 0x56, 0x61, 0x6c, 0x69,
	0x64, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x2e, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e, 0x76,
	0x31, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x44, 0x61, 0x74, 0x61, 0x52,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x32, 0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x52, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x36, 0x0a, 0x07, 0x62, 0x61, 0x6c,
	0x61, 0x6e, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e,
	0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07, 0x62, 0x61, 0x6c, 0x61, 0x6e, 0x63,
	0x65, 0x12, 0x34, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x6f, 0x70,
	0x73, 0x2f, 0x78, 0x61, 0x74, 0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x65, 0x74, 0x68, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_eth_v1_validator_proto_rawDescOnce sync.Once
	file_pkg_proto_eth_v1_validator_proto_rawDescData = file_pkg_proto_eth_v1_validator_proto_rawDesc
)

func file_pkg_proto_eth_v1_validator_proto_rawDescGZIP() []byte {
	file_pkg_proto_eth_v1_validator_proto_rawDescOnce.Do(func() {
		file_pkg_proto_eth_v1_validator_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_eth_v1_validator_proto_rawDescData)
	})
	return file_pkg_proto_eth_v1_validator_proto_rawDescData
}

var file_pkg_proto_eth_v1_validator_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_pkg_proto_eth_v1_validator_proto_goTypes = []interface{}{
	(*ValidatorData)(nil),          // 0: xatu.eth.v1.ValidatorData
	(*Validator)(nil),              // 1: xatu.eth.v1.Validator
	(*wrapperspb.StringValue)(nil), // 2: google.protobuf.StringValue
	(*wrapperspb.UInt64Value)(nil), // 3: google.protobuf.UInt64Value
	(*wrapperspb.BoolValue)(nil),   // 4: google.protobuf.BoolValue
}
var file_pkg_proto_eth_v1_validator_proto_depIdxs = []int32{
	2,  // 0: xatu.eth.v1.ValidatorData.pubkey:type_name -> google.protobuf.StringValue
	2,  // 1: xatu.eth.v1.ValidatorData.withdrawal_credentials:type_name -> google.protobuf.StringValue
	3,  // 2: xatu.eth.v1.ValidatorData.effective_balance:type_name -> google.protobuf.UInt64Value
	4,  // 3: xatu.eth.v1.ValidatorData.slashed:type_name -> google.protobuf.BoolValue
	3,  // 4: xatu.eth.v1.ValidatorData.activation_eligibility_epoch:type_name -> google.protobuf.UInt64Value
	3,  // 5: xatu.eth.v1.ValidatorData.activation_epoch:type_name -> google.protobuf.UInt64Value
	3,  // 6: xatu.eth.v1.ValidatorData.exit_epoch:type_name -> google.protobuf.UInt64Value
	3,  // 7: xatu.eth.v1.ValidatorData.withdrawable_epoch:type_name -> google.protobuf.UInt64Value
	0,  // 8: xatu.eth.v1.Validator.data:type_name -> xatu.eth.v1.ValidatorData
	3,  // 9: xatu.eth.v1.Validator.index:type_name -> google.protobuf.UInt64Value
	3,  // 10: xatu.eth.v1.Validator.balance:type_name -> google.protobuf.UInt64Value
	2,  // 11: xatu.eth.v1.Validator.status:type_name -> google.protobuf.StringValue
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_pkg_proto_eth_v1_validator_proto_init() }
func file_pkg_proto_eth_v1_validator_proto_init() {
	if File_pkg_proto_eth_v1_validator_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_eth_v1_validator_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValidatorData); i {
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
		file_pkg_proto_eth_v1_validator_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Validator); i {
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
			RawDescriptor: file_pkg_proto_eth_v1_validator_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_eth_v1_validator_proto_goTypes,
		DependencyIndexes: file_pkg_proto_eth_v1_validator_proto_depIdxs,
		MessageInfos:      file_pkg_proto_eth_v1_validator_proto_msgTypes,
	}.Build()
	File_pkg_proto_eth_v1_validator_proto = out.File
	file_pkg_proto_eth_v1_validator_proto_rawDesc = nil
	file_pkg_proto_eth_v1_validator_proto_goTypes = nil
	file_pkg_proto_eth_v1_validator_proto_depIdxs = nil
}
