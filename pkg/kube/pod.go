package kube

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
)

// PodOptions specifies options for `CreatePod`
type PodOptions struct {
	Namespace    string
	GenerateName string
	Image        string
	Command      []string
	Volumes      map[string]string
}

// CreatePod creates a pod with a single container based on the specified image
func CreatePod(ctx context.Context, cli kubernetes.Interface, opts *PodOptions) (*v1.Pod, error) {
	volumeMounts, podVolumes := createVolumeSpecs(opts.Volumes)
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: opts.GenerateName,
			Namespace:    opts.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:            "container",
					Image:           opts.Image,
					Command:         opts.Command,
					ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
					VolumeMounts:    volumeMounts,
				},
			},
			Volumes: podVolumes,
		},
	}
	pod, err := cli.Core().Pods(opts.Namespace).Create(pod)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod. Namespace: %s, NameFmt: %s", opts.Namespace, opts.GenerateName)
	}
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		p, err := cli.Core().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return (p.Status.Phase == v1.PodRunning), nil
	})
	if err != nil {
		defer DeletePod(context.Background(), cli, pod)
		return nil, errors.Wrapf(err, "Pod did not transition to running state. Namespace:%s, Name:%s", pod.Namespace, pod.Name)
	}
	return pod, nil
}

// DeletePod deletes the specified pod
func DeletePod(ctx context.Context, cli kubernetes.Interface, pod *v1.Pod) error {
	if err := cli.Core().Pods(pod.Namespace).Delete(pod.Name, nil); err != nil {
		log.Errorf("DeletePod failed: %v", err)
	}
	return nil
}
