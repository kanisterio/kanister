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
	"os"

	"github.com/kanisterio/errkit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	nsFile        = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	PodNSEnvVar   = "POD_NAMESPACE"
	PodSAEnvVar   = "POD_SERVICE_ACCOUNT"
	podNameEnvVar = "POD_NAME"
)

// GetControllerNamespace returns controller namespace
func GetControllerNamespace() (string, error) {
	if ns, ok := os.LookupEnv(PodNSEnvVar); ok {
		return ns, nil
	}

	ns, err := os.ReadFile(nsFile)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to read namespace form k8s mounted file")
	}

	return string(ns), nil
}

// GetControllerServiceAccount returns controller ServiceAccount
func GetControllerServiceAccount(k8sclient kubernetes.Interface) (string, error) {
	if ns, ok := os.LookupEnv(PodSAEnvVar); ok {
		return ns, nil
	}
	ns, err := GetControllerNamespace()
	if err != nil {
		return "", errkit.Wrap(err, "Failed to get Controller namespace")
	}

	podName, err := GetControllerPodName()
	if err != nil {
		return "", errkit.Wrap(err, "Failed to get Controller pod name")
	}

	pod, err := k8sclient.CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", errkit.Wrap(err, "Failed to get Controller pod object from k8s")
	}
	return pod.Spec.ServiceAccountName, nil
}

// GetControllerPodName returns controller pod name
func GetControllerPodName() (string, error) {
	if podName, ok := os.LookupEnv(podNameEnvVar); ok {
		return podName, nil
	}
	podName, err := os.Hostname()
	if err != nil {
		return "", errkit.Wrap(err, "Failed to get pod name from Hostname")
	}

	return podName, nil
}
