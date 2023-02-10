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

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

// PodFileRemover provides the mechanism to remove written file from the pod.
type PodFileRemover interface {
	Remove(ctx context.Context) error
	Path() string
}

type podFileRemover struct {
	namespace     string
	podName       string
	containerName string
	podWriter     PodWriter
	path          string
}

func (pfr *podFileRemover) Remove(ctx context.Context) error {
	return pfr.podWriter.Remove(ctx, pfr.namespace, pfr.podName, pfr.containerName)
}

func (pfr *podFileRemover) Path() string {
	return pfr.path
}

// PodFileWriter allows us to write file to the pod.
type PodFileWriter interface {
	Write(ctx context.Context, filePath string, content io.Reader) (PodFileRemover, error)
}

// podFileWriterProcessor aids in unit testing.
type podFileWriterProcessor interface {
	newPodWriter(cli kubernetes.Interface, filePath string, content io.Reader) PodWriter
}

// podFileWriter keeps everything required to write a file to POD.
type podFileWriter struct {
	cli           kubernetes.Interface
	podName       string
	namespace     string
	containerName string

	pfwp podFileWriterProcessor
}

// WriteFileToPod writes specified file content to a file in the pod and returns an interface
// with which the file can be removed.
func (p *podFileWriter) Write(ctx context.Context, filePath string, content io.Reader) (PodFileRemover, error) {
	pw := p.pfwp.newPodWriter(p.cli, filePath, content)
	if err := pw.Write(ctx, p.namespace, p.podName, p.containerName); err != nil {
		return nil, errors.Wrap(err, "Write file to pod failed")
	}

	return &podFileRemover{
		namespace:     p.namespace,
		podName:       p.podName,
		containerName: p.containerName,
		podWriter:     pw,
		path:          filePath,
	}, nil
}

func (p *podFileWriter) newPodWriter(cli kubernetes.Interface, filepath string, content io.Reader) PodWriter {
	return NewPodWriter(cli, filepath, content)
}
