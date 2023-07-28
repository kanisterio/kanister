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
	"k8s.io/client-go/kubernetes"
)

// testBarrier supports race-free synchronization between a controller and a background goroutine.
type testBarrier struct {
	fgStartedChan chan (struct{})
	bgStartedChan chan (struct{})
}

// FakePodCommandExecutorProcessor implements PodCommandExecutor interface
type FakePodCommandExecutorProcessor struct {
	InExecWithOptionsCli  kubernetes.Interface
	InExecWithOptionsOpts *ExecOptions
	ExecWithOptionsStdout string
	ExecWithOptionsStderr string
	ExecWithOptionsRet1   string
	ExecWithOptionsRet2   string
	ExecWithOptionsErr    error

	// Signal to `execWithOptions` to start "executing" command.
	// Command will remain "executing" until `execWithOptionsSyncEnd.Sync()`
	ExecWithOptionsSyncStart testBarrier
	ExecWithOptionsSyncEnd   testBarrier
}

func (s *testBarrier) Setup() {
	s.bgStartedChan = make(chan struct{})
	s.fgStartedChan = make(chan struct{})
}

func (s *testBarrier) Sync() {
	if s.bgStartedChan != nil {
		<-s.bgStartedChan
		close(s.fgStartedChan)
	}
}

func (s *testBarrier) SyncWithController() { // background method
	if s.bgStartedChan != nil {
		close(s.bgStartedChan)
		<-s.fgStartedChan
	}
}

func (fprp *FakePodCommandExecutorProcessor) execWithOptions(cli kubernetes.Interface, opts ExecOptions) (string, string, error) {
	fprp.InExecWithOptionsCli = cli
	fprp.InExecWithOptionsOpts = &opts
	fprp.ExecWithOptionsSyncStart.SyncWithController()
	if opts.Stdout != nil && len(fprp.ExecWithOptionsStdout) > 0 {
		opts.Stdout.Write([]byte(fprp.ExecWithOptionsStdout)) //nolint: errcheck
	}
	if opts.Stderr != nil && len(fprp.ExecWithOptionsStderr) > 0 {
		opts.Stderr.Write([]byte(fprp.ExecWithOptionsStderr)) //nolint: errcheck
	}
	fprp.ExecWithOptionsSyncEnd.SyncWithController()

	return fprp.ExecWithOptionsRet1, fprp.ExecWithOptionsRet2, fprp.ExecWithOptionsErr
}
