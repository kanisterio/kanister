package kube

import (
	snapshot "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes" // Load the GCP plugin - required to authenticate against
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

// NewClient returns a k8 client configured by the kanister environment.
func NewClient() (kubernetes.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// NewClientSnapshot returns a VolumeSnapshot client configured by the Kanister environment.
func NewSnapshotClient() (snapshot.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := snapshot.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
