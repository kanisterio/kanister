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
