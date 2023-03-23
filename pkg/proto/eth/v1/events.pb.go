// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.9
// source: pkg/proto/eth/v1/events.proto

// Note: largely inspired by
// https://github.com/prysmaticlabs/prysm/tree/develop/proto/eth/v1

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EventHead struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Slot                      uint64 `protobuf:"varint,1,opt,name=slot,proto3" json:"slot,omitempty"`
	Block                     string `protobuf:"bytes,2,opt,name=block,proto3" json:"block,omitempty"`
	State                     string `protobuf:"bytes,3,opt,name=state,proto3" json:"state,omitempty"`
	EpochTransition           bool   `protobuf:"varint,4,opt,name=epoch_transition,proto3" json:"epoch_transition,omitempty"`
	PreviousDutyDependentRoot string `protobuf:"bytes,5,opt,name=previous_duty_dependent_root,proto3" json:"previous_duty_dependent_root,omitempty"`
	CurrentDutyDependentRoot  string `protobuf:"bytes,6,opt,name=current_duty_dependent_root,proto3" json:"current_duty_dependent_root,omitempty"`
}

func (x *EventHead) Reset() {
	*x = EventHead{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventHead) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventHead) ProtoMessage() {}

func (x *EventHead) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventHead.ProtoReflect.Descriptor instead.
func (*EventHead) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{0}
}

func (x *EventHead) GetSlot() uint64 {
	if x != nil {
		return x.Slot
	}
	return 0
}

func (x *EventHead) GetBlock() string {
	if x != nil {
		return x.Block
	}
	return ""
}

func (x *EventHead) GetState() string {
	if x != nil {
		return x.State
	}
	return ""
}

func (x *EventHead) GetEpochTransition() bool {
	if x != nil {
		return x.EpochTransition
	}
	return false
}

func (x *EventHead) GetPreviousDutyDependentRoot() string {
	if x != nil {
		return x.PreviousDutyDependentRoot
	}
	return ""
}

func (x *EventHead) GetCurrentDutyDependentRoot() string {
	if x != nil {
		return x.CurrentDutyDependentRoot
	}
	return ""
}

type EventBlock struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Slot                uint64 `protobuf:"varint,1,opt,name=slot,proto3" json:"slot,omitempty"`
	Block               string `protobuf:"bytes,2,opt,name=block,proto3" json:"block,omitempty"`
	ExecutionOptimistic bool   `protobuf:"varint,3,opt,name=execution_optimistic,proto3" json:"execution_optimistic,omitempty"`
}

func (x *EventBlock) Reset() {
	*x = EventBlock{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventBlock) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventBlock) ProtoMessage() {}

func (x *EventBlock) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventBlock.ProtoReflect.Descriptor instead.
func (*EventBlock) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{1}
}

func (x *EventBlock) GetSlot() uint64 {
	if x != nil {
		return x.Slot
	}
	return 0
}

func (x *EventBlock) GetBlock() string {
	if x != nil {
		return x.Block
	}
	return ""
}

func (x *EventBlock) GetExecutionOptimistic() bool {
	if x != nil {
		return x.ExecutionOptimistic
	}
	return false
}

type EventChainReorg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Slot         uint64 `protobuf:"varint,1,opt,name=slot,proto3" json:"slot,omitempty"`
	Depth        uint64 `protobuf:"varint,2,opt,name=depth,proto3" json:"depth,omitempty"`
	OldHeadBlock string `protobuf:"bytes,3,opt,name=old_head_block,proto3" json:"old_head_block,omitempty"`
	NewHeadBlock string `protobuf:"bytes,4,opt,name=new_head_block,proto3" json:"new_head_block,omitempty"`
	OldHeadState string `protobuf:"bytes,5,opt,name=old_head_state,proto3" json:"old_head_state,omitempty"`
	NewHeadState string `protobuf:"bytes,6,opt,name=new_head_state,proto3" json:"new_head_state,omitempty"`
	Epoch        uint64 `protobuf:"varint,7,opt,name=epoch,proto3" json:"epoch,omitempty"`
}

func (x *EventChainReorg) Reset() {
	*x = EventChainReorg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventChainReorg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventChainReorg) ProtoMessage() {}

