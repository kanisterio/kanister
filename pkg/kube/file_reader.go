// Copyright 2022 The Kanister Authors.
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
	"path"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type PodFileReader struct {
	cli       kubernetes.Interface
	podName   string
	namespace string
	container string
}

func NewPodFileReader(cli kubernetes.Interface, podName, namespace, container string) *PodFileReader {
	return &PodFileReader{
		cli:       cli,
		podName:   podName,
		namespace: namespace,
		container: container,
	}
}

func (r *PodFileReader) ReadFile(ctx context.Context, path string) (string, error) {
	cmd := []string{"sh", "-c", "cat " + path}
	stdout, stderr, err := Exec(r.cli, r.namespace, r.podName, r.container, cmd, nil)
	if err != nil {
		if stderr != "" {
			log.Print("Error executing command", field.M{"stderr": stderr})
		}
		return "", errors.Wrap(err, "Failed to write contents to file")
	}
	return stdout, nil
}

func (r *PodFileReader) ReadDir(ctx context.Context, dirPath string) (map[string]string, error) {
	cmd := []string{"sh", "-c", "ls -1 " + dirPath}
	stdout, stderr, err := Exec(r.cli, r.namespace, r.podName, r.container, cmd, nil)
	if err != nil {
		if stderr != "" {
			log.Print("Error executing command", field.M{"stderr": stderr})
		}
		return nil, errors.Wrap(err, "Failed to list files of directory")
	}
	op := map[string]string{}
	data := strings.Split(stdout, "\n")
	for _, file := range data {
		out, err := r.ReadFile(ctx, path.Join(dirPath, file))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read contents of file")
		}
		op[file] = out
	}
	return op, nil
}
