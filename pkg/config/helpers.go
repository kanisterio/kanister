// Package config provides utilities for managing Kubernetes cluster configurations.
package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// ClusterNameEnvName is a env var name to get cluster name
	ClusterNameEnvName   = "CLUSTER_NAME"
	defaultNamespaceName = "default"
)

// clusterNameLookupTimeout bounds the API call used to get the "default"
// namespace UUID, so that an unresponsive API server can't block the caller.
var clusterNameLookupTimeout = 10 * time.Second

// GetClusterName checks CLUSTER_NAME
// if empty gets "default" namespace UUID
func GetClusterName(cli kubernetes.Interface) (string, error) {
	if clusterName, ok := os.LookupEnv(ClusterNameEnvName); ok {
		return clusterName, nil
	}
	if cli == nil {
		tmpcli, err := newKubeClient()
		if err != nil {
			return "", err
		}
		cli = tmpcli
	}

	ctx, cancel := context.WithTimeout(context.Background(), clusterNameLookupTimeout)
	defer cancel()
	ns, err := cli.CoreV1().Namespaces().Get(ctx, defaultNamespaceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(ns.GetUID()), nil
}

// GetEnvOrSkip test helper to skip test if env var not presented
func GetEnvOrSkip(c *check.C, varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		reason := fmt.Sprintf("Test %s requires the environment variable '%s'", c.TestName(), varName)
		c.Skip(reason)
	}
	return v
}

// due to cycle imports issues pks/kube can not be used
func newKubeClient() (kubernetes.Interface, error) {
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	c, err := cc.ClientConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