func (x *EventChainReorg) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventChainReorg.ProtoReflect.Descriptor instead.
func (*EventChainReorg) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{2}
}

func (x *EventChainReorg) GetSlot() uint64 {
	if x != nil {
		return x.Slot
	}
	return 0
}

func (x *EventChainReorg) GetDepth() uint64 {
	if x != nil {
		return x.Depth
	}
	return 0
}

func (x *EventChainReorg) GetOldHeadBlock() string {
	if x != nil {
		return x.OldHeadBlock
	}
	return ""
}

func (x *EventChainReorg) GetNewHeadBlock() string {
	if x != nil {
		return x.NewHeadBlock
	}
	return ""
}

func (x *EventChainReorg) GetOldHeadState() string {
	if x != nil {
		return x.OldHeadState
	}
	return ""
}

func (x *EventChainReorg) GetNewHeadState() string {
	if x != nil {
		return x.NewHeadState
	}
	return ""
}

func (x *EventChainReorg) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

type EventFinalizedCheckpoint struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Block string `protobuf:"bytes,1,opt,name=block,proto3" json:"block,omitempty"`
	State string `protobuf:"bytes,2,opt,name=state,proto3" json:"state,omitempty"`
	Epoch uint64 `protobuf:"varint,3,opt,name=epoch,proto3" json:"epoch,omitempty"`
}

func (x *EventFinalizedCheckpoint) Reset() {
	*x = EventFinalizedCheckpoint{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventFinalizedCheckpoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventFinalizedCheckpoint) ProtoMessage() {}

func (x *EventFinalizedCheckpoint) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventFinalizedCheckpoint.ProtoReflect.Descriptor instead.
func (*EventFinalizedCheckpoint) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{3}
}

func (x *EventFinalizedCheckpoint) GetBlock() string {
	if x != nil {
		return x.Block
	}
	return ""
}

func (x *EventFinalizedCheckpoint) GetState() string {
	if x != nil {
		return x.State
	}
	return ""
}

func (x *EventFinalizedCheckpoint) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

type EventVoluntaryExit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Epoch          uint64 `protobuf:"varint,1,opt,name=epoch,proto3" json:"epoch,omitempty"`
	ValidatorIndex uint64 `protobuf:"varint,2,opt,name=validator_index,proto3" json:"validator_index,omitempty"`
}

func (x *EventVoluntaryExit) Reset() {
	*x = EventVoluntaryExit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventVoluntaryExit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventVoluntaryExit) ProtoMessage() {}

func (x *EventVoluntaryExit) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventVoluntaryExit.ProtoReflect.Descriptor instead.
func (*EventVoluntaryExit) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{4}
}

func (x *EventVoluntaryExit) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *EventVoluntaryExit) GetValidatorIndex() uint64 {
	if x != nil {
		return x.ValidatorIndex
	}
	return 0
}

type ContributionAndProof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AggregatorIndex uint64                     `protobuf:"varint,1,opt,name=aggregator_index,proto3" json:"aggregator_index,omitempty"`
	Contribution    *SyncCommitteeContribution `protobuf:"bytes,2,opt,name=contribution,proto3" json:"contribution,omitempty"`
	SelectionProof  string                     `protobuf:"bytes,3,opt,name=selection_proof,proto3" json:"selection_proof,omitempty"`
}

func (x *ContributionAndProof) Reset() {
	*x = ContributionAndProof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ContributionAndProof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContributionAndProof) ProtoMessage() {}

func (x *ContributionAndProof) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContributionAndProof.ProtoReflect.Descriptor instead.
func (*ContributionAndProof) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{5}
}

func (x *ContributionAndProof) GetAggregatorIndex() uint64 {
	if x != nil {
		return x.AggregatorIndex
	}
	return 0
}

func (x *ContributionAndProof) GetContribution() *SyncCommitteeContribution {
	if x != nil {
		return x.Contribution
	}
	return nil
}

func (x *ContributionAndProof) GetSelectionProof() string {
	if x != nil {
		return x.SelectionProof
	}
	return ""
}

