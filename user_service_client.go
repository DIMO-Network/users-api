// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/grpc/users_grpc.pb.go
//
// Generated by this command:
//
//	mockgen -source pkg/grpc/users_grpc.pb.go -destination user_service_client.go -package mock_services
//

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"

	grpc "github.com/DIMO-Network/users-api/pkg/grpc"
	gomock "go.uber.org/mock/gomock"
	grpc0 "google.golang.org/grpc"
)

// MockUserServiceClient is a mock of UserServiceClient interface.
type MockUserServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockUserServiceClientMockRecorder
	isgomock struct{}
}

// MockUserServiceClientMockRecorder is the mock recorder for MockUserServiceClient.
type MockUserServiceClientMockRecorder struct {
	mock *MockUserServiceClient
}

// NewMockUserServiceClient creates a new mock instance.
func NewMockUserServiceClient(ctrl *gomock.Controller) *MockUserServiceClient {
	mock := &MockUserServiceClient{ctrl: ctrl}
	mock.recorder = &MockUserServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserServiceClient) EXPECT() *MockUserServiceClientMockRecorder {
	return m.recorder
}

// GetUser mocks base method.
func (m *MockUserServiceClient) GetUser(ctx context.Context, in *grpc.GetUserRequest, opts ...grpc0.CallOption) (*grpc.User, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetUser", varargs...)
	ret0, _ := ret[0].(*grpc.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserServiceClientMockRecorder) GetUser(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserServiceClient)(nil).GetUser), varargs...)
}

// GetUserByEthAddr mocks base method.
func (m *MockUserServiceClient) GetUserByEthAddr(ctx context.Context, in *grpc.GetUserByEthRequest, opts ...grpc0.CallOption) (*grpc.User, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetUserByEthAddr", varargs...)
	ret0, _ := ret[0].(*grpc.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByEthAddr indicates an expected call of GetUserByEthAddr.
func (mr *MockUserServiceClientMockRecorder) GetUserByEthAddr(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByEthAddr", reflect.TypeOf((*MockUserServiceClient)(nil).GetUserByEthAddr), varargs...)
}

// GetUsersByEthereumAddress mocks base method.
func (m *MockUserServiceClient) GetUsersByEthereumAddress(ctx context.Context, in *grpc.GetUsersByEthereumAddressRequest, opts ...grpc0.CallOption) (*grpc.GetUsersByEthereumAddressResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetUsersByEthereumAddress", varargs...)
	ret0, _ := ret[0].(*grpc.GetUsersByEthereumAddressResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUsersByEthereumAddress indicates an expected call of GetUsersByEthereumAddress.
func (mr *MockUserServiceClientMockRecorder) GetUsersByEthereumAddress(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsersByEthereumAddress", reflect.TypeOf((*MockUserServiceClient)(nil).GetUsersByEthereumAddress), varargs...)
}

// MockUserServiceServer is a mock of UserServiceServer interface.
type MockUserServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUserServiceServerMockRecorder
	isgomock struct{}
}

// MockUserServiceServerMockRecorder is the mock recorder for MockUserServiceServer.
type MockUserServiceServerMockRecorder struct {
	mock *MockUserServiceServer
}

// NewMockUserServiceServer creates a new mock instance.
func NewMockUserServiceServer(ctrl *gomock.Controller) *MockUserServiceServer {
	mock := &MockUserServiceServer{ctrl: ctrl}
	mock.recorder = &MockUserServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserServiceServer) EXPECT() *MockUserServiceServerMockRecorder {
	return m.recorder
}

// GetUser mocks base method.
func (m *MockUserServiceServer) GetUser(arg0 context.Context, arg1 *grpc.GetUserRequest) (*grpc.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", arg0, arg1)
	ret0, _ := ret[0].(*grpc.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserServiceServerMockRecorder) GetUser(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserServiceServer)(nil).GetUser), arg0, arg1)
}

// GetUserByEthAddr mocks base method.
func (m *MockUserServiceServer) GetUserByEthAddr(arg0 context.Context, arg1 *grpc.GetUserByEthRequest) (*grpc.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByEthAddr", arg0, arg1)
	ret0, _ := ret[0].(*grpc.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByEthAddr indicates an expected call of GetUserByEthAddr.
func (mr *MockUserServiceServerMockRecorder) GetUserByEthAddr(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByEthAddr", reflect.TypeOf((*MockUserServiceServer)(nil).GetUserByEthAddr), arg0, arg1)
}

// GetUsersByEthereumAddress mocks base method.
func (m *MockUserServiceServer) GetUsersByEthereumAddress(arg0 context.Context, arg1 *grpc.GetUsersByEthereumAddressRequest) (*grpc.GetUsersByEthereumAddressResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsersByEthereumAddress", arg0, arg1)
	ret0, _ := ret[0].(*grpc.GetUsersByEthereumAddressResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUsersByEthereumAddress indicates an expected call of GetUsersByEthereumAddress.
func (mr *MockUserServiceServerMockRecorder) GetUsersByEthereumAddress(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsersByEthereumAddress", reflect.TypeOf((*MockUserServiceServer)(nil).GetUsersByEthereumAddress), arg0, arg1)
}

// mustEmbedUnimplementedUserServiceServer mocks base method.
func (m *MockUserServiceServer) mustEmbedUnimplementedUserServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedUserServiceServer")
}

// mustEmbedUnimplementedUserServiceServer indicates an expected call of mustEmbedUnimplementedUserServiceServer.
func (mr *MockUserServiceServerMockRecorder) mustEmbedUnimplementedUserServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedUserServiceServer", reflect.TypeOf((*MockUserServiceServer)(nil).mustEmbedUnimplementedUserServiceServer))
}

// MockUnsafeUserServiceServer is a mock of UnsafeUserServiceServer interface.
type MockUnsafeUserServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUnsafeUserServiceServerMockRecorder
	isgomock struct{}
}

// MockUnsafeUserServiceServerMockRecorder is the mock recorder for MockUnsafeUserServiceServer.
type MockUnsafeUserServiceServerMockRecorder struct {
	mock *MockUnsafeUserServiceServer
}

// NewMockUnsafeUserServiceServer creates a new mock instance.
func NewMockUnsafeUserServiceServer(ctrl *gomock.Controller) *MockUnsafeUserServiceServer {
	mock := &MockUnsafeUserServiceServer{ctrl: ctrl}
	mock.recorder = &MockUnsafeUserServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnsafeUserServiceServer) EXPECT() *MockUnsafeUserServiceServerMockRecorder {
	return m.recorder
}

// mustEmbedUnimplementedUserServiceServer mocks base method.
func (m *MockUnsafeUserServiceServer) mustEmbedUnimplementedUserServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedUserServiceServer")
}

// mustEmbedUnimplementedUserServiceServer indicates an expected call of mustEmbedUnimplementedUserServiceServer.
func (mr *MockUnsafeUserServiceServerMockRecorder) mustEmbedUnimplementedUserServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedUserServiceServer", reflect.TypeOf((*MockUnsafeUserServiceServer)(nil).mustEmbedUnimplementedUserServiceServer))
}
