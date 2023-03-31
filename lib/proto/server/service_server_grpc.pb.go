// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.5
// source: misc/proto/service_server.proto

package server

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

// NXProxyClient is the client API for NXProxy service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type NXProxyClient interface {
	Ver(ctx context.Context, in *Noop, opts ...grpc.CallOption) (*Noop, error)
	Proxy(ctx context.Context, opts ...grpc.CallOption) (NXProxy_ProxyClient, error)
	ReverseProxyListen(ctx context.Context, opts ...grpc.CallOption) (NXProxy_ReverseProxyListenClient, error)
	ReverseProxyWork(ctx context.Context, opts ...grpc.CallOption) (NXProxy_ReverseProxyWorkClient, error)
	Ping(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*StringMsg, error)
	PortScan(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*StringMsg, error)
	Nc(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*StringMsg, error)
	SpeedTest(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*BytesMsg, error)
	KeepAlive(ctx context.Context, in *Noop, opts ...grpc.CallOption) (NXProxy_KeepAliveClient, error)
	GetConfigs(ctx context.Context, in *Noop, opts ...grpc.CallOption) (*Bridges, error)
	StreamConfig(ctx context.Context, in *Noop, opts ...grpc.CallOption) (NXProxy_StreamConfigClient, error)
	Login(ctx context.Context, in *LoginReq, opts ...grpc.CallOption) (*StringMsg, error)
	Logout(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*Noop, error)
}

type nXProxyClient struct {
	cc grpc.ClientConnInterface
}

func NewNXProxyClient(cc grpc.ClientConnInterface) NXProxyClient {
	return &nXProxyClient{cc}
}

func (c *nXProxyClient) Ver(ctx context.Context, in *Noop, opts ...grpc.CallOption) (*Noop, error) {
	out := new(Noop)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/Ver", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) Proxy(ctx context.Context, opts ...grpc.CallOption) (NXProxy_ProxyClient, error) {
	stream, err := c.cc.NewStream(ctx, &NXProxy_ServiceDesc.Streams[0], "/proxy.NXProxy/Proxy", opts...)
	if err != nil {
		return nil, err
	}
	x := &nXProxyProxyClient{stream}
	return x, nil
}

type NXProxy_ProxyClient interface {
	Send(*ConnOut) error
	Recv() (*ConnIn, error)
	grpc.ClientStream
}

type nXProxyProxyClient struct {
	grpc.ClientStream
}

func (x *nXProxyProxyClient) Send(m *ConnOut) error {
	return x.ClientStream.SendMsg(m)
}

func (x *nXProxyProxyClient) Recv() (*ConnIn, error) {
	m := new(ConnIn)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nXProxyClient) ReverseProxyListen(ctx context.Context, opts ...grpc.CallOption) (NXProxy_ReverseProxyListenClient, error) {
	stream, err := c.cc.NewStream(ctx, &NXProxy_ServiceDesc.Streams[1], "/proxy.NXProxy/ReverseProxyListen", opts...)
	if err != nil {
		return nil, err
	}
	x := &nXProxyReverseProxyListenClient{stream}
	return x, nil
}

type NXProxy_ReverseProxyListenClient interface {
	Send(*ConnOut) error
	Recv() (*RevProxyRequest, error)
	grpc.ClientStream
}

type nXProxyReverseProxyListenClient struct {
	grpc.ClientStream
}

func (x *nXProxyReverseProxyListenClient) Send(m *ConnOut) error {
	return x.ClientStream.SendMsg(m)
}

func (x *nXProxyReverseProxyListenClient) Recv() (*RevProxyRequest, error) {
	m := new(RevProxyRequest)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nXProxyClient) ReverseProxyWork(ctx context.Context, opts ...grpc.CallOption) (NXProxy_ReverseProxyWorkClient, error) {
	stream, err := c.cc.NewStream(ctx, &NXProxy_ServiceDesc.Streams[2], "/proxy.NXProxy/ReverseProxyWork", opts...)
	if err != nil {
		return nil, err
	}
	x := &nXProxyReverseProxyWorkClient{stream}
	return x, nil
}

