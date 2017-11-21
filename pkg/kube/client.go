package kube

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	// Load the GCP plugin - required to authenticate against
	// GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// LoadConfig returns a kubernetes client config based on global settings.
func LoadConfig() (config *rest.Config, err error) {
	homeConfig := filepath.Join(os.Getenv("HOME"), ".kube/config")
	potentialConfigs := []string{
		"", // An empty config path is used when we're in-cluster.
		homeConfig,
	}
	for _, pc := range potentialConfigs {
		config, err = clientcmd.BuildConfigFromFlags("", pc)
		if err == nil {
			return
		}
	}
	return // result of last clientcmd.BuildConfigFromFlags
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
