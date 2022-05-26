// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.4
// source: protos/agent.proto

package protos

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

// AgentClient is the client API for Agent service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AgentClient interface {
	// Manifest Endpoints
	CreateManifest(ctx context.Context, in *CreateManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	AddToManifest(ctx context.Context, in *AddToManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	RemoveFromManifest(ctx context.Context, in *RemoveFromManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	DeleteManifest(ctx context.Context, in *DeleteManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	ManifestStatus(ctx context.Context, in *ManifestStatusRequest, opts ...grpc.CallOption) (*ManifestStatusResponse, error)
	ListManifestFiles(ctx context.Context, in *ListManifestFilesRequest, opts ...grpc.CallOption) (*ListFilesResponse, error)
	// Upload Endpoints
	UploadManifest(ctx context.Context, in *UploadManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	CancelUpload(ctx context.Context, in *CancelUploadRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	// Server Endpoints
	Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (Agent_SubscribeClient, error)
	Unsubscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (*SubsrcribeResponse, error)
	// User Endpoints
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*UserResponse, error)
	SwitchProfile(ctx context.Context, in *SwitchProfileRequest, opts ...grpc.CallOption) (*UserResponse, error)
}

type agentClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentClient(cc grpc.ClientConnInterface) AgentClient {
	return &agentClient{cc}
}

func (c *agentClient) CreateManifest(ctx context.Context, in *CreateManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/CreateManifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) AddToManifest(ctx context.Context, in *AddToManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/AddToManifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) RemoveFromManifest(ctx context.Context, in *RemoveFromManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/RemoveFromManifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) DeleteManifest(ctx context.Context, in *DeleteManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/DeleteManifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) ManifestStatus(ctx context.Context, in *ManifestStatusRequest, opts ...grpc.CallOption) (*ManifestStatusResponse, error) {
	out := new(ManifestStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/ManifestStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) ListManifestFiles(ctx context.Context, in *ListManifestFilesRequest, opts ...grpc.CallOption) (*ListFilesResponse, error) {
	out := new(ListFilesResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/ListManifestFiles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) UploadManifest(ctx context.Context, in *UploadManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/UploadManifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) CancelUpload(ctx context.Context, in *CancelUploadRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/CancelUpload", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (Agent_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &Agent_ServiceDesc.Streams[0], "/protos.Agent/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Agent_SubscribeClient interface {
	Recv() (*SubsrcribeResponse, error)
	grpc.ClientStream
}

type agentSubscribeClient struct {
	grpc.ClientStream
}

func (x *agentSubscribeClient) Recv() (*SubsrcribeResponse, error) {
	m := new(SubsrcribeResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *agentClient) Unsubscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (*SubsrcribeResponse, error) {
	out := new(SubsrcribeResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/Unsubscribe", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*UserResponse, error) {
	out := new(UserResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/GetUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) SwitchProfile(ctx context.Context, in *SwitchProfileRequest, opts ...grpc.CallOption) (*UserResponse, error) {
	out := new(UserResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/SwitchProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AgentServer is the server API for Agent service.
// All implementations must embed UnimplementedAgentServer
// for forward compatibility
type AgentServer interface {
	// Manifest Endpoints
	CreateManifest(context.Context, *CreateManifestRequest) (*SimpleStatusResponse, error)
	AddToManifest(context.Context, *AddToManifestRequest) (*SimpleStatusResponse, error)
	RemoveFromManifest(context.Context, *RemoveFromManifestRequest) (*SimpleStatusResponse, error)
	DeleteManifest(context.Context, *DeleteManifestRequest) (*SimpleStatusResponse, error)
	ManifestStatus(context.Context, *ManifestStatusRequest) (*ManifestStatusResponse, error)
	ListManifestFiles(context.Context, *ListManifestFilesRequest) (*ListFilesResponse, error)
	// Upload Endpoints
	UploadManifest(context.Context, *UploadManifestRequest) (*SimpleStatusResponse, error)
	CancelUpload(context.Context, *CancelUploadRequest) (*SimpleStatusResponse, error)
	// Server Endpoints
	Subscribe(*SubscribeRequest, Agent_SubscribeServer) error
	Unsubscribe(context.Context, *SubscribeRequest) (*SubsrcribeResponse, error)
	// User Endpoints
	GetUser(context.Context, *GetUserRequest) (*UserResponse, error)
	SwitchProfile(context.Context, *SwitchProfileRequest) (*UserResponse, error)
	mustEmbedUnimplementedAgentServer()
}

// UnimplementedAgentServer must be embedded to have forward compatible implementations.
type UnimplementedAgentServer struct {
}

func (UnimplementedAgentServer) CreateManifest(context.Context, *CreateManifestRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateManifest not implemented")
}
func (UnimplementedAgentServer) AddToManifest(context.Context, *AddToManifestRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddToManifest not implemented")
}
func (UnimplementedAgentServer) RemoveFromManifest(context.Context, *RemoveFromManifestRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveFromManifest not implemented")
}
func (UnimplementedAgentServer) DeleteManifest(context.Context, *DeleteManifestRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteManifest not implemented")
}
func (UnimplementedAgentServer) ManifestStatus(context.Context, *ManifestStatusRequest) (*ManifestStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ManifestStatus not implemented")
}
func (UnimplementedAgentServer) ListManifestFiles(context.Context, *ListManifestFilesRequest) (*ListFilesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManifestFiles not implemented")
}
func (UnimplementedAgentServer) UploadManifest(context.Context, *UploadManifestRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UploadManifest not implemented")
}
func (UnimplementedAgentServer) CancelUpload(context.Context, *CancelUploadRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelUpload not implemented")
}
func (UnimplementedAgentServer) Subscribe(*SubscribeRequest, Agent_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (UnimplementedAgentServer) Unsubscribe(context.Context, *SubscribeRequest) (*SubsrcribeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unsubscribe not implemented")
}
func (UnimplementedAgentServer) GetUser(context.Context, *GetUserRequest) (*UserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}
func (UnimplementedAgentServer) SwitchProfile(context.Context, *SwitchProfileRequest) (*UserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SwitchProfile not implemented")
}
func (UnimplementedAgentServer) mustEmbedUnimplementedAgentServer() {}

// UnsafeAgentServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AgentServer will
// result in compilation errors.
type UnsafeAgentServer interface {
	mustEmbedUnimplementedAgentServer()
}

func RegisterAgentServer(s grpc.ServiceRegistrar, srv AgentServer) {
	s.RegisterService(&Agent_ServiceDesc, srv)
}

func _Agent_CreateManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).CreateManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/CreateManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).CreateManifest(ctx, req.(*CreateManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_AddToManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddToManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).AddToManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/AddToManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).AddToManifest(ctx, req.(*AddToManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_RemoveFromManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveFromManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).RemoveFromManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/RemoveFromManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).RemoveFromManifest(ctx, req.(*RemoveFromManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_DeleteManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).DeleteManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/DeleteManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).DeleteManifest(ctx, req.(*DeleteManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_ManifestStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ManifestStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).ManifestStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/ManifestStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).ManifestStatus(ctx, req.(*ManifestStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_ListManifestFiles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManifestFilesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).ListManifestFiles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/ListManifestFiles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).ListManifestFiles(ctx, req.(*ListManifestFilesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_UploadManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UploadManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).UploadManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/UploadManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).UploadManifest(ctx, req.(*UploadManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_CancelUpload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelUploadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).CancelUpload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/CancelUpload",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).CancelUpload(ctx, req.(*CancelUploadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentServer).Subscribe(m, &agentSubscribeServer{stream})
}

type Agent_SubscribeServer interface {
	Send(*SubsrcribeResponse) error
	grpc.ServerStream
}

type agentSubscribeServer struct {
	grpc.ServerStream
}

func (x *agentSubscribeServer) Send(m *SubsrcribeResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _Agent_Unsubscribe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubscribeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).Unsubscribe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/Unsubscribe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).Unsubscribe(ctx, req.(*SubscribeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_GetUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).GetUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/GetUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).GetUser(ctx, req.(*GetUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_SwitchProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SwitchProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).SwitchProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/SwitchProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).SwitchProfile(ctx, req.(*SwitchProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Agent_ServiceDesc is the grpc.ServiceDesc for Agent service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Agent_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "protos.Agent",
	HandlerType: (*AgentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateManifest",
			Handler:    _Agent_CreateManifest_Handler,
		},
		{
			MethodName: "AddToManifest",
			Handler:    _Agent_AddToManifest_Handler,
		},
		{
			MethodName: "RemoveFromManifest",
			Handler:    _Agent_RemoveFromManifest_Handler,
		},
		{
			MethodName: "DeleteManifest",
			Handler:    _Agent_DeleteManifest_Handler,
		},
		{
			MethodName: "ManifestStatus",
			Handler:    _Agent_ManifestStatus_Handler,
		},
		{
			MethodName: "ListManifestFiles",
			Handler:    _Agent_ListManifestFiles_Handler,
		},
		{
			MethodName: "UploadManifest",
			Handler:    _Agent_UploadManifest_Handler,
		},
		{
			MethodName: "CancelUpload",
			Handler:    _Agent_CancelUpload_Handler,
		},
		{
			MethodName: "Unsubscribe",
			Handler:    _Agent_Unsubscribe_Handler,
		},
		{
			MethodName: "GetUser",
			Handler:    _Agent_GetUser_Handler,
		},
		{
			MethodName: "SwitchProfile",
			Handler:    _Agent_SwitchProfile_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _Agent_Subscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "protos/agent.proto",
}
