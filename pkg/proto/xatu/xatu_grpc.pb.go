// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.9
// source: pkg/proto/xatu/xatu.proto

package xatu

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// XatuClient is the client API for Xatu service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type XatuClient interface {
	CreateEvents(ctx context.Context, in *CreateEventsRequest, opts ...grpc.CallOption) (*CreateEventsResponse, error)
}

type xatuClient struct {
	cc grpc.ClientConnInterface
}

func NewXatuClient(cc grpc.ClientConnInterface) XatuClient {
	return &xatuClient{cc}
}

func (c *xatuClient) CreateEvents(ctx context.Context, in *CreateEventsRequest, opts ...grpc.CallOption) (*CreateEventsResponse, error) {
	out := new(CreateEventsResponse)
	err := c.cc.Invoke(ctx, "/xatu.Xatu/CreateEvents", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// XatuServer is the server API for Xatu service.
// All implementations must embed UnimplementedXatuServer
// for forward compatibility
type XatuServer interface {
	CreateEvents(context.Context, *CreateEventsRequest) (*CreateEventsResponse, error)
	mustEmbedUnimplementedXatuServer()
}

// UnimplementedXatuServer must be embedded to have forward compatible implementations.
type UnimplementedXatuServer struct {
}

func (UnimplementedXatuServer) CreateEvents(context.Context, *CreateEventsRequest) (*CreateEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateEvents not implemented")
}
func (UnimplementedXatuServer) mustEmbedUnimplementedXatuServer() {}

// UnsafeXatuServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to XatuServer will
// result in compilation errors.
type UnsafeXatuServer interface {
	mustEmbedUnimplementedXatuServer()
}

func RegisterXatuServer(s grpc.ServiceRegistrar, srv XatuServer) {
	s.RegisterService(&Xatu_ServiceDesc, srv)
}

func _Xatu_CreateEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(XatuServer).CreateEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/xatu.Xatu/CreateEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(XatuServer).CreateEvents(ctx, req.(*CreateEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Xatu_ServiceDesc is the grpc.ServiceDesc for Xatu service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Xatu_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "xatu.Xatu",
	HandlerType: (*XatuServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateEvents",
			Handler:    _Xatu_CreateEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/proto/xatu/xatu.proto",
}