type NXProxy_ReverseProxyWorkClient interface {
	Send(*RevProxyConnIn) error
	Recv() (*RevProxyConnOut, error)
	grpc.ClientStream
}

type nXProxyReverseProxyWorkClient struct {
	grpc.ClientStream
}

func (x *nXProxyReverseProxyWorkClient) Send(m *RevProxyConnIn) error {
	return x.ClientStream.SendMsg(m)
}

func (x *nXProxyReverseProxyWorkClient) Recv() (*RevProxyConnOut, error) {
	m := new(RevProxyConnOut)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nXProxyClient) Ping(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*StringMsg, error) {
	out := new(StringMsg)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/Ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) PortScan(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*StringMsg, error) {
	out := new(StringMsg)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/PortScan", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) Nc(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*StringMsg, error) {
	out := new(StringMsg)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/Nc", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) SpeedTest(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*BytesMsg, error) {
	out := new(BytesMsg)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/SpeedTest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) KeepAlive(ctx context.Context, in *Noop, opts ...grpc.CallOption) (NXProxy_KeepAliveClient, error) {
	stream, err := c.cc.NewStream(ctx, &NXProxy_ServiceDesc.Streams[3], "/proxy.NXProxy/KeepAlive", opts...)
	if err != nil {
		return nil, err
	}
	x := &nXProxyKeepAliveClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type NXProxy_KeepAliveClient interface {
	Recv() (*StringMsg, error)
	grpc.ClientStream
}

type nXProxyKeepAliveClient struct {
	grpc.ClientStream
}

func (x *nXProxyKeepAliveClient) Recv() (*StringMsg, error) {
	m := new(StringMsg)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nXProxyClient) GetConfigs(ctx context.Context, in *Noop, opts ...grpc.CallOption) (*Bridges, error) {
	out := new(Bridges)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/GetConfigs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) StreamConfig(ctx context.Context, in *Noop, opts ...grpc.CallOption) (NXProxy_StreamConfigClient, error) {
	stream, err := c.cc.NewStream(ctx, &NXProxy_ServiceDesc.Streams[4], "/proxy.NXProxy/StreamConfig", opts...)
	if err != nil {
		return nil, err
	}
	x := &nXProxyStreamConfigClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type NXProxy_StreamConfigClient interface {
	Recv() (*Bridges, error)
	grpc.ClientStream
}

type nXProxyStreamConfigClient struct {
	grpc.ClientStream
}

func (x *nXProxyStreamConfigClient) Recv() (*Bridges, error) {
	m := new(Bridges)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nXProxyClient) Login(ctx context.Context, in *LoginReq, opts ...grpc.CallOption) (*StringMsg, error) {
	out := new(StringMsg)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/Login", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nXProxyClient) Logout(ctx context.Context, in *StringMsg, opts ...grpc.CallOption) (*Noop, error) {
	out := new(Noop)
	err := c.cc.Invoke(ctx, "/proxy.NXProxy/Logout", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NXProxyServer is the server API for NXProxy service.
// All implementations must embed UnimplementedNXProxyServer
// for forward compatibility
type NXProxyServer interface {
	Ver(context.Context, *Noop) (*Noop, error)
	Proxy(NXProxy_ProxyServer) error
	ReverseProxyListen(NXProxy_ReverseProxyListenServer) error
	ReverseProxyWork(NXProxy_ReverseProxyWorkServer) error
	Ping(context.Context, *StringMsg) (*StringMsg, error)
	PortScan(context.Context, *StringMsg) (*StringMsg, error)
	Nc(context.Context, *StringMsg) (*StringMsg, error)
	SpeedTest(context.Context, *StringMsg) (*BytesMsg, error)
	KeepAlive(*Noop, NXProxy_KeepAliveServer) error
	GetConfigs(context.Context, *Noop) (*Bridges, error)
	StreamConfig(*Noop, NXProxy_StreamConfigServer) error
	Login(context.Context, *LoginReq) (*StringMsg, error)
	Logout(context.Context, *StringMsg) (*Noop, error)
	mustEmbedUnimplementedNXProxyServer()
}

// UnimplementedNXProxyServer must be embedded to have forward compatible implementations.
type UnimplementedNXProxyServer struct {
}

func (UnimplementedNXProxyServer) Ver(context.Context, *Noop) (*Noop, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ver not implemented")
}
func (UnimplementedNXProxyServer) Proxy(NXProxy_ProxyServer) error {
	return status.Errorf(codes.Unimplemented, "method Proxy not implemented")
}
func (UnimplementedNXProxyServer) ReverseProxyListen(NXProxy_ReverseProxyListenServer) error {
	return status.Errorf(codes.Unimplemented, "method ReverseProxyListen not implemented")
}
func (UnimplementedNXProxyServer) ReverseProxyWork(NXProxy_ReverseProxyWorkServer) error {
	return status.Errorf(codes.Unimplemented, "method ReverseProxyWork not implemented")
}
func (UnimplementedNXProxyServer) Ping(context.Context, *StringMsg) (*StringMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedNXProxyServer) PortScan(context.Context, *StringMsg) (*StringMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PortScan not implemented")
}
func (UnimplementedNXProxyServer) Nc(context.Context, *StringMsg) (*StringMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Nc not implemented")
}
func (UnimplementedNXProxyServer) SpeedTest(context.Context, *StringMsg) (*BytesMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SpeedTest not implemented")
}
func (UnimplementedNXProxyServer) KeepAlive(*Noop, NXProxy_KeepAliveServer) error {
	return status.Errorf(codes.Unimplemented, "method KeepAlive not implemented")
}
func (UnimplementedNXProxyServer) GetConfigs(context.Context, *Noop) (*Bridges, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigs not implemented")
}
func (UnimplementedNXProxyServer) StreamConfig(*Noop, NXProxy_StreamConfigServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamConfig not implemented")
}
func (UnimplementedNXProxyServer) Login(context.Context, *LoginReq) (*StringMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedNXProxyServer) Logout(context.Context, *StringMsg) (*Noop, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Logout not implemented")
}
func (UnimplementedNXProxyServer) mustEmbedUnimplementedNXProxyServer() {}

// UnsafeNXProxyServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NXProxyServer will
// result in compilation errors.
type UnsafeNXProxyServer interface {
	mustEmbedUnimplementedNXProxyServer()
}

func RegisterNXProxyServer(s grpc.ServiceRegistrar, srv NXProxyServer) {
	s.RegisterService(&NXProxy_ServiceDesc, srv)
}

func _NXProxy_Ver_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Noop)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).Ver(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/Ver",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).Ver(ctx, req.(*Noop))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_Proxy_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NXProxyServer).Proxy(&nXProxyProxyServer{stream})
}

type NXProxy_ProxyServer interface {
	Send(*ConnIn) error
	Recv() (*ConnOut, error)
	grpc.ServerStream
}

type nXProxyProxyServer struct {
	grpc.ServerStream
}

func (x *nXProxyProxyServer) Send(m *ConnIn) error {
	return x.ServerStream.SendMsg(m)
}

func (x *nXProxyProxyServer) Recv() (*ConnOut, error) {
	m := new(ConnOut)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _NXProxy_ReverseProxyListen_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NXProxyServer).ReverseProxyListen(&nXProxyReverseProxyListenServer{stream})
}

type NXProxy_ReverseProxyListenServer interface {
	Send(*RevProxyRequest) error
	Recv() (*ConnOut, error)
	grpc.ServerStream
}

type nXProxyReverseProxyListenServer struct {
	grpc.ServerStream
}

func (x *nXProxyReverseProxyListenServer) Send(m *RevProxyRequest) error {
	return x.ServerStream.SendMsg(m)
}

func (x *nXProxyReverseProxyListenServer) Recv() (*ConnOut, error) {
	m := new(ConnOut)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _NXProxy_ReverseProxyWork_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NXProxyServer).ReverseProxyWork(&nXProxyReverseProxyWorkServer{stream})
}

