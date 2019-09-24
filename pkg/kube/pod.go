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
	"encoding/json"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
)

// PodOptions specifies options for `CreatePod`
type PodOptions struct {
	Namespace          string
	GenerateName       string
	Image              string
	Command            []string
	Volumes            map[string]string
	ServiceAccountName string
	PodOverride        v1.PodSpec
}

// CreatePod creates a pod with a single container based on the specified image
func CreatePod(ctx context.Context, cli kubernetes.Interface, opts *PodOptions) (pod *v1.Pod, err error) {
	volumeMounts, podVolumes := createVolumeSpecs(opts.Volumes)
	defaultSpecs := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:            "container",
				Image:           opts.Image,
				Command:         opts.Command,
				ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
				VolumeMounts:    volumeMounts,
			},
		},
		// RestartPolicy dictates when the containers of the pod should be restarted.
		// The possible values include Always, OnFailure and Never with Always being the default.
		// OnFailure policy will result in failed containers being restarted with an exponential back-off delay.
		RestartPolicy:      v1.RestartPolicyOnFailure,
		Volumes:            podVolumes,
		ServiceAccountName: opts.ServiceAccountName,
	}
	// Override default specs if podspecs are passed
	if !reflect.DeepEqual(opts.PodOverride, v1.PodSpec{}) {
		defaultSpecs, err = PodSpecOverride(ctx, defaultSpecs, opts.PodOverride)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create pod. Failed to override pod specs. Namespace: %s, NameFmt: %s", opts.Namespace, opts.GenerateName)
		}
	}
	// Make sure that container name exists after overriding
	for i, _ := range defaultSpecs.Containers {
		if len(defaultSpecs.Containers[i].Name) == 0 {
			defaultSpecs.Containers[i].Name = "container"
		}
	}

	pod = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: opts.GenerateName,
			Namespace:    opts.Namespace,
		},
		Spec: defaultSpecs,
	}
	pod, err = cli.CoreV1().Pods(opts.Namespace).Create(pod)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod. Namespace: %s, NameFmt: %s", opts.Namespace, opts.GenerateName)
	}
	return pod, nil
}

// DeletePod deletes the specified pod
func DeletePod(ctx context.Context, cli kubernetes.Interface, pod *v1.Pod) error {
	if err := cli.CoreV1().Pods(pod.Namespace).Delete(pod.Name, nil); err != nil {
		log.Errorf("DeletePod failed: %v", err)
	}
	return nil
}

func StreamPodLogs(ctx context.Context, cli kubernetes.Interface, namespace, name string) (io.ReadCloser, error) {
	plo := &v1.PodLogOptions{
		Follow: true,
	}
	return cli.CoreV1().Pods(namespace).GetLogs(name, plo).Stream()
}

// GetPodLogs fetches the logs from the given pod
func GetPodLogs(ctx context.Context, cli kubernetes.Interface, namespace, name string) (string, error) {
	reader, err := cli.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Stream()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// WaitForPodReady waits for a pod to exit the pending state
func WaitForPodReady(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		p, err := cli.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return p.Status.Phase != v1.PodPending && p.Status.Phase != "", nil
	})
	return errors.Wrapf(err, "Pod did not transition into running state. Namespace:%s, Name:%s", namespace, name)
}

// WaitForPodCompletion waits for a pod to reach a terminal state
func WaitForPodCompletion(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		p, err := cli.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		switch p.Status.Phase {
		case v1.PodRunning:
			for _, con := range p.Status.ContainerStatuses {
				if con.State.Terminated != nil {
					return false, errors.Errorf("Container %v is terminated, while Pod %v is Running", con.Name, name)
				}
			}
		case v1.PodFailed:
			return false, errors.Errorf("Pod %s failed", name)
		}
		return p.Status.Phase == v1.PodSucceeded, nil
	})
	return errors.Wrap(err, "Pod did not transition into complete state")
}

// PodSpecOverride override default pod Spec with the ones provided via specs
func PodSpecOverride(ctx context.Context, defaultSpecs, overrideSpecs v1.PodSpec) (v1.PodSpec, error) {
	containers := defaultSpecs.Containers
	// - Marshal override specs
	override, err := json.Marshal(overrideSpecs)
	if err != nil {
		return v1.PodSpec{}, err
	}
	// - Unmarshal override specs on default object so that it overrides only the fields that are present in override specs
	err = json.Unmarshal(override, &defaultSpecs)
	if err != nil {
		return v1.PodSpec{}, err
	}
	if defaultSpecs.Containers == nil {
		defaultSpecs.Containers = containers
	}
	return defaultSpecs, nil
}
