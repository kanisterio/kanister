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
	"context"
	"io"
	"path/filepath"

	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
)

// PodWriter specifies interface needed for manipulating files in a pod
type PodWriter interface {
	// Write will create a new file(if not present) and write the provided content to the file
	Write(ctx context.Context, namespace, podName, containerName string) error
	// Remove will delete the file created by Write() func
	Remove(ctx context.Context, namespace, podName, containerName string) error
}

// podWriter specifies Kubernetes Client and the other params needed for writing content to a file
type podWriter struct {
	cli     kubernetes.Interface
	path    string
	content io.Reader
}

var _ PodWriter = (*podWriter)(nil)

// NewPodWriter returns a new PodWriter given Kubernetes Client, path of file and content
func NewPodWriter(cli kubernetes.Interface, path string, content io.Reader) PodWriter {
	return &podWriter{
		cli:     cli,
		path:    filepath.Clean(path),
		content: content,
	}
}

// Write will create a new file(if not present) and write the provided content to the file
func (p *podWriter) Write(ctx context.Context, namespace, podName, containerName string) error {
	cmd := []string{"sh", "-c", "cat - > " + p.path + " && :"}
	stdout, stderr, err := Exec(ctx, p.cli, namespace, podName, containerName, cmd, p.content)
	format.LogWithCtx(ctx, podName, containerName, stdout)
	format.LogWithCtx(ctx, podName, containerName, stderr)
	if err != nil {
		return errkit.Wrap(err, "Failed to write contents to file")
	}

	return nil
}

// Remove will delete the file created by Write() func
func (p *podWriter) Remove(ctx context.Context, namespace, podName, containerName string) error {
	cmd := []string{"sh", "-c", "rm " + p.path}
	stdout, stderr, err := Exec(ctx, p.cli, namespace, podName, containerName, cmd, nil)
	format.LogWithCtx(ctx, podName, containerName, stdout)
	format.LogWithCtx(ctx, podName, containerName, stderr)
	if err != nil {
		return errkit.Wrap(err, "Failed to delete file")
	}

	return nil
}