type NXProxy_ReverseProxyWorkServer interface {
	Send(*RevProxyConnOut) error
	Recv() (*RevProxyConnIn, error)
	grpc.ServerStream
}

type nXProxyReverseProxyWorkServer struct {
	grpc.ServerStream
}

func (x *nXProxyReverseProxyWorkServer) Send(m *RevProxyConnOut) error {
	return x.ServerStream.SendMsg(m)
}

func (x *nXProxyReverseProxyWorkServer) Recv() (*RevProxyConnIn, error) {
	m := new(RevProxyConnIn)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _NXProxy_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StringMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).Ping(ctx, req.(*StringMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_PortScan_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StringMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).PortScan(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/PortScan",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).PortScan(ctx, req.(*StringMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_Nc_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StringMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).Nc(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/Nc",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).Nc(ctx, req.(*StringMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_SpeedTest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StringMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).SpeedTest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/SpeedTest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).SpeedTest(ctx, req.(*StringMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_KeepAlive_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Noop)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(NXProxyServer).KeepAlive(m, &nXProxyKeepAliveServer{stream})
}

type NXProxy_KeepAliveServer interface {
	Send(*StringMsg) error
	grpc.ServerStream
}

type nXProxyKeepAliveServer struct {
	grpc.ServerStream
}

func (x *nXProxyKeepAliveServer) Send(m *StringMsg) error {
	return x.ServerStream.SendMsg(m)
}

func _NXProxy_GetConfigs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Noop)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).GetConfigs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/GetConfigs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).GetConfigs(ctx, req.(*Noop))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_StreamConfig_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Noop)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(NXProxyServer).StreamConfig(m, &nXProxyStreamConfigServer{stream})
}

