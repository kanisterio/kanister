// Copyright 2019 Kasten Inc.
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

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
)

// PodWriter specifies Kubernetes Client and the other params needed for writing content to a file
type PodWriter struct {
	cli     kubernetes.Interface
	path    string
	content io.Reader
}

// NewPodWriter returns a new PodWriter given Kubernetes Client, path of file and content
func NewPodWriter(cli kubernetes.Interface, path string, content io.Reader) *PodWriter {
	return &PodWriter{
		cli:     cli,
		path:    filepath.Clean(path),
		content: content,
	}
}

// Write will create a new file(if not present) and write the provided content to the file
func (p *PodWriter) Write(ctx context.Context, namespace, podName, containerName string) error {
	cmd := []string{"sh", "-c", "cat - > " + p.path}
	stdout, stderr, err := Exec(p.cli, namespace, podName, containerName, cmd, p.content)
	format.Log(podName, containerName, stdout)
	format.Log(podName, containerName, stderr)
	return errors.Wrap(err, "Failed to write contents to file")
}

// Remove will delete the file created by Write() func
func (p *PodWriter) Remove(ctx context.Context, namespace, podName, containerName string) error {
	cmd := []string{"sh", "-c", "rm " + p.path}
	stdout, stderr, err := Exec(p.cli, namespace, podName, containerName, cmd, nil)
	format.Log(podName, containerName, stdout)
	format.Log(podName, containerName, stderr)
	return errors.Wrap(err, "Failed to delete file")
}
