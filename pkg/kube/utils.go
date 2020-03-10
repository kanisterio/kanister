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
	"fmt"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// FDZoneLabelName is a known k8s label. used to specify volume zone
	FDZoneLabelName = "failure-domain.beta.kubernetes.io/zone"
	// TopologyZoneLabelName is a known k8s label. used to specify volume zone for kubernetes 1.17 onwards
	TopologyZoneLabelName = "topology.kubernetes.io/zone"
	// FDRegionLabelName is a known k8s label
	FDRegionLabelName = "failure-domain.beta.kubernetes.io/region"
	// TopologyRegionLabelName is a known k8s label. used to specify volume region for kubernetes 1.17 onwards
	TopologyRegionLabelName = "topology.kubernetes.io/region"
)

// GetPodContainerFromDeployment returns a pod and container running the deployment
func GetPodContainerFromDeployment(ctx context.Context, cli kubernetes.Interface, namespace, deployName string) (podName string, containerName string, err error) {
	pod, _, err := DeploymentPods(ctx, cli, namespace, deployName)
	if err != nil {
		return podName, containerName, err
	}
	if len(pod) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find ready pod for deployment %s/%s", namespace, deployName)
	}
	podName = pod[0].GetName()
	container, err := PodContainers(ctx, cli, namespace, podName)
	if err != nil {
		return podName, containerName, err
	}
	if len(container) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find containers in pod %s/%s", namespace, podName)
	}
	return podName, container[0].Name, nil
}

// GetPodContainerFromDeploymentConfig returns a pod and container that is running from the provided deployment config
func GetPodContainerFromDeploymentConfig(ctx context.Context, osCli osversioned.Interface, cli kubernetes.Interface, namespace, deployConfigName string) (podName, containerName string, err error) {
	pods, _, err := DeploymentConfigPods(ctx, osCli, cli, namespace, deployConfigName)
	if err != nil {
		return podName, containerName, err
	}
	if len(pods) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find ready pod for deploymentconfig %s/%s", namespace, deployConfigName)
	}

	podName = pods[0].GetName()
	containers, err := PodContainers(ctx, cli, namespace, podName)
	if err != nil {
		return podName, containerName, err
	}

	if len(containers) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find containers in pod %s/%s", namespace, podName)
	}
	return podName, containers[0].Name, nil
}

// GetPodContainerFromStatefulSet returns a pod and container running the stateful set
func GetPodContainerFromStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace, ssName string) (podName string, containerName string, err error) {
	pod, _, err := StatefulSetPods(ctx, cli, namespace, ssName)
	if err != nil {
		return podName, containerName, err
	}
	if len(pod) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find ready pod for statefulset %s/%s", namespace, ssName)
	}
	podName = pod[0].GetName()
	container, err := PodContainers(ctx, cli, namespace, podName)
	if err != nil {
		return podName, containerName, err
	}
	if len(container) == 0 {
		return podName, containerName, fmt.Errorf("Unable to find containers in pod %s/%s", namespace, podName)
	}
	return podName, container[0].Name, nil
}

func GetZoneFromNode(node v1.Node) string {
	return GetZoneFromLabels(node.Labels)
}

func GetZoneFromPV(pv v1.PersistentVolume) string {
	return GetZoneFromLabels(pv.Labels)
}

func GetZoneFromLabels(labels map[string]string) string {
	var zone string
	if v, ok := labels[FDZoneLabelName]; ok {
		zone = v
	} else if v, ok := labels[TopologyZoneLabelName]; ok {
		zone = v
	}
	return zone
}

func GetRegionFromNode(node v1.Node) string {
	return GetRegionFromLabels(node.Labels)
}

func GetRegionFromPV(pv v1.PersistentVolume) string {
	return GetRegionFromLabels(pv.Labels)
}

func GetRegionFromLabels(labels map[string]string) string {
	var region string
	if v, ok := labels[FDRegionLabelName]; ok {
		region = v
	} else if v, ok := labels[TopologyRegionLabelName]; ok {
		region = v
	}
	return region
}
