// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: agrpc.proto

package agrpc

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

// AssistAgentClient is the client API for AssistAgent service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AssistAgentClient interface {
	GenRsaKeyPair(ctx context.Context, in *GenRsaKeyPairReq, opts ...grpc.CallOption) (*GenRsaKeyPairResp, error)
	EncryptText(ctx context.Context, in *EncryptReq, opts ...grpc.CallOption) (*EncryptResp, error)
	DecryptText(ctx context.Context, in *DecryptReq, opts ...grpc.CallOption) (*DecryptResp, error)
	CheckKey(ctx context.Context, in *CheckKeyReq, opts ...grpc.CallOption) (*CheckKeyResp, error)
}

type assistAgentClient struct {
	cc grpc.ClientConnInterface
}

func NewAssistAgentClient(cc grpc.ClientConnInterface) AssistAgentClient {
	return &assistAgentClient{cc}
}

func (c *assistAgentClient) GenRsaKeyPair(ctx context.Context, in *GenRsaKeyPairReq, opts ...grpc.CallOption) (*GenRsaKeyPairResp, error) {
	out := new(GenRsaKeyPairResp)
	err := c.cc.Invoke(ctx, "/protos.AssistAgent/GenRsaKeyPair", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *assistAgentClient) EncryptText(ctx context.Context, in *EncryptReq, opts ...grpc.CallOption) (*EncryptResp, error) {
	out := new(EncryptResp)
	err := c.cc.Invoke(ctx, "/protos.AssistAgent/EncryptText", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *assistAgentClient) DecryptText(ctx context.Context, in *DecryptReq, opts ...grpc.CallOption) (*DecryptResp, error) {
	out := new(DecryptResp)
	err := c.cc.Invoke(ctx, "/protos.AssistAgent/DecryptText", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *assistAgentClient) CheckKey(ctx context.Context, in *CheckKeyReq, opts ...grpc.CallOption) (*CheckKeyResp, error) {
	out := new(CheckKeyResp)
	err := c.cc.Invoke(ctx, "/protos.AssistAgent/CheckKey", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AssistAgentServer is the server API for AssistAgent service.
// All implementations must embed UnimplementedAssistAgentServer
// for forward compatibility
type AssistAgentServer interface {
	GenRsaKeyPair(context.Context, *GenRsaKeyPairReq) (*GenRsaKeyPairResp, error)
	EncryptText(context.Context, *EncryptReq) (*EncryptResp, error)
	DecryptText(context.Context, *DecryptReq) (*DecryptResp, error)
	CheckKey(context.Context, *CheckKeyReq) (*CheckKeyResp, error)
	mustEmbedUnimplementedAssistAgentServer()
}

// UnimplementedAssistAgentServer must be embedded to have forward compatible implementations.
type UnimplementedAssistAgentServer struct {
}

func (UnimplementedAssistAgentServer) GenRsaKeyPair(context.Context, *GenRsaKeyPairReq) (*GenRsaKeyPairResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenRsaKeyPair not implemented")
}
func (UnimplementedAssistAgentServer) EncryptText(context.Context, *EncryptReq) (*EncryptResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EncryptText not implemented")
}
func (UnimplementedAssistAgentServer) DecryptText(context.Context, *DecryptReq) (*DecryptResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecryptText not implemented")
}
func (UnimplementedAssistAgentServer) CheckKey(context.Context, *CheckKeyReq) (*CheckKeyResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckKey not implemented")
}
func (UnimplementedAssistAgentServer) mustEmbedUnimplementedAssistAgentServer() {}

// UnsafeAssistAgentServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AssistAgentServer will
// result in compilation errors.
type UnsafeAssistAgentServer interface {
	mustEmbedUnimplementedAssistAgentServer()
}

func RegisterAssistAgentServer(s grpc.ServiceRegistrar, srv AssistAgentServer) {
	s.RegisterService(&AssistAgent_ServiceDesc, srv)
}

func _AssistAgent_GenRsaKeyPair_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GenRsaKeyPairReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AssistAgentServer).GenRsaKeyPair(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.AssistAgent/GenRsaKeyPair",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AssistAgentServer).GenRsaKeyPair(ctx, req.(*GenRsaKeyPairReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _AssistAgent_EncryptText_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EncryptReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AssistAgentServer).EncryptText(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.AssistAgent/EncryptText",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AssistAgentServer).EncryptText(ctx, req.(*EncryptReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _AssistAgent_DecryptText_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DecryptReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AssistAgentServer).DecryptText(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.AssistAgent/DecryptText",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AssistAgentServer).DecryptText(ctx, req.(*DecryptReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _AssistAgent_CheckKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckKeyReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AssistAgentServer).CheckKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.AssistAgent/CheckKey",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AssistAgentServer).CheckKey(ctx, req.(*CheckKeyReq))
	}
	return interceptor(ctx, in, info, handler)
}

// AssistAgent_ServiceDesc is the grpc.ServiceDesc for AssistAgent service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AssistAgent_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "protos.AssistAgent",
	HandlerType: (*AssistAgentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GenRsaKeyPair",
			Handler:    _AssistAgent_GenRsaKeyPair_Handler,
		},
		{
			MethodName: "EncryptText",
			Handler:    _AssistAgent_EncryptText_Handler,
		},
		{
			MethodName: "DecryptText",
			Handler:    _AssistAgent_DecryptText_Handler,
		},
		{
			MethodName: "CheckKey",
			Handler:    _AssistAgent_CheckKey_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "agrpc.proto",
}