type EventContributionAndProof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Signature string                `protobuf:"bytes,1,opt,name=signature,proto3" json:"signature,omitempty"`
	Message   *ContributionAndProof `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *EventContributionAndProof) Reset() {
	*x = EventContributionAndProof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventContributionAndProof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventContributionAndProof) ProtoMessage() {}

func (x *EventContributionAndProof) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_events_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventContributionAndProof.ProtoReflect.Descriptor instead.
func (*EventContributionAndProof) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_events_proto_rawDescGZIP(), []int{6}
}

func (x *EventContributionAndProof) GetSignature() string {
	if x != nil {
		return x.Signature
	}
	return ""
}

func (x *EventContributionAndProof) GetMessage() *ContributionAndProof {
	if x != nil {
		return x.Message
	}
	return nil
}

var File_pkg_proto_eth_v1_events_proto protoreflect.FileDescriptor

var file_pkg_proto_eth_v1_events_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f,
	0x76, 0x31, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0b, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x1a, 0x20, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65,
	0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x25,
	0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f, 0x76, 0x31,
	0x2f, 0x73, 0x79, 0x6e, 0x63, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xfd, 0x01, 0x0a, 0x09, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x48,
	0x65, 0x61, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x14, 0x0a,
	0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x12, 0x2a, 0x0a, 0x10, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x5f, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x65,
	0x70, 0x6f, 0x63, 0x68, 0x5f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x42, 0x0a, 0x1c, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x5f, 0x64, 0x75, 0x74, 0x79,
	0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x74, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x1c, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x5f,
	0x64, 0x75, 0x74, 0x79, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x74, 0x5f, 0x72,
	0x6f, 0x6f, 0x74, 0x12, 0x40, 0x0a, 0x1b, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x64,
	0x75, 0x74, 0x79, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x74, 0x5f, 0x72, 0x6f,
	0x6f, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x1b, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e,
	0x74, 0x5f, 0x64, 0x75, 0x74, 0x79, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x74,
	0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x22, 0x6a, 0x0a, 0x0a, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x42, 0x6c,
	0x6f, 0x63, 0x6b, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x32, 0x0a,
	0x14, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6d,
	0x69, 0x73, 0x74, 0x69, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x14, 0x65, 0x78, 0x65,
	0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6d, 0x69, 0x73, 0x74, 0x69,
	0x63, 0x22, 0xf1, 0x01, 0x0a, 0x0f, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x43, 0x68, 0x61, 0x69, 0x6e,
	0x52, 0x65, 0x6f, 0x72, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x64, 0x65, 0x70,
	0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x64, 0x65, 0x70, 0x74, 0x68, 0x12,
	0x26, 0x0a, 0x0e, 0x6f, 0x6c, 0x64, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x6f, 0x6c, 0x64, 0x5f, 0x68, 0x65, 0x61,
	0x64, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x26, 0x0a, 0x0e, 0x6e, 0x65, 0x77, 0x5f, 0x68,
	0x65, 0x61, 0x64, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0e, 0x6e, 0x65, 0x77, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12,
	0x26, 0x0a, 0x0e, 0x6f, 0x6c, 0x64, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x6f, 0x6c, 0x64, 0x5f, 0x68, 0x65, 0x61,
	0x64, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x26, 0x0a, 0x0e, 0x6e, 0x65, 0x77, 0x5f, 0x68,
	0x65, 0x61, 0x64, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0e, 0x6e, 0x65, 0x77, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12,
	0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x07, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05,
	0x65, 0x70, 0x6f, 0x63, 0x68, 0x22, 0x5c, 0x0a, 0x18, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x46, 0x69,
	0x6e, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x12, 0x14, 0x0a, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x14, 0x0a,
	0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70,
	0x6f, 0x63, 0x68, 0x22, 0x54, 0x0a, 0x12, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x56, 0x6f, 0x6c, 0x75,
	0x6e, 0x74, 0x61, 0x72, 0x79, 0x45, 0x78, 0x69, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f,
	0x63, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12,
	0x28, 0x0a, 0x0f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x5f, 0x69, 0x6e, 0x64,
	0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61,
	0x74, 0x6f, 0x72, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x22, 0xb8, 0x01, 0x0a, 0x14, 0x43, 0x6f,
	0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x41, 0x6e, 0x64, 0x50, 0x72, 0x6f,
	0x6f, 0x66, 0x12, 0x2a, 0x0a, 0x10, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x6f, 0x72,
	0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x10, 0x61, 0x67,
	0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x6f, 0x72, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x4a,
	0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e,
	0x76, 0x31, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65,
	0x43, 0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x63, 0x6f,
	0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x28, 0x0a, 0x0f, 0x73, 0x65,
	0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x70,
	0x72, 0x6f, 0x6f, 0x66, 0x22, 0x76, 0x0a, 0x19, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e,
	0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x41, 0x6e, 0x64, 0x50, 0x72, 0x6f, 0x6f,
	0x66, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12,
	0x3b, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x21, 0x2e, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x43,
	0x6f, 0x6e, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x41, 0x6e, 0x64, 0x50, 0x72,
	0x6f, 0x6f, 0x66, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x42, 0x2e, 0x5a, 0x2c,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61,
	0x6e, 0x64, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x78, 0x61, 0x74, 0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_eth_v1_events_proto_rawDescOnce sync.Once
	file_pkg_proto_eth_v1_events_proto_rawDescData = file_pkg_proto_eth_v1_events_proto_rawDesc
)

func file_pkg_proto_eth_v1_events_proto_rawDescGZIP() []byte {
	file_pkg_proto_eth_v1_events_proto_rawDescOnce.Do(func() {
		file_pkg_proto_eth_v1_events_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_eth_v1_events_proto_rawDescData)
	})
	return file_pkg_proto_eth_v1_events_proto_rawDescData
}

var file_pkg_proto_eth_v1_events_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_pkg_proto_eth_v1_events_proto_goTypes = []interface{}{
	(*EventHead)(nil),                 // 0: xatu.eth.v1.EventHead
	(*EventBlock)(nil),                // 1: xatu.eth.v1.EventBlock
	(*EventChainReorg)(nil),           // 2: xatu.eth.v1.EventChainReorg
	(*EventFinalizedCheckpoint)(nil),  // 3: xatu.eth.v1.EventFinalizedCheckpoint
	(*EventVoluntaryExit)(nil),        // 4: xatu.eth.v1.EventVoluntaryExit
	(*ContributionAndProof)(nil),      // 5: xatu.eth.v1.ContributionAndProof
	(*EventContributionAndProof)(nil), // 6: xatu.eth.v1.EventContributionAndProof
	(*SyncCommitteeContribution)(nil), // 7: xatu.eth.v1.SyncCommitteeContribution
}
var file_pkg_proto_eth_v1_events_proto_depIdxs = []int32{
	7, // 0: xatu.eth.v1.ContributionAndProof.contribution:type_name -> xatu.eth.v1.SyncCommitteeContribution
	5, // 1: xatu.eth.v1.EventContributionAndProof.message:type_name -> xatu.eth.v1.ContributionAndProof
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_pkg_proto_eth_v1_events_proto_init() }
func file_pkg_proto_eth_v1_events_proto_init() {
	if File_pkg_proto_eth_v1_events_proto != nil {
		return
	}
	file_pkg_proto_eth_v1_sync_committee_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_eth_v1_events_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventHead); i {
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
		file_pkg_proto_eth_v1_events_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventBlock); i {
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
		file_pkg_proto_eth_v1_events_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventChainReorg); i {
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
		file_pkg_proto_eth_v1_events_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventFinalizedCheckpoint); i {
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
		file_pkg_proto_eth_v1_events_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventVoluntaryExit); i {
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
		file_pkg_proto_eth_v1_events_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ContributionAndProof); i {
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
		file_pkg_proto_eth_v1_events_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventContributionAndProof); i {
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
			RawDescriptor: file_pkg_proto_eth_v1_events_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_eth_v1_events_proto_goTypes,
		DependencyIndexes: file_pkg_proto_eth_v1_events_proto_depIdxs,
		MessageInfos:      file_pkg_proto_eth_v1_events_proto_msgTypes,
	}.Build()
	File_pkg_proto_eth_v1_events_proto = out.File
	file_pkg_proto_eth_v1_events_proto_rawDesc = nil
	file_pkg_proto_eth_v1_events_proto_goTypes = nil
	file_pkg_proto_eth_v1_events_proto_depIdxs = nil
}
