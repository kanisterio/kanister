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
	StartPodErr    error

	WaitForPodReadyCalled bool
	WaitForPodReadyErr    error

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

func (fkpc *FakeKubePodController) Pod() *corev1.Pod {
	return nil
}

func (fkpc *FakeKubePodController) PodName() string {
	return fkpc.Podname
}

func (fkpc *FakeKubePodController) Run(ctx context.Context, fn func(context.Context, *corev1.Pod) (map[string]interface{}, error)) (map[string]interface{}, error) {
	return nil, errors.New("Not implemented")
}

func (fkpc *FakeKubePodController) StartPod(_ context.Context) error {
	fkpc.StartPodCalled = true
	return fkpc.StartPodErr
}

func (fkpc *FakeKubePodController) WaitForPodReady(_ context.Context) error {
	fkpc.WaitForPodReadyCalled = true
	return fkpc.WaitForPodReadyErr
}

func (fkpc *FakeKubePodController) WaitForPodCompletion(_ context.Context) error {
	return errors.New("Not implemented")
}

func (fkpc *FakeKubePodController) StreamPodLogs(_ context.Context) (io.ReadCloser, error) {
	return nil, errors.New("Not implemented")
}

func (fkpc *FakeKubePodController) GetCommandExecutor() (PodCommandExecutor, error) {
	return fkpc.GetCommandExecutorRet, fkpc.GetCommandExecutorErr
}

func (fkpc *FakeKubePodController) GetFileWriter() (PodFileWriter, error) {
	fkpc.GetFileWriterCalled = true
	return fkpc.GetFileWriterRet, fkpc.GetFileWriterErr
}

func (fkpc *FakeKubePodController) StopPod(ctx context.Context, stopTimeout time.Duration, gracePeriodSeconds int64) error {
	fkpc.StopPodCalled = true
	fkpc.InStopPodStopTimeout = stopTimeout
	fkpc.InStopPodGracePeriod = gracePeriodSeconds
	return fkpc.StopPodErr
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
