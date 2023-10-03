// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"io"

	"k8s.io/client-go/kubernetes"
)

// PodFileWriterProcessor is an interface wrapping kubernetes API invocation
// it is purposed to be replaced by fake implementation in tests
type PodFileWriterProcessor interface {
	NewPodWriter(filePath string, content io.Reader) PodWriter
}

type podFileWriterProcessor struct {
	cli kubernetes.Interface
}

// NewPodWriter returns a new PodWriter given path of file and content
func (p *podFileWriterProcessor) NewPodWriter(filepath string, content io.Reader) PodWriter {
	return NewPodWriter(p.cli, filepath, content)
}
