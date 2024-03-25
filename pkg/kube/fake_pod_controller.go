// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"context"
	"io"
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
)

type FakePodController struct {
	Podname string
	PodObj  *corev1.Pod

	StartPodCalled bool
	StartPodErr    error

	WaitForPodReadyCalled bool
	WaitForPodReadyErr    error

	GetCommandExecutorRet PodCommandExecutor
	GetCommandExecutorErr error

	GetFileWriterCalled bool
	GetFileWriterRet    *FakePodFileWriter
	GetFileWriterErr    error

	StopPodCalled        bool
	StopPodErr           error
	InStopPodStopTimeout time.Duration
	InStopPodGracePeriod int64
}

func (fpc *FakePodController) Pod() *corev1.Pod {
	return nil
}

func (fpc *FakePodController) PodName() string {
	return fpc.Podname
}

func (fpc *FakePodController) Run(ctx context.Context, fn func(context.Context, *corev1.Pod) (map[string]interface{}, error)) (map[string]interface{}, error) {
	return nil, errkit.New("Not implemented")
}

func (fpc *FakePodController) StartPod(_ context.Context) error {
	fpc.StartPodCalled = true
	return fpc.StartPodErr
}

func (fpc *FakePodController) WaitForPodReady(_ context.Context) error {
	fpc.WaitForPodReadyCalled = true
	return fpc.WaitForPodReadyErr
}

func (fpc *FakePodController) WaitForPodCompletion(_ context.Context) error {
	return errkit.New("Not implemented")
}

func (fpc *FakePodController) StreamPodLogs(_ context.Context) (io.ReadCloser, error) {
	return nil, errkit.New("Not implemented")
}

func (fpc *FakePodController) GetCommandExecutor() (PodCommandExecutor, error) {
	return fpc.GetCommandExecutorRet, fpc.GetCommandExecutorErr
}

func (fpc *FakePodController) GetFileWriter() (PodFileWriter, error) {
	fpc.GetFileWriterCalled = true
	return fpc.GetFileWriterRet, fpc.GetFileWriterErr
}

func (fpc *FakePodController) StopPod(ctx context.Context, stopTimeout time.Duration, gracePeriodSeconds int64) error {
	fpc.StopPodCalled = true
	fpc.InStopPodStopTimeout = stopTimeout
	fpc.InStopPodGracePeriod = gracePeriodSeconds
	return fpc.StopPodErr
}

type FakePodFileWriter struct {
	writeCalled     bool
	writeErr        error
	writeRet        *FakePodFileRemover
	inWriteFilePath string
	inWriteContent  io.Reader
}

func (fpfw *FakePodFileWriter) Write(_ context.Context, filePath string, content io.Reader) (PodFileRemover, error) {
	fpfw.writeCalled = true
	fpfw.inWriteFilePath = filePath
	fpfw.inWriteContent = content

	return PodFileRemover(fpfw.writeRet), fpfw.writeErr
}

type FakePodFileRemover struct {
	removeCalled bool
	removeErr    error
	path         string
}

func (fr *FakePodFileRemover) Remove(_ context.Context) error {
	fr.removeCalled = true
	return fr.removeErr
}

func (fr *FakePodFileRemover) Path() string {
	return fr.path
}
