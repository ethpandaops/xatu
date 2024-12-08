// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
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
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	EventIngester_CreateEvents_FullMethodName       = "/xatu.EventIngester/CreateEvents"
	EventIngester_CreateEventsStream_FullMethodName = "/xatu.EventIngester/CreateEventsStream"
)

// EventIngesterClient is the client API for EventIngester service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EventIngesterClient interface {
	// CreateEvents is a unary endpoint for creating events.
	CreateEvents(ctx context.Context, in *CreateEventsRequest, opts ...grpc.CallOption) (*CreateEventsResponse, error)
	// CreateEventsStream is a streaming endpoint for creating events.
	CreateEventsStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[CreateEventsRequest, CreateEventsResponse], error)
}

type eventIngesterClient struct {
	cc grpc.ClientConnInterface
}

func NewEventIngesterClient(cc grpc.ClientConnInterface) EventIngesterClient {
	return &eventIngesterClient{cc}
}

func (c *eventIngesterClient) CreateEvents(ctx context.Context, in *CreateEventsRequest, opts ...grpc.CallOption) (*CreateEventsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateEventsResponse)
	err := c.cc.Invoke(ctx, EventIngester_CreateEvents_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *eventIngesterClient) CreateEventsStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[CreateEventsRequest, CreateEventsResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &EventIngester_ServiceDesc.Streams[0], EventIngester_CreateEventsStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[CreateEventsRequest, CreateEventsResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type EventIngester_CreateEventsStreamClient = grpc.BidiStreamingClient[CreateEventsRequest, CreateEventsResponse]

// EventIngesterServer is the server API for EventIngester service.
// All implementations must embed UnimplementedEventIngesterServer
// for forward compatibility.
type EventIngesterServer interface {
	// CreateEvents is a unary endpoint for creating events.
	CreateEvents(context.Context, *CreateEventsRequest) (*CreateEventsResponse, error)
	// CreateEventsStream is a streaming endpoint for creating events.
	CreateEventsStream(grpc.BidiStreamingServer[CreateEventsRequest, CreateEventsResponse]) error
	mustEmbedUnimplementedEventIngesterServer()
}

// UnimplementedEventIngesterServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedEventIngesterServer struct{}

func (UnimplementedEventIngesterServer) CreateEvents(context.Context, *CreateEventsRequest) (*CreateEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateEvents not implemented")
}
func (UnimplementedEventIngesterServer) CreateEventsStream(grpc.BidiStreamingServer[CreateEventsRequest, CreateEventsResponse]) error {
	return status.Errorf(codes.Unimplemented, "method CreateEventsStream not implemented")
}
func (UnimplementedEventIngesterServer) mustEmbedUnimplementedEventIngesterServer() {}
func (UnimplementedEventIngesterServer) testEmbeddedByValue()                       {}

// UnsafeEventIngesterServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EventIngesterServer will
// result in compilation errors.
type UnsafeEventIngesterServer interface {
	mustEmbedUnimplementedEventIngesterServer()
}

func RegisterEventIngesterServer(s grpc.ServiceRegistrar, srv EventIngesterServer) {
	// If the following call pancis, it indicates UnimplementedEventIngesterServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
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

func _EventIngester_CreateEventsStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EventIngesterServer).CreateEventsStream(&grpc.GenericServerStream[CreateEventsRequest, CreateEventsResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type EventIngester_CreateEventsStreamServer = grpc.BidiStreamingServer[CreateEventsRequest, CreateEventsResponse]

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
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CreateEventsStream",
			Handler:       _EventIngester_CreateEventsStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "pkg/proto/xatu/event_ingester.proto",
}
