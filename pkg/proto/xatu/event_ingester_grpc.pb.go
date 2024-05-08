// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.3
// source: pkg/proto/xatu/event_ingester.proto

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

const (
	EventIngester_CreateEvents_FullMethodName = "/xatu.EventIngester/CreateEvents"
)

// EventIngesterClient is the client API for EventIngester service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EventIngesterClient interface {
	CreateEvents(ctx context.Context, in *CreateEventsRequest, opts ...grpc.CallOption) (*CreateEventsResponse, error)
}

type eventIngesterClient struct {
	cc grpc.ClientConnInterface
}

func NewEventIngesterClient(cc grpc.ClientConnInterface) EventIngesterClient {
	return &eventIngesterClient{cc}
}

func (c *eventIngesterClient) CreateEvents(ctx context.Context, in *CreateEventsRequest, opts ...grpc.CallOption) (*CreateEventsResponse, error) {
	out := new(CreateEventsResponse)
	err := c.cc.Invoke(ctx, EventIngester_CreateEvents_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EventIngesterServer is the server API for EventIngester service.
// All implementations must embed UnimplementedEventIngesterServer
// for forward compatibility
type EventIngesterServer interface {
	CreateEvents(context.Context, *CreateEventsRequest) (*CreateEventsResponse, error)
	mustEmbedUnimplementedEventIngesterServer()
}

// UnimplementedEventIngesterServer must be embedded to have forward compatible implementations.
type UnimplementedEventIngesterServer struct {
}

func (UnimplementedEventIngesterServer) CreateEvents(context.Context, *CreateEventsRequest) (*CreateEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateEvents not implemented")
}
func (UnimplementedEventIngesterServer) mustEmbedUnimplementedEventIngesterServer() {}

// UnsafeEventIngesterServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EventIngesterServer will
// result in compilation errors.
type UnsafeEventIngesterServer interface {
	mustEmbedUnimplementedEventIngesterServer()
}

func RegisterEventIngesterServer(s grpc.ServiceRegistrar, srv EventIngesterServer) {
	s.RegisterService(&EventIngester_ServiceDesc, srv)
}

func _EventIngester_CreateEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventIngesterServer).CreateEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EventIngester_CreateEvents_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventIngesterServer).CreateEvents(ctx, req.(*CreateEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// EventIngester_ServiceDesc is the grpc.ServiceDesc for EventIngester service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EventIngester_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "xatu.EventIngester",
	HandlerType: (*EventIngesterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateEvents",
			Handler:    _EventIngester_CreateEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/proto/xatu/event_ingester.proto",
}
