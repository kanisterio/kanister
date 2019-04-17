package kube

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// PodRunner specifies Kubernetes Client and PodOptions needed for creating Pod
type PodRunner struct {
	cli        kubernetes.Interface
	podOptions *PodOptions
}

// NewPodRunner returns a new PodRunner given Kubernetes Client and PodOptions
func NewPodRunner(cli kubernetes.Interface, options *PodOptions) *PodRunner {
	return &PodRunner{
		cli:        cli,
		podOptions: options,
	}
}

// Run will create a new Pod based on PodRunner contents and execute the given function
func (p *PodRunner) Run(ctx context.Context, fn func(context.Context, *v1.Pod) (map[string]interface{}, error)) (map[string]interface{}, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if p.cli == nil || p.podOptions == nil {
		return nil, errors.New("Pod Runner not initialized")
	}
	pod, err := CreatePod(ctx, p.cli, p.podOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod")
	}
	go func() {
		select {
		case <-ctx.Done():
			defer DeletePod(context.Background(), p.cli, pod)
		}
	}()
	// Wait for pod to reach running state
	if err := WaitForPodReady(ctx, p.cli, pod.Namespace, pod.Name); err != nil {
		return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
	}
	return fn(ctx, pod)
}
