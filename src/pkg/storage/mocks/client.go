// Code generated by MockGen. DO NOT EDIT.
// Source: interfaces.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	storage "questspace/pkg/storage"
)

// MockQuestSpaceStorage is a mock of QuestSpaceStorage interface.
type MockQuestSpaceStorage struct {
	ctrl     *gomock.Controller
	recorder *MockQuestSpaceStorageMockRecorder
}

// MockQuestSpaceStorageMockRecorder is the mock recorder for MockQuestSpaceStorage.
type MockQuestSpaceStorageMockRecorder struct {
	mock *MockQuestSpaceStorage
}

// NewMockQuestSpaceStorage creates a new mock instance.
func NewMockQuestSpaceStorage(ctrl *gomock.Controller) *MockQuestSpaceStorage {
	mock := &MockQuestSpaceStorage{ctrl: ctrl}
	mock.recorder = &MockQuestSpaceStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQuestSpaceStorage) EXPECT() *MockQuestSpaceStorageMockRecorder {
	return m.recorder
}

// CreateQuest mocks base method.
func (m *MockQuestSpaceStorage) CreateQuest(ctx context.Context, req *storage.CreateQuestRequest) (*storage.Quest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateQuest", ctx, req)
	ret0, _ := ret[0].(*storage.Quest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateQuest indicates an expected call of CreateQuest.
func (mr *MockQuestSpaceStorageMockRecorder) CreateQuest(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateQuest", reflect.TypeOf((*MockQuestSpaceStorage)(nil).CreateQuest), ctx, req)
}

// CreateUser mocks base method.
func (m *MockQuestSpaceStorage) CreateUser(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, req)
	ret0, _ := ret[0].(*storage.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockQuestSpaceStorageMockRecorder) CreateUser(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockQuestSpaceStorage)(nil).CreateUser), ctx, req)
}

// GetQuest mocks base method.
func (m *MockQuestSpaceStorage) GetQuest(ctx context.Context, req *storage.GetQuestRequest) (*storage.Quest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetQuest", ctx, req)
	ret0, _ := ret[0].(*storage.Quest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetQuest indicates an expected call of GetQuest.
func (mr *MockQuestSpaceStorageMockRecorder) GetQuest(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetQuest", reflect.TypeOf((*MockQuestSpaceStorage)(nil).GetQuest), ctx, req)
}

// GetUser mocks base method.
func (m *MockQuestSpaceStorage) GetUser(ctx context.Context, req *storage.GetUserRequest) (*storage.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, req)
	ret0, _ := ret[0].(*storage.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockQuestSpaceStorageMockRecorder) GetUser(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockQuestSpaceStorage)(nil).GetUser), ctx, req)
}

// GetUserPasswordHash mocks base method.
func (m *MockQuestSpaceStorage) GetUserPasswordHash(ctx context.Context, req *storage.GetUserRequest) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserPasswordHash", ctx, req)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserPasswordHash indicates an expected call of GetUserPasswordHash.
func (mr *MockQuestSpaceStorageMockRecorder) GetUserPasswordHash(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserPasswordHash", reflect.TypeOf((*MockQuestSpaceStorage)(nil).GetUserPasswordHash), ctx, req)
}

// UpdateQuest mocks base method.
func (m *MockQuestSpaceStorage) UpdateQuest(ctx context.Context, req *storage.UpdateQuestRequest) (*storage.Quest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateQuest", ctx, req)
	ret0, _ := ret[0].(*storage.Quest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateQuest indicates an expected call of UpdateQuest.
func (mr *MockQuestSpaceStorageMockRecorder) UpdateQuest(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateQuest", reflect.TypeOf((*MockQuestSpaceStorage)(nil).UpdateQuest), ctx, req)
}

// UpdateUser mocks base method.
func (m *MockQuestSpaceStorage) UpdateUser(ctx context.Context, req *storage.UpdateUserRequest) (*storage.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, req)
	ret0, _ := ret[0].(*storage.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockQuestSpaceStorageMockRecorder) UpdateUser(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockQuestSpaceStorage)(nil).UpdateUser), ctx, req)
}

