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
	ListManifests(ctx context.Context, in *ListManifestsRequest, opts ...grpc.CallOption) (*ListManifestsResponse, error)
	ListManifestFiles(ctx context.Context, in *ListManifestFilesRequest, opts ...grpc.CallOption) (*ListManifestFilesResponse, error)
	RelocateManifestFiles(ctx context.Context, in *RelocateManifestFilesRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	SyncManifest(ctx context.Context, in *SyncManifestRequest, opts ...grpc.CallOption) (*SyncManifestResponse, error)
	ResetManifest(ctx context.Context, in *ResetManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	// Upload Endpoints
	UploadManifest(ctx context.Context, in *UploadManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	CancelUpload(ctx context.Context, in *CancelUploadRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error)
	// Server Endpoints
	Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (Agent_SubscribeClient, error)
	Unsubscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (*SubsrcribeResponse, error)
	// User Endpoints
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*UserResponse, error)
	SwitchProfile(ctx context.Context, in *SwitchProfileRequest, opts ...grpc.CallOption) (*UserResponse, error)
	ReAuthenticate(ctx context.Context, in *ReAuthenticateRequest, opts ...grpc.CallOption) (*UserResponse, error)
	// Datasets Endpoints
	UseDataset(ctx context.Context, in *UseDatasetRequest, opts ...grpc.CallOption) (*UseDatasetResponse, error)
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

func (c *agentClient) ListManifests(ctx context.Context, in *ListManifestsRequest, opts ...grpc.CallOption) (*ListManifestsResponse, error) {
	out := new(ListManifestsResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/ListManifests", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) ListManifestFiles(ctx context.Context, in *ListManifestFilesRequest, opts ...grpc.CallOption) (*ListManifestFilesResponse, error) {
	out := new(ListManifestFilesResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/ListManifestFiles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) RelocateManifestFiles(ctx context.Context, in *RelocateManifestFilesRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/RelocateManifestFiles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) SyncManifest(ctx context.Context, in *SyncManifestRequest, opts ...grpc.CallOption) (*SyncManifestResponse, error) {
	out := new(SyncManifestResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/SyncManifest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) ResetManifest(ctx context.Context, in *ResetManifestRequest, opts ...grpc.CallOption) (*SimpleStatusResponse, error) {
	out := new(SimpleStatusResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/ResetManifest", in, out, opts...)
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

func (c *agentClient) ReAuthenticate(ctx context.Context, in *ReAuthenticateRequest, opts ...grpc.CallOption) (*UserResponse, error) {
	out := new(UserResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/ReAuthenticate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentClient) UseDataset(ctx context.Context, in *UseDatasetRequest, opts ...grpc.CallOption) (*UseDatasetResponse, error) {
	out := new(UseDatasetResponse)
	err := c.cc.Invoke(ctx, "/protos.Agent/UseDataset", in, out, opts...)
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
	ListManifests(context.Context, *ListManifestsRequest) (*ListManifestsResponse, error)
	ListManifestFiles(context.Context, *ListManifestFilesRequest) (*ListManifestFilesResponse, error)
	RelocateManifestFiles(context.Context, *RelocateManifestFilesRequest) (*SimpleStatusResponse, error)
	SyncManifest(context.Context, *SyncManifestRequest) (*SyncManifestResponse, error)
	ResetManifest(context.Context, *ResetManifestRequest) (*SimpleStatusResponse, error)
	// Upload Endpoints
	UploadManifest(context.Context, *UploadManifestRequest) (*SimpleStatusResponse, error)
	CancelUpload(context.Context, *CancelUploadRequest) (*SimpleStatusResponse, error)
	// Server Endpoints
	Subscribe(*SubscribeRequest, Agent_SubscribeServer) error
	Unsubscribe(context.Context, *SubscribeRequest) (*SubsrcribeResponse, error)
	// User Endpoints
	GetUser(context.Context, *GetUserRequest) (*UserResponse, error)
	SwitchProfile(context.Context, *SwitchProfileRequest) (*UserResponse, error)
	ReAuthenticate(context.Context, *ReAuthenticateRequest) (*UserResponse, error)
	// Datasets Endpoints
	UseDataset(context.Context, *UseDatasetRequest) (*UseDatasetResponse, error)
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
func (UnimplementedAgentServer) ListManifests(context.Context, *ListManifestsRequest) (*ListManifestsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManifests not implemented")
}
func (UnimplementedAgentServer) ListManifestFiles(context.Context, *ListManifestFilesRequest) (*ListManifestFilesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManifestFiles not implemented")
}
func (UnimplementedAgentServer) RelocateManifestFiles(context.Context, *RelocateManifestFilesRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RelocateManifestFiles not implemented")
}
func (UnimplementedAgentServer) SyncManifest(context.Context, *SyncManifestRequest) (*SyncManifestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncManifest not implemented")
}
func (UnimplementedAgentServer) ResetManifest(context.Context, *ResetManifestRequest) (*SimpleStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ResetManifest not implemented")
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
func (UnimplementedAgentServer) ReAuthenticate(context.Context, *ReAuthenticateRequest) (*UserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReAuthenticate not implemented")
}
func (UnimplementedAgentServer) UseDataset(context.Context, *UseDatasetRequest) (*UseDatasetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UseDataset not implemented")
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

func _Agent_ListManifests_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManifestsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).ListManifests(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/ListManifests",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).ListManifests(ctx, req.(*ListManifestsRequest))
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

func _Agent_RelocateManifestFiles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RelocateManifestFilesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).RelocateManifestFiles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/RelocateManifestFiles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).RelocateManifestFiles(ctx, req.(*RelocateManifestFilesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_SyncManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SyncManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).SyncManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/SyncManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).SyncManifest(ctx, req.(*SyncManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_ResetManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ResetManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).ResetManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/ResetManifest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).ResetManifest(ctx, req.(*ResetManifestRequest))
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

func _Agent_ReAuthenticate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReAuthenticateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).ReAuthenticate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/ReAuthenticate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).ReAuthenticate(ctx, req.(*ReAuthenticateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Agent_UseDataset_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UseDatasetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentServer).UseDataset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Agent/UseDataset",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentServer).UseDataset(ctx, req.(*UseDatasetRequest))
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
			MethodName: "ListManifests",
			Handler:    _Agent_ListManifests_Handler,
		},
		{
			MethodName: "ListManifestFiles",
			Handler:    _Agent_ListManifestFiles_Handler,
		},
		{
			MethodName: "RelocateManifestFiles",
			Handler:    _Agent_RelocateManifestFiles_Handler,
		},
		{
			MethodName: "SyncManifest",
			Handler:    _Agent_SyncManifest_Handler,
		},
		{
			MethodName: "ResetManifest",
			Handler:    _Agent_ResetManifest_Handler,
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
		{
			MethodName: "ReAuthenticate",
			Handler:    _Agent_ReAuthenticate_Handler,
		},
		{
			MethodName: "UseDataset",
			Handler:    _Agent_UseDataset_Handler,
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
