package kube

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	// Load the GCP plugin - required to authenticate against
	// GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func newClientConfig() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
}

// ConfigNamespace returns the namespace from config
func ConfigNamespace() (string, error) {
	cc := newClientConfig()
	ns, _, err := cc.Namespace()
	if err != nil {
		return "", errors.Wrap(err, "Could not get namespace from config")
	}
	return ns, nil
}

// LoadConfig returns a kubernetes client config based on global settings.
func LoadConfig() (*rest.Config, error) {
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	return newClientConfig().ClientConfig()
}

// NewClient returns a k8 client configured by the k10 environment.
func NewClient() kubernetes.Interface {
	config, err := LoadConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}
