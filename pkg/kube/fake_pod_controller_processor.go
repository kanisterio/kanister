package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// FakePodControllerProcessor implements podControllerProcessor
type FakePodControllerProcessor struct {
	InWaitForPodReadyNamespace string
	InWaitForPodReadyPodName   string
	WaitForPodReadyErr         error

	InWaitForPodCompletionNamespace string
	InWaitForPodCompletionPodName   string
	WaitForPodCompletionErr         error

	InDeletePodNamespace string
	InDeletePodPodName   string
	InDeletePodOptions   metav1.DeleteOptions
	DeletePodErr         error

	InCreatePodCli     kubernetes.Interface
	InCreatePodOptions *PodOptions
	CreatePodRet       *corev1.Pod
	CreatePodErr       error
}

func (f *FakePodControllerProcessor) CreatePod(_ context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error) {
	f.InCreatePodCli = cli
	f.InCreatePodOptions = options
	return f.CreatePodRet, f.CreatePodErr
}

func (f *FakePodControllerProcessor) WaitForPodCompletionPCP(ctx context.Context, namespace, podName string) error {
	f.InWaitForPodCompletionNamespace = namespace
	f.InWaitForPodCompletionPodName = podName
	return f.WaitForPodCompletionErr
}

func (f *FakePodControllerProcessor) WaitForPodReadyPCP(ctx context.Context, namespace, podName string) error {
	f.InWaitForPodReadyPodName = podName
	f.InWaitForPodReadyNamespace = namespace
	return f.WaitForPodReadyErr
}

func (f *FakePodControllerProcessor) DeletePod(_ context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	f.InDeletePodNamespace = namespace
	f.InDeletePodPodName = podName
	f.InDeletePodOptions = opts

	return f.DeletePodErr
}
