package kube

import (
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nsFile        = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	podNSEnvVar   = "POD_NAMESPACE"
	podSAEnvVar   = "POD_SERVICE_ACCOUNT"
	podNameEnvVar = "POD_NAME"
)

//GetControllerNamespace returns controller namespace
func GetControllerNamespace() (string, error) {
	if ns, ok := os.LookupEnv(podNSEnvVar); ok {
		return ns, nil
	}

	ns, err := ioutil.ReadFile(nsFile)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to read namespace form k8s mounted file")
	}

	return string(ns), nil
}

//GetControllerServiceAccount returns controller ServiceAccount
func GetControllerServiceAccount(k8sclient kubernetes.Interface) (string, error) {
	if ns, ok := os.LookupEnv(podSAEnvVar); ok {
		return ns, nil
	}
	ns, err := GetControllerNamespace()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Controller namespace")
	}

	podName, err := GetControllerPodName()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Controller pod name")
	}

	pod, err := k8sclient.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Controller pod object from k8s")
	}
	return pod.Spec.ServiceAccountName, nil
}

//GetControllerPodName returns controller pod name
func GetControllerPodName() (string, error) {
	if podName, ok := os.LookupEnv(podNameEnvVar); ok {
		return podName, nil
	}
	podName, err := os.Hostname()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get pod name from Hostname")
	}

	return podName, nil
}
