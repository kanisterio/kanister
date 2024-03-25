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

	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"
)

// PodFileRemover provides mechanism for removing particular file written to the pod by PodFileWriter.
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

// Remove deletes file from the pod.
func (pfr *podFileRemover) Remove(ctx context.Context) error {
	return pfr.podWriter.Remove(ctx, pfr.namespace, pfr.podName, pfr.containerName)
}

// Path returns path of the file within pod to be removed.
func (pfr *podFileRemover) Path() string {
	return pfr.path
}

// PodFileWriter provides a way to write a file to the pod.
// Is intended to be returned by PodController and works with pod controlled by it.
type PodFileWriter interface {
	Write(ctx context.Context, filePath string, content io.Reader) (PodFileRemover, error)
}

// podFileWriter keeps everything required to write a file to POD.
type podFileWriter struct {
	cli           kubernetes.Interface
	podName       string
	namespace     string
	containerName string

	fileWriterProcessor PodFileWriterProcessor
}

// Write writes specified file content to a file in the pod and returns PodFileRemover
// which should be used to remove written file.
func (p *podFileWriter) Write(ctx context.Context, filePath string, content io.Reader) (PodFileRemover, error) {
	pw := p.fileWriterProcessor.NewPodWriter(filePath, content)
	if err := pw.Write(ctx, p.namespace, p.podName, p.containerName); err != nil {
		return nil, errkit.Wrap(err, "Write file to pod failed")
	}

	return &podFileRemover{
		namespace:     p.namespace,
		podName:       p.podName,
		containerName: p.containerName,
		podWriter:     pw,
		path:          filePath,
	}, nil
}
