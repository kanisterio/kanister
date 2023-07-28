package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// FakePodControllerProcessor is a fake pod controller processor
type FakePodControllerProcessor struct {
	inWaitForPodReadyNamespace string
	inWaitForPodReadyPodName   string
	waitForPodReadyErr         error

	inWaitForPodCompletionNamespace string
	inWaitForPodCompletionPodName   string
	waitForPodCompletionErr         error

	inDeletePodNamespace string
	inDeletePodPodName   string
	inDeletePodOptions   metav1.DeleteOptions
	deletePodErr         error

	inCreatePodCli     kubernetes.Interface
	inCreatePodOptions *PodOptions
	createPodRet       *corev1.Pod
	createPodErr       error
}

func (f *FakePodControllerProcessor) createPod(_ context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error) {
	f.inCreatePodCli = cli
	f.inCreatePodOptions = options
	return f.createPodRet, f.createPodErr
}

func (f *FakePodControllerProcessor) waitForPodCompletion(ctx context.Context, namespace, podName string) error {
	f.inWaitForPodCompletionNamespace = namespace
	f.inWaitForPodCompletionPodName = podName
	return f.waitForPodCompletionErr
}

func (f *FakePodControllerProcessor) waitForPodReady(ctx context.Context, namespace, podName string) error {
	f.inWaitForPodReadyPodName = podName
	f.inWaitForPodReadyNamespace = namespace
	return f.waitForPodReadyErr
}

func (f *FakePodControllerProcessor) deletePod(_ context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	f.inDeletePodNamespace = namespace
	f.inDeletePodPodName = podName
	f.inDeletePodOptions = opts

	return f.deletePodErr
}