type NXProxy_StreamConfigServer interface {
	Send(*Bridges) error
	grpc.ServerStream
}

type nXProxyStreamConfigServer struct {
	grpc.ServerStream
}

func (x *nXProxyStreamConfigServer) Send(m *Bridges) error {
	return x.ServerStream.SendMsg(m)
}

func _NXProxy_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoginReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/Login",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).Login(ctx, req.(*LoginReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _NXProxy_Logout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StringMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NXProxyServer).Logout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proxy.NXProxy/Logout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NXProxyServer).Logout(ctx, req.(*StringMsg))
	}
	return interceptor(ctx, in, info, handler)
}

// NXProxy_ServiceDesc is the grpc.ServiceDesc for NXProxy service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var NXProxy_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proxy.NXProxy",
	HandlerType: (*NXProxyServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ver",
			Handler:    _NXProxy_Ver_Handler,
		},
		{
			MethodName: "Ping",
			Handler:    _NXProxy_Ping_Handler,
		},
		{
			MethodName: "PortScan",
			Handler:    _NXProxy_PortScan_Handler,
		},
		{
			MethodName: "Nc",
			Handler:    _NXProxy_Nc_Handler,
		},
		{
			MethodName: "SpeedTest",
			Handler:    _NXProxy_SpeedTest_Handler,
		},
		{
			MethodName: "GetConfigs",
			Handler:    _NXProxy_GetConfigs_Handler,
		},
		{
			MethodName: "Login",
			Handler:    _NXProxy_Login_Handler,
		},
		{
			MethodName: "Logout",
			Handler:    _NXProxy_Logout_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Proxy",
			Handler:       _NXProxy_Proxy_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "ReverseProxyListen",
			Handler:       _NXProxy_ReverseProxyListen_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "ReverseProxyWork",
			Handler:       _NXProxy_ReverseProxyWork_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "KeepAlive",
			Handler:       _NXProxy_KeepAlive_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "StreamConfig",
			Handler:       _NXProxy_StreamConfig_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "misc/proto/service_server.proto",
}
