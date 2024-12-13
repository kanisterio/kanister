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
	"bytes"
	"context"
	"os"
	"sync"
	"time"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes/fake"
)

type PodCommandExecutorTestSuite struct{}

var _ = check.Suite(&PodCommandExecutorTestSuite{})

const (
	podCommandExecutorNS            = "pod-runner-test"
	podCommandExecutorPodName       = "test-pod"
	podCommandExecutorContainerName = "test-container"
)

func (s *PodCommandExecutorTestSuite) SetUpSuite(c *check.C) {
	err := os.Setenv("POD_NAMESPACE", podCommandExecutorNS)
	c.Assert(err, check.IsNil)
}

// testBarrier supports race-free synchronization between a controller and a background goroutine.
type testBarrier struct {
	fgStartedChan chan (struct{})
	bgStartedChan chan (struct{})
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

type fakePodCommandExecutorProcessor struct {
	inExecWithOptionsOpts *ExecOptions
	execWithOptionsStdout string
	execWithOptionsStderr string
	execWithOptionsErr    error

	// Signal to `ExecWithOptions` to start "executing" command.
	// Command will remain "executing" until `execWithOptionsSyncEnd.Sync()`
	execWithOptionsSyncStart testBarrier
	execWithOptionsSyncEnd   testBarrier
}

func (fprp *fakePodCommandExecutorProcessor) ExecWithOptions(ctx context.Context, opts ExecOptions) error {
	fprp.inExecWithOptionsOpts = &opts
	fprp.execWithOptionsSyncStart.SyncWithController()
	if opts.Stdout != nil && len(fprp.execWithOptionsStdout) > 0 {
		if _, err := opts.Stdout.Write([]byte(fprp.execWithOptionsStdout)); err != nil {
			return err
		}
	}
	if opts.Stderr != nil && len(fprp.execWithOptionsStderr) > 0 {
		if _, err := opts.Stderr.Write([]byte(fprp.execWithOptionsStderr)); err != nil {
			return err
		}
	}
	fprp.execWithOptionsSyncEnd.SyncWithController()

	return fprp.execWithOptionsErr
}

func (s *PodCommandExecutorTestSuite) TestPodRunnerExec(c *check.C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	command := []string{"command", "arg1"}

	cases := map[string]func(ctx context.Context, pr PodCommandExecutor, prp *fakePodCommandExecutorProcessor){
		"Timed out": func(ctx context.Context, pr PodCommandExecutor, prp *fakePodCommandExecutorProcessor) {
			var err error
			// Prepare context which will timeout immediately
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Millisecond) // timeout within the call
			defer cancel()

			prp.execWithOptionsSyncStart.Setup()
			prp.execWithOptionsSyncEnd.Setup()
			var bStdin, bStdout, bStderr bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err = pr.Exec(ctx, command, &bStdin, &bStdout, &bStderr)
				wg.Done()
			}()
			// Signal to `Exec` to start "executing" command. Command will remain "executing"
			// until `syncEndKubeExecWithOptions.Sync()`, which won't happen until an error is returned
			// from `Exec` and `WaitGroup` is released. This guarantees the error returned is from
			// the expired Context.
			prp.execWithOptionsSyncStart.Sync()
			wg.Wait()
			// allow the background goroutine to terminate (no-op if not Setup)
			prp.execWithOptionsSyncEnd.Sync()

			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, context.DeadlineExceeded), check.Equals, true)
		},
		"Cancelled": func(ctx context.Context, pr PodCommandExecutor, prp *fakePodCommandExecutorProcessor) {
			var err error
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Second*100)
			cancel() // prepare cancelled context
			prp.execWithOptionsSyncStart.Setup()
			prp.execWithOptionsSyncEnd.Setup()

			var bStdin, bStdout, bStderr bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err = pr.Exec(ctx, command, &bStdin, &bStdout, &bStderr)
				wg.Done()
			}()
			prp.execWithOptionsSyncStart.Sync() // Ensure ExecWithOptions is called
			wg.Wait()
			prp.execWithOptionsSyncEnd.Sync() // Release ExecWithOptions

			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, context.Canceled), check.Equals, true)
		},
		"Successful execution": func(ctx context.Context, pr PodCommandExecutor, prp *fakePodCommandExecutorProcessor) {
			var err error
			prp.execWithOptionsStdout = "{\"where\":\"standard output\"}\n{\"what\":\"output json\"}"
			prp.execWithOptionsStderr = "{\"where\":\"standard error\"}\n{\"what\":\"error json\"}"
			expStdout := prp.execWithOptionsStdout
			expStderr := prp.execWithOptionsStderr

			var bStdin, bStdout, bStderr bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err = pr.Exec(ctx, command, &bStdin, &bStdout, &bStderr)
				wg.Done()
			}()
			prp.execWithOptionsSyncStart.Sync() // Ensure ExecWithOptions is called
			wg.Wait()
			prp.execWithOptionsSyncEnd.Sync() // Release ExecWithOptions

			c.Assert(err, check.IsNil)
			c.Assert(prp.inExecWithOptionsOpts.Command, check.DeepEquals, command)
			c.Assert(prp.inExecWithOptionsOpts.Namespace, check.Equals, podCommandExecutorNS)
			c.Assert(prp.inExecWithOptionsOpts.PodName, check.Equals, podCommandExecutorPodName)
			c.Assert(prp.inExecWithOptionsOpts.ContainerName, check.Equals, podCommandExecutorContainerName)
			c.Assert(prp.inExecWithOptionsOpts.Stdin, check.Equals, &bStdin)
			c.Assert(prp.inExecWithOptionsOpts.Stdout, check.Not(check.IsNil))
			c.Assert(prp.inExecWithOptionsOpts.Stderr, check.Not(check.IsNil))
			c.Assert(bStdout.Len() > 0, check.Equals, true)
			c.Assert(bStderr.Len() > 0, check.Equals, true)
			c.Assert(bStdout.String(), check.Equals, expStdout)
			c.Assert(bStderr.String(), check.Equals, expStderr)
		},
	}

	for l, tc := range cases {
		c.Log(l)
		prp := &fakePodCommandExecutorProcessor{}

		pr := &podCommandExecutor{
			cli:           cli,
			namespace:     podCommandExecutorNS,
			podName:       podCommandExecutorPodName,
			containerName: podCommandExecutorContainerName,
			pcep:          prp,
		}

		tc(ctx, pr, prp)
	}
}
