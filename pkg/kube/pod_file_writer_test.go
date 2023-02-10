// Copyright 2019 The Kanister Authors.
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
	"io"
	"os"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type PodFileWriterTestSuite struct{}

var _ = Suite(&PodFileWriterTestSuite{})

const (
	podFileWriterNS            = "pod-runner-test"
	podFileWriterPodName       = "test-pod"
	podFileWriterContainerName = "test-container"
)

func (s *PodFileWriterTestSuite) SetUpSuite(c *C) {
	os.Setenv("POD_NAMESPACE", podFileWriterNS)
}

type fakePodFileWriterProcessor struct {
	podWriter *fakePodWriter
}

func (fprp *fakePodFileWriterProcessor) newPodWriter(cli kubernetes.Interface, filepath string, content io.Reader) PodWriter {
	return fprp.podWriter
}

type fakePodWriter struct {
	inWriteNamespace     string
	inWritePodName       string
	inWriteContainerName string
	writeErr             error

	inRemoveNamespace     string
	inRemovePodName       string
	inRemoveContainerName string
	removeErr             error
}

func (fpw *fakePodWriter) Write(ctx context.Context, namespace, podName, containerName string) error {
	fpw.inWriteNamespace = namespace
	fpw.inWritePodName = podName
	fpw.inWriteContainerName = containerName
	return fpw.writeErr
}
func (fpw *fakePodWriter) Remove(ctx context.Context, namespace, podName, containerName string) error {
	fpw.inRemoveNamespace = namespace
	fpw.inRemovePodName = podName
	fpw.inRemoveContainerName = containerName
	return fpw.removeErr
}

var _ PodWriter = (*fakePodWriter)(nil)

func (s *PodFileWriterTestSuite) TestPodRunnerWriteFile(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	simulatedError := errors.New("SimulatedError")

	cases := map[string]func(pfwp *fakePodFileWriterProcessor, pfw PodFileWriter){
		"Write to pod failed": func(pfwp *fakePodFileWriterProcessor, pfw PodFileWriter) {
			pfwp.podWriter = &fakePodWriter{}
			pfwp.podWriter.writeErr = simulatedError

			buf := bytes.NewBuffer([]byte("some file content"))
			remover, err := pfw.Write(ctx, "/path/to/file", buf)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, simulatedError), Equals, true)
			c.Assert(remover, IsNil)

			c.Assert(pfwp.podWriter.inWriteNamespace, Equals, podFileWriterNS)
			c.Assert(pfwp.podWriter.inWritePodName, Equals, podFileWriterPodName)
			c.Assert(pfwp.podWriter.inWriteContainerName, Equals, podFileWriterContainerName)
		},
		"Write to pod succeeded": func(pfwp *fakePodFileWriterProcessor, pfw PodFileWriter) {
			pfwp.podWriter = &fakePodWriter{}

			buf := bytes.NewBuffer([]byte("some file content"))
			remover, err := pfw.Write(ctx, "/path/to/file", buf)
			c.Assert(err, IsNil)
			c.Assert(remover, Not(IsNil))

			c.Assert(pfwp.podWriter.inWriteNamespace, Equals, podFileWriterNS)
			c.Assert(pfwp.podWriter.inWritePodName, Equals, podFileWriterPodName)
			c.Assert(pfwp.podWriter.inWriteContainerName, Equals, podFileWriterContainerName)

			err = remover.Remove(ctx)
			c.Assert(err, IsNil)
			c.Assert(pfwp.podWriter.inRemoveNamespace, Equals, podFileWriterNS)
			c.Assert(pfwp.podWriter.inRemovePodName, Equals, podFileWriterPodName)
			c.Assert(pfwp.podWriter.inRemoveContainerName, Equals, podFileWriterContainerName)
		},
		"Write to pod succeeded but remove failed": func(pfwp *fakePodFileWriterProcessor, pfw PodFileWriter) {
			pfwp.podWriter = &fakePodWriter{}
			pfwp.podWriter.removeErr = simulatedError

			buf := bytes.NewBuffer([]byte("some file content"))
			remover, err := pfw.Write(ctx, "/path/to/file", buf)
			c.Assert(err, IsNil)
			c.Assert(remover, Not(IsNil))

			c.Assert(pfwp.podWriter.inWriteNamespace, Equals, podFileWriterNS)
			c.Assert(pfwp.podWriter.inWritePodName, Equals, podFileWriterPodName)
			c.Assert(pfwp.podWriter.inWriteContainerName, Equals, podFileWriterContainerName)

			err = remover.Remove(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, simulatedError), Equals, true)
			c.Assert(pfwp.podWriter.inRemoveNamespace, Equals, podFileWriterNS)
			c.Assert(pfwp.podWriter.inRemovePodName, Equals, podFileWriterPodName)
			c.Assert(pfwp.podWriter.inRemoveContainerName, Equals, podFileWriterContainerName)
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pfwp := &fakePodFileWriterProcessor{}

		pr := &podFileWriter{
			cli:           cli,
			podName:       podFileWriterPodName,
			namespace:     podFileWriterNS,
			containerName: podFileWriterContainerName,
			pfwp:          pfwp,
		}

		tc(pfwp, pr)
	}
}
