package kube

import (
	"k8s.io/client-go/kubernetes"
)

// PodCommandExecutorProcessor is an interface wrapping kubernetes API invocation
// it is purposed to be replaced by fake implementation in tests
type PodCommandExecutorProcessor interface {
	ExecWithOptions(opts ExecOptions) (string, string, error)
}

type podCommandExecutorProcessor struct {
	cli kubernetes.Interface
}

// ExecWithOptions executes a command in the specified pod and container,
// returning stdout, stderr and error.
func (p *podCommandExecutorProcessor) ExecWithOptions(opts ExecOptions) (string, string, error) {
	return ExecWithOptions(p.cli, opts)
}
