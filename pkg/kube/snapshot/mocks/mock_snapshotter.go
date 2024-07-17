// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/kube/snapshot/snapshot.go

// Package mock_snapshot is a generated GoMock package.
package mock_snapshot

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"

	snapshot "github.com/kanisterio/kanister/pkg/kube/snapshot"
)

// MockSnapshotter is a mock of Snapshotter interface.
type MockSnapshotter struct {
	ctrl     *gomock.Controller
	recorder *MockSnapshotterMockRecorder
}

// MockSnapshotterMockRecorder is the mock recorder for MockSnapshotter.
type MockSnapshotterMockRecorder struct {
	mock *MockSnapshotter
}

// NewMockSnapshotter creates a new mock instance.
func NewMockSnapshotter(ctrl *gomock.Controller) *MockSnapshotter {
	mock := &MockSnapshotter{ctrl: ctrl}
	mock.recorder = &MockSnapshotterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSnapshotter) EXPECT() *MockSnapshotterMockRecorder {
	return m.recorder
}

// Clone mocks base method.
func (m *MockSnapshotter) Clone(ctx context.Context, name, namespace, cloneName, cloneNamespace string, waitForReady bool, labels map[string]string, annotations map[string]interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Clone", ctx, name, namespace, cloneName, cloneNamespace, waitForReady, labels, annotations)
	ret0, _ := ret[0].(error)
	return ret0
}

// Clone indicates an expected call of Clone.
func (mr *MockSnapshotterMockRecorder) Clone(ctx, name, namespace, cloneName, cloneNamespace, waitForReady, labels, annotations interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Clone", reflect.TypeOf((*MockSnapshotter)(nil).Clone), ctx, name, namespace, cloneName, cloneNamespace, waitForReady, labels, annotations)
}

// CloneVolumeSnapshotClass mocks base method.
func (m *MockSnapshotter) CloneVolumeSnapshotClass(ctx context.Context, sourceClassName, targetClassName, newDeletionPolicy string, excludeAnnotations []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloneVolumeSnapshotClass", ctx, sourceClassName, targetClassName, newDeletionPolicy, excludeAnnotations)
	ret0, _ := ret[0].(error)
	return ret0
}

// CloneVolumeSnapshotClass indicates an expected call of CloneVolumeSnapshotClass.
func (mr *MockSnapshotterMockRecorder) CloneVolumeSnapshotClass(ctx, sourceClassName, targetClassName, newDeletionPolicy, excludeAnnotations interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloneVolumeSnapshotClass", reflect.TypeOf((*MockSnapshotter)(nil).CloneVolumeSnapshotClass), ctx, sourceClassName, targetClassName, newDeletionPolicy, excludeAnnotations)
}

// Create mocks base method.
func (m *MockSnapshotter) Create(ctx context.Context, name, namespace, pvcName string, snapshotClass *string, waitForReady bool, labels, annotations map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, name, namespace, pvcName, snapshotClass, waitForReady, labels, annotations)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockSnapshotterMockRecorder) Create(ctx, name, namespace, pvcName, snapshotClass, waitForReady, labels, annotations interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockSnapshotter)(nil).Create), ctx, name, namespace, pvcName, snapshotClass, waitForReady, labels, annotations)
}

