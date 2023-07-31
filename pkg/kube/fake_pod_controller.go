package kube

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type FakeKubePodController struct {
	Podname string
	PodObj  *corev1.Pod

	StartPodCalled bool
	startPodErr    error

	WaitForPodReadyCalled bool
	WaitForPodReadyErr    error

	// getCommandExecutorCalled bool
	GetCommandExecutorRet PodCommandExecutor
	GetCommandExecutorErr error

	GetFileWriterCalled bool
	GetFileWriterRet    *FakeKubePodFileWriter
	GetFileWriterErr    error

	StopPodCalled        bool
	StopPodErr           error
	InStopPodStopTimeout time.Duration
	InStopPodGracePeriod int64
}

func (fkpr *FakeKubePodController) Pod() *corev1.Pod {
	return nil
}

func (fkpr *FakeKubePodController) PodName() string {
	return fkpr.Podname
}

func (fkpr *FakeKubePodController) Run(ctx context.Context, fn func(context.Context, *corev1.Pod) (map[string]interface{}, error)) (map[string]interface{}, error) {
	return nil, errors.New("Not implemented")
}

func (fkpr *FakeKubePodController) StartPod(_ context.Context) error {
	fkpr.StartPodCalled = true
	return fkpr.startPodErr
}

func (fkpr *FakeKubePodController) WaitForPodReady(_ context.Context) error {
	fkpr.WaitForPodReadyCalled = true
	return fkpr.WaitForPodReadyErr
}

func (fkpr *FakeKubePodController) WaitForPodCompletion(_ context.Context) error {
	return errors.New("Not implemented")
}

func (fkpr *FakeKubePodController) StreamPodLogs(_ context.Context) (io.ReadCloser, error) {
	return nil, errors.New("Not implemented")
}

func (fkpr *FakeKubePodController) GetCommandExecutor() (PodCommandExecutor, error) {
	return fkpr.GetCommandExecutorRet, fkpr.GetCommandExecutorErr
}

func (fkpr *FakeKubePodController) GetFileWriter() (PodFileWriter, error) {
	fkpr.GetFileWriterCalled = true
	return fkpr.GetFileWriterRet, fkpr.GetFileWriterErr
}

func (fkpr *FakeKubePodController) StopPod(ctx context.Context, stopTimeout time.Duration, gracePeriodSeconds int64) error {
	fkpr.StopPodCalled = true
	fkpr.InStopPodStopTimeout = stopTimeout
	fkpr.InStopPodGracePeriod = gracePeriodSeconds
	return fkpr.StopPodErr
}

type FakeKubePodFileWriter struct {
	writeCalled     bool
	writeErr        error
	writeRet        *FakeKubePodFileRemover
	inWriteFilePath string
	inWriteContent  io.Reader
}

func (fkpfw *FakeKubePodFileWriter) Write(_ context.Context, filePath string, content io.Reader) (PodFileRemover, error) {
	fkpfw.writeCalled = true
	fkpfw.inWriteFilePath = filePath
	fkpfw.inWriteContent = content

	return PodFileRemover(fkpfw.writeRet), fkpfw.writeErr
}

type FakeKubePodFileRemover struct {
	removeCalled bool
	removeErr    error
	path         string
}

func (fr *FakeKubePodFileRemover) Remove(_ context.Context) error {
	fr.removeCalled = true
	return fr.removeErr
}

func (fr *FakeKubePodFileRemover) Path() string {
	return fr.path
}
