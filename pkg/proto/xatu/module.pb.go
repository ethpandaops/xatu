// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: pkg/proto/xatu/module.proto

package xatu

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

// ModuleName is the Xatu module.
type ModuleName int32

const (
	ModuleName_UNSPECIFIED   ModuleName = 0
	ModuleName_SENTRY        ModuleName = 1
	ModuleName_CANNON        ModuleName = 2
	ModuleName_SERVER        ModuleName = 3
	ModuleName_DISCOVERY     ModuleName = 4
	ModuleName_CL_MIMICRY    ModuleName = 5
	ModuleName_EL_MIMICRY    ModuleName = 6
	ModuleName_RELAY_MONITOR ModuleName = 7
	ModuleName_TYSM          ModuleName = 8
)

// Enum value maps for ModuleName.
var (
	ModuleName_name = map[int32]string{
		0: "UNSPECIFIED",
		1: "SENTRY",
		2: "CANNON",
		3: "SERVER",
		4: "DISCOVERY",
		5: "CL_MIMICRY",
		6: "EL_MIMICRY",
		7: "RELAY_MONITOR",
		8: "TYSM",
	}
	ModuleName_value = map[string]int32{
		"UNSPECIFIED":   0,
		"SENTRY":        1,
		"CANNON":        2,
		"SERVER":        3,
		"DISCOVERY":     4,
		"CL_MIMICRY":    5,
		"EL_MIMICRY":    6,
		"RELAY_MONITOR": 7,
		"TYSM":          8,
	}
)

func (x ModuleName) Enum() *ModuleName {
	p := new(ModuleName)
	*p = x
	return p
}

func (x ModuleName) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ModuleName) Descriptor() protoreflect.EnumDescriptor {
	return file_pkg_proto_xatu_module_proto_enumTypes[0].Descriptor()
}

func (ModuleName) Type() protoreflect.EnumType {
	return &file_pkg_proto_xatu_module_proto_enumTypes[0]
}

func (x ModuleName) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ModuleName.Descriptor instead.
func (ModuleName) EnumDescriptor() ([]byte, []int) {
	return file_pkg_proto_xatu_module_proto_rawDescGZIP(), []int{0}
}

var File_pkg_proto_xatu_module_proto protoreflect.FileDescriptor

var file_pkg_proto_xatu_module_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x78, 0x61, 0x74, 0x75,
	0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x78,
	0x61, 0x74, 0x75, 0x2a, 0x8d, 0x01, 0x0a, 0x0a, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45,
	0x44, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x53, 0x45, 0x4e, 0x54, 0x52, 0x59, 0x10, 0x01, 0x12,
	0x0a, 0x0a, 0x06, 0x43, 0x41, 0x4e, 0x4e, 0x4f, 0x4e, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x53,
	0x45, 0x52, 0x56, 0x45, 0x52, 0x10, 0x03, 0x12, 0x0d, 0x0a, 0x09, 0x44, 0x49, 0x53, 0x43, 0x4f,
	0x56, 0x45, 0x52, 0x59, 0x10, 0x04, 0x12, 0x0e, 0x0a, 0x0a, 0x43, 0x4c, 0x5f, 0x4d, 0x49, 0x4d,
	0x49, 0x43, 0x52, 0x59, 0x10, 0x05, 0x12, 0x0e, 0x0a, 0x0a, 0x45, 0x4c, 0x5f, 0x4d, 0x49, 0x4d,
	0x49, 0x43, 0x52, 0x59, 0x10, 0x06, 0x12, 0x11, 0x0a, 0x0d, 0x52, 0x45, 0x4c, 0x41, 0x59, 0x5f,
	0x4d, 0x4f, 0x4e, 0x49, 0x54, 0x4f, 0x52, 0x10, 0x07, 0x12, 0x08, 0x0a, 0x04, 0x54, 0x59, 0x53,
	0x4d, 0x10, 0x08, 0x42, 0x2c, 0x5a, 0x2a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x78, 0x61,
	0x74, 0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x78, 0x61, 0x74,
	0x75, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_xatu_module_proto_rawDescOnce sync.Once
	file_pkg_proto_xatu_module_proto_rawDescData = file_pkg_proto_xatu_module_proto_rawDesc
)

func file_pkg_proto_xatu_module_proto_rawDescGZIP() []byte {
	file_pkg_proto_xatu_module_proto_rawDescOnce.Do(func() {
		file_pkg_proto_xatu_module_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_xatu_module_proto_rawDescData)
	})
	return file_pkg_proto_xatu_module_proto_rawDescData
}

var file_pkg_proto_xatu_module_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_pkg_proto_xatu_module_proto_goTypes = []any{
	(ModuleName)(0), // 0: xatu.ModuleName
}
var file_pkg_proto_xatu_module_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_pkg_proto_xatu_module_proto_init() }
func file_pkg_proto_xatu_module_proto_init() {
	if File_pkg_proto_xatu_module_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pkg_proto_xatu_module_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_xatu_module_proto_goTypes,
		DependencyIndexes: file_pkg_proto_xatu_module_proto_depIdxs,
		EnumInfos:         file_pkg_proto_xatu_module_proto_enumTypes,
	}.Build()
	File_pkg_proto_xatu_module_proto = out.File
	file_pkg_proto_xatu_module_proto_rawDesc = nil
	file_pkg_proto_xatu_module_proto_goTypes = nil
	file_pkg_proto_xatu_module_proto_depIdxs = nil
}