// CreateContentFromSource mocks base method.
func (m *MockSnapshotter) CreateContentFromSource(ctx context.Context, source *snapshot.Source, contentName, snapshotName, namespace, deletionPolicy string, annotations map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateContentFromSource", ctx, source, contentName, snapshotName, namespace, deletionPolicy, annotations)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateContentFromSource indicates an expected call of CreateContentFromSource.
func (mr *MockSnapshotterMockRecorder) CreateContentFromSource(ctx, source, contentName, snapshotName, namespace, deletionPolicy, annotations interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateContentFromSource", reflect.TypeOf((*MockSnapshotter)(nil).CreateContentFromSource), ctx, source, contentName, snapshotName, namespace, deletionPolicy, annotations)
}

// CreateFromSource mocks base method.
func (m *MockSnapshotter) CreateFromSource(ctx context.Context, source *snapshot.Source, snapshotName, namespace string, waitForReady bool, labels map[string]string, annotations map[string]interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFromSource", ctx, source, snapshotName, namespace, waitForReady, labels, annotations)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateFromSource indicates an expected call of CreateFromSource.
func (mr *MockSnapshotterMockRecorder) CreateFromSource(ctx, source, snapshotName, namespace, waitForReady, labels, annotations interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFromSource", reflect.TypeOf((*MockSnapshotter)(nil).CreateFromSource), ctx, source, snapshotName, namespace, waitForReady, labels, annotations)
}

// Delete mocks base method.
func (m *MockSnapshotter) Delete(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, name, namespace)
	ret0, _ := ret[0].(*v1.VolumeSnapshot)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockSnapshotterMockRecorder) Delete(ctx, name, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockSnapshotter)(nil).Delete), ctx, name, namespace)
}

// DeleteContent mocks base method.
func (m *MockSnapshotter) DeleteContent(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteContent", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteContent indicates an expected call of DeleteContent.
func (mr *MockSnapshotterMockRecorder) DeleteContent(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteContent", reflect.TypeOf((*MockSnapshotter)(nil).DeleteContent), ctx, name)
}

// Get mocks base method.
func (m *MockSnapshotter) Get(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, name, namespace)
	ret0, _ := ret[0].(*v1.VolumeSnapshot)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockSnapshotterMockRecorder) Get(ctx, name, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockSnapshotter)(nil).Get), ctx, name, namespace)
}

// GetSource mocks base method.
func (m *MockSnapshotter) GetSource(ctx context.Context, snapshotName, namespace string) (*snapshot.Source, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSource", ctx, snapshotName, namespace)
	ret0, _ := ret[0].(*snapshot.Source)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSource indicates an expected call of GetSource.
func (mr *MockSnapshotterMockRecorder) GetSource(ctx, snapshotName, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSource", reflect.TypeOf((*MockSnapshotter)(nil).GetSource), ctx, snapshotName, namespace)
}

// GetVolumeSnapshotClass mocks base method.
func (m *MockSnapshotter) GetVolumeSnapshotClass(ctx context.Context, annotationKey, annotationValue, storageClassName string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolumeSnapshotClass", ctx, annotationKey, annotationValue, storageClassName)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolumeSnapshotClass indicates an expected call of GetVolumeSnapshotClass.
func (mr *MockSnapshotterMockRecorder) GetVolumeSnapshotClass(ctx, annotationKey, annotationValue, storageClassName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolumeSnapshotClass", reflect.TypeOf((*MockSnapshotter)(nil).GetVolumeSnapshotClass), ctx, annotationKey, annotationValue, storageClassName)
}

// GroupVersion mocks base method.
func (m *MockSnapshotter) GroupVersion(ctx context.Context) schema.GroupVersion {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GroupVersion", ctx)
	ret0, _ := ret[0].(schema.GroupVersion)
	return ret0
}

// GroupVersion indicates an expected call of GroupVersion.
func (mr *MockSnapshotterMockRecorder) GroupVersion(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GroupVersion", reflect.TypeOf((*MockSnapshotter)(nil).GroupVersion), ctx)
}

// List mocks base method.
func (m *MockSnapshotter) List(ctx context.Context, namespace string, labels map[string]string) (*v1.VolumeSnapshotList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, namespace, labels)
	ret0, _ := ret[0].(*v1.VolumeSnapshotList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockSnapshotterMockRecorder) List(ctx, namespace, labels interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockSnapshotter)(nil).List), ctx, namespace, labels)
}

// WaitOnReadyToUse mocks base method.
func (m *MockSnapshotter) WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitOnReadyToUse", ctx, snapshotName, namespace)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitOnReadyToUse indicates an expected call of WaitOnReadyToUse.
func (mr *MockSnapshotterMockRecorder) WaitOnReadyToUse(ctx, snapshotName, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitOnReadyToUse", reflect.TypeOf((*MockSnapshotter)(nil).WaitOnReadyToUse), ctx, snapshotName, namespace)
}
