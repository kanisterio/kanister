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
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes/fake"
)

type PodCommandExecutorTestSuite struct{}

var _ = Suite(&PodCommandExecutorTestSuite{})

const (
	podCommandExecutorNS            = "pod-runner-test"
	podCommandExecutorPodName       = "test-pod"
	podCommandExecutorContainerName = "test-container"
)

func (s *PodCommandExecutorTestSuite) SetUpSuite(c *C) {
	os.Setenv("POD_NAMESPACE", podCommandExecutorNS)
}

// // testBarrier supports race-free synchronization between a controller and a background goroutine.
// type testBarrier struct {
// 	fgStartedChan chan (struct{})
// 	bgStartedChan chan (struct{})
// }

// func (s *testBarrier) Setup() {
// 	s.bgStartedChan = make(chan struct{})
// 	s.fgStartedChan = make(chan struct{})
// }

// func (s *testBarrier) Sync() {
// 	if s.bgStartedChan != nil {
// 		<-s.bgStartedChan
// 		close(s.fgStartedChan)
// 	}
// }

// func (s *testBarrier) SyncWithController() { // background method
// 	if s.bgStartedChan != nil {
// 		close(s.bgStartedChan)
// 		<-s.fgStartedChan
// 	}
// }

// type fakePodCommandExecutorProcessor struct {
// 	inExecWithOptionsCli  kubernetes.Interface
// 	inExecWithOptionsOpts *ExecOptions
// 	execWithOptionsStdout string
// 	execWithOptionsStderr string
// 	execWithOptionsRet1   string
// 	execWithOptionsRet2   string
// 	execWithOptionsErr    error

// 	// Signal to `execWithOptions` to start "executing" command.
// 	// Command will remain "executing" until `execWithOptionsSyncEnd.Sync()`
// 	execWithOptionsSyncStart testBarrier
// 	execWithOptionsSyncEnd   testBarrier
// }

// func (fprp *fakePodCommandExecutorProcessor) execWithOptions(cli kubernetes.Interface, opts ExecOptions) (string, string, error) {
// 	fprp.inExecWithOptionsCli = cli
// 	fprp.inExecWithOptionsOpts = &opts
// 	fprp.execWithOptionsSyncStart.SyncWithController()
// 	if opts.Stdout != nil && len(fprp.execWithOptionsStdout) > 0 {
// 		opts.Stdout.Write([]byte(fprp.execWithOptionsStdout)) //nolint: errcheck
// 	}
// 	if opts.Stderr != nil && len(fprp.execWithOptionsStderr) > 0 {
// 		opts.Stderr.Write([]byte(fprp.execWithOptionsStderr)) //nolint: errcheck
// 	}
// 	fprp.execWithOptionsSyncEnd.SyncWithController()

// 	return fprp.execWithOptionsRet1, fprp.execWithOptionsRet2, fprp.execWithOptionsErr
// }