// MockUserStorage is a mock of UserStorage interface.
type MockUserStorage struct {
	ctrl     *gomock.Controller
	recorder *MockUserStorageMockRecorder
}

// MockUserStorageMockRecorder is the mock recorder for MockUserStorage.
type MockUserStorageMockRecorder struct {
	mock *MockUserStorage
}

// NewMockUserStorage creates a new mock instance.
func NewMockUserStorage(ctrl *gomock.Controller) *MockUserStorage {
	mock := &MockUserStorage{ctrl: ctrl}
	mock.recorder = &MockUserStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserStorage) EXPECT() *MockUserStorageMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockUserStorage) CreateUser(ctx context.Context, req *storage.CreateUserRequest) (*storage.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, req)
	ret0, _ := ret[0].(*storage.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockUserStorageMockRecorder) CreateUser(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockUserStorage)(nil).CreateUser), ctx, req)
}

// GetUser mocks base method.
func (m *MockUserStorage) GetUser(ctx context.Context, req *storage.GetUserRequest) (*storage.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, req)
	ret0, _ := ret[0].(*storage.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserStorageMockRecorder) GetUser(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserStorage)(nil).GetUser), ctx, req)
}

// GetUserPasswordHash mocks base method.
func (m *MockUserStorage) GetUserPasswordHash(ctx context.Context, req *storage.GetUserRequest) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserPasswordHash", ctx, req)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserPasswordHash indicates an expected call of GetUserPasswordHash.
func (mr *MockUserStorageMockRecorder) GetUserPasswordHash(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserPasswordHash", reflect.TypeOf((*MockUserStorage)(nil).GetUserPasswordHash), ctx, req)
}

// UpdateUser mocks base method.
func (m *MockUserStorage) UpdateUser(ctx context.Context, req *storage.UpdateUserRequest) (*storage.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, req)
	ret0, _ := ret[0].(*storage.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserStorageMockRecorder) UpdateUser(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserStorage)(nil).UpdateUser), ctx, req)
}

// MockQuestStorage is a mock of QuestStorage interface.
type MockQuestStorage struct {
	ctrl     *gomock.Controller
	recorder *MockQuestStorageMockRecorder
}

// MockQuestStorageMockRecorder is the mock recorder for MockQuestStorage.
type MockQuestStorageMockRecorder struct {
	mock *MockQuestStorage
}

// NewMockQuestStorage creates a new mock instance.
func NewMockQuestStorage(ctrl *gomock.Controller) *MockQuestStorage {
	mock := &MockQuestStorage{ctrl: ctrl}
	mock.recorder = &MockQuestStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQuestStorage) EXPECT() *MockQuestStorageMockRecorder {
	return m.recorder
}

// CreateQuest mocks base method.
func (m *MockQuestStorage) CreateQuest(ctx context.Context, req *storage.CreateQuestRequest) (*storage.Quest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateQuest", ctx, req)
	ret0, _ := ret[0].(*storage.Quest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateQuest indicates an expected call of CreateQuest.
func (mr *MockQuestStorageMockRecorder) CreateQuest(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateQuest", reflect.TypeOf((*MockQuestStorage)(nil).CreateQuest), ctx, req)
}

// GetQuest mocks base method.
func (m *MockQuestStorage) GetQuest(ctx context.Context, req *storage.GetQuestRequest) (*storage.Quest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetQuest", ctx, req)
	ret0, _ := ret[0].(*storage.Quest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetQuest indicates an expected call of GetQuest.
func (mr *MockQuestStorageMockRecorder) GetQuest(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetQuest", reflect.TypeOf((*MockQuestStorage)(nil).GetQuest), ctx, req)
}

// UpdateQuest mocks base method.
func (m *MockQuestStorage) UpdateQuest(ctx context.Context, req *storage.UpdateQuestRequest) (*storage.Quest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateQuest", ctx, req)
	ret0, _ := ret[0].(*storage.Quest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateQuest indicates an expected call of UpdateQuest.
func (mr *MockQuestStorageMockRecorder) UpdateQuest(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateQuest", reflect.TypeOf((*MockQuestStorage)(nil).UpdateQuest), ctx, req)
}
