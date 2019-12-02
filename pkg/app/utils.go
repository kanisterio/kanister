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

package app

import (
	"context"
	"fmt"

	"github.com/kanisterio/kanister/pkg/kube"
	"k8s.io/client-go/kubernetes"
)

func getPodContainerFromDeployment(ctx context.Context, cli kubernetes.Interface, namespace, deployName string) (podName string, containerName string, err error) {
	pod, _, err := kube.DeploymentPods(ctx, cli, namespace, deployName)
	if err != nil {
		return podName, containerName, err
	}
	if len(pod) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find ready pod for deployment %s/%s", namespace, deployName)
	}
	podName = pod[0].GetName()
	container, err := kube.PodContainers(ctx, cli, namespace, podName)
	if err != nil {
		return podName, containerName, err
	}
	if len(container) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find containers in pod %s/%s", namespace, podName)
	}
	return podName, container[0].Name, nil
}

func getPodContainerFromStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace, ssName string) (podName string, containerName string, err error) {
	pod, _, err := kube.StatefulSetPods(ctx, cli, namespace, ssName)
	if err != nil {
		return podName, containerName, err
	}
	if len(pod) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find ready pod for statefulset %s/%s", namespace, ssName)
	}
	podName = pod[0].GetName()
	container, err := kube.PodContainers(ctx, cli, namespace, podName)
	if err != nil {
		return podName, containerName, err
	}
	if len(container) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find containers in pod %s/%s", namespace, podName)
	}
	return podName, container[0].Name, nil
}