func (s *PodCommandExecutorTestSuite) TestPodRunnerExec(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	command := []string{"command", "arg1"}

	cases := map[string]func(ctx context.Context, pr PodCommandExecutor, prp *FakePodCommandExecutorProcessor){
		"Timed out": func(ctx context.Context, pr PodCommandExecutor, prp *FakePodCommandExecutorProcessor) {
			var err error
			// Prepare context which will timeout immediately
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Millisecond) // timeout within the call
			defer cancel()

			prp.ExecWithOptionsSyncStart.Setup()
			prp.ExecWithOptionsSyncEnd.Setup()
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
			prp.ExecWithOptionsSyncStart.Sync()
			wg.Wait()
			// allow the background goroutine to terminate (no-op if not Setup)
			prp.ExecWithOptionsSyncEnd.Sync()

			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, context.DeadlineExceeded), Equals, true)
		},
		"Cancelled": func(ctx context.Context, pr PodCommandExecutor, prp *FakePodCommandExecutorProcessor) {
			var err error
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Second*100)
			cancel() // prepare cancelled context
			prp.ExecWithOptionsSyncStart.Setup()
			prp.ExecWithOptionsSyncEnd.Setup()

			var bStdin, bStdout, bStderr bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err = pr.Exec(ctx, command, &bStdin, &bStdout, &bStderr)
				wg.Done()
			}()
			prp.ExecWithOptionsSyncStart.Sync() // Ensure ExecWithOptions is called
			wg.Wait()
			prp.ExecWithOptionsSyncEnd.Sync() // Release execWithOptions

			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, context.Canceled), Equals, true)
		},
		"Successfull execution": func(ctx context.Context, pr PodCommandExecutor, prp *FakePodCommandExecutorProcessor) {
			var err error
			prp.ExecWithOptionsStdout = "{\"where\":\"standard output\"}\n{\"what\":\"output json\"}"
			prp.ExecWithOptionsStderr = "{\"where\":\"standard error\"}\n{\"what\":\"error json\"}"
			expStdout := prp.ExecWithOptionsStdout
			expStderr := prp.ExecWithOptionsStderr

			var bStdin, bStdout, bStderr bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err = pr.Exec(ctx, command, &bStdin, &bStdout, &bStderr)
				wg.Done()
			}()
			prp.ExecWithOptionsSyncStart.Sync() // Ensure ExecWithOptions is called
			wg.Wait()
			prp.ExecWithOptionsSyncEnd.Sync() // Release execWithOptions

			c.Assert(err, IsNil)
			c.Assert(prp.InExecWithOptionsCli, Equals, cli)
			c.Assert(prp.InExecWithOptionsOpts.Command, DeepEquals, command)
			c.Assert(prp.InExecWithOptionsOpts.Namespace, Equals, podCommandExecutorNS)
			c.Assert(prp.InExecWithOptionsOpts.PodName, Equals, podCommandExecutorPodName)
			c.Assert(prp.InExecWithOptionsOpts.ContainerName, Equals, podCommandExecutorContainerName)
			c.Assert(prp.InExecWithOptionsOpts.Stdin, Equals, &bStdin)
			c.Assert(prp.InExecWithOptionsOpts.Stdout, Not(IsNil))
			c.Assert(prp.InExecWithOptionsOpts.Stderr, Not(IsNil))
			c.Assert(bStdout.Len() > 0, Equals, true)
			c.Assert(bStderr.Len() > 0, Equals, true)
			c.Assert(bStdout.String(), Equals, expStdout)
			c.Assert(bStderr.String(), Equals, expStderr)
		},
		"In case of failure, we have tail of logs": func(ctx context.Context, pr PodCommandExecutor, prp *FakePodCommandExecutorProcessor) {
			var errorLines []string
			var outputLines []string
			for i := 1; i <= 12; i++ {
				errorLines = append(errorLines, fmt.Sprintf("error line %d", i))
				outputLines = append(outputLines, fmt.Sprintf("output line %d", i))
			}

			var err error
			prp.ExecWithOptionsStdout = strings.Join(outputLines, "\n")
			prp.ExecWithOptionsStdout = strings.Join(errorLines, "\n")
			prp.ExecWithOptionsErr = errors.New("SimulatedError")

			expStdout := prp.ExecWithOptionsStdout
			expStderr := prp.ExecWithOptionsStderr
			expErrorStderr := strings.Join(errorLines[2:], "\r\n")
			expErrorStdout := strings.Join(outputLines[2:], "\r\n")

			var bStdin, bStdout, bStderr bytes.Buffer
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err = pr.Exec(ctx, command, &bStdin, &bStdout, &bStderr)
				wg.Done()
			}()
			prp.ExecWithOptionsSyncStart.Sync() // Ensure ExecWithOptions is called
			wg.Wait()
			prp.ExecWithOptionsSyncEnd.Sync() // Release execWithOptions

			c.Assert(err, Not(IsNil))
			c.Assert(prp.InExecWithOptionsOpts.Stdout, Not(IsNil))
			c.Assert(prp.InExecWithOptionsOpts.Stderr, Not(IsNil))
			c.Assert(bStdout.Len() > 0, Equals, true)
			c.Assert(bStderr.Len() > 0, Equals, true)
			c.Assert(bStdout.String(), Equals, expStdout)
			c.Assert(bStderr.String(), Equals, expStderr)

			var ee *ExecError
			c.Assert(errors.As(err, &ee), Equals, true)
			c.Assert(ee.Error(), Equals, "SimulatedError")
			c.Assert(ee.Stderr(), Equals, expErrorStderr)
			c.Assert(ee.Stdout(), Equals, expErrorStdout)
		},
	}

	for l, tc := range cases {
		c.Log(l)
		prp := &FakePodCommandExecutorProcessor{}

		pr := &podCommandExecutor{
			cli:           cli,
			namespace:     podCommandExecutorNS,
			podName:       podCommandExecutorPodName,
			containerName: podCommandExecutorContainerName,
			Pcep:          prp,
		}

		tc(ctx, pr, prp)
	}
}
