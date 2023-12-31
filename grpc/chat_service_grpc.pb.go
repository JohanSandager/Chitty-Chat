// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.24.3
// source: grpc/chat_service.proto

package chat_service

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
	ChitChat_Chat_FullMethodName = "/chat_service.ChitChat/Chat"
)

// ChitChatClient is the client API for ChitChat service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChitChatClient interface {
	Chat(ctx context.Context, opts ...grpc.CallOption) (ChitChat_ChatClient, error)
}

type chitChatClient struct {
	cc grpc.ClientConnInterface
}

func NewChitChatClient(cc grpc.ClientConnInterface) ChitChatClient {
	return &chitChatClient{cc}
}

func (c *chitChatClient) Chat(ctx context.Context, opts ...grpc.CallOption) (ChitChat_ChatClient, error) {
	stream, err := c.cc.NewStream(ctx, &ChitChat_ServiceDesc.Streams[0], ChitChat_Chat_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &chitChatChatClient{stream}
	return x, nil
}

type ChitChat_ChatClient interface {
	Send(*ChitChatInformationContainer) error
	Recv() (*ChitChatInformationContainer, error)
	grpc.ClientStream
}

type chitChatChatClient struct {
	grpc.ClientStream
}

func (x *chitChatChatClient) Send(m *ChitChatInformationContainer) error {
	return x.ClientStream.SendMsg(m)
}

func (x *chitChatChatClient) Recv() (*ChitChatInformationContainer, error) {
	m := new(ChitChatInformationContainer)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChitChatServer is the server API for ChitChat service.
// All implementations must embed UnimplementedChitChatServer
// for forward compatibility
type ChitChatServer interface {
	Chat(ChitChat_ChatServer) error
	mustEmbedUnimplementedChitChatServer()
}

// UnimplementedChitChatServer must be embedded to have forward compatible implementations.
type UnimplementedChitChatServer struct {
}

func (UnimplementedChitChatServer) Chat(ChitChat_ChatServer) error {
	return status.Errorf(codes.Unimplemented, "method Chat not implemented")
}
func (UnimplementedChitChatServer) mustEmbedUnimplementedChitChatServer() {}

// UnsafeChitChatServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChitChatServer will
// result in compilation errors.
type UnsafeChitChatServer interface {
	mustEmbedUnimplementedChitChatServer()
}

func RegisterChitChatServer(s grpc.ServiceRegistrar, srv ChitChatServer) {
	s.RegisterService(&ChitChat_ServiceDesc, srv)
}

func _ChitChat_Chat_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ChitChatServer).Chat(&chitChatChatServer{stream})
}

type ChitChat_ChatServer interface {
	Send(*ChitChatInformationContainer) error
	Recv() (*ChitChatInformationContainer, error)
	grpc.ServerStream
}

type chitChatChatServer struct {
	grpc.ServerStream
}

func (x *chitChatChatServer) Send(m *ChitChatInformationContainer) error {
	return x.ServerStream.SendMsg(m)
}

func (x *chitChatChatServer) Recv() (*ChitChatInformationContainer, error) {
	m := new(ChitChatInformationContainer)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChitChat_ServiceDesc is the grpc.ServiceDesc for ChitChat service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChitChat_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "chat_service.ChitChat",
	HandlerType: (*ChitChatServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Chat",
			Handler:       _ChitChat_Chat_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "grpc/chat_service.proto",
}
