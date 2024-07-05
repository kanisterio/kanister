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
	"github.com/kanisterio/errkit"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
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
		return "", errkit.Wrap(err, "Could not get namespace from config")
	}
	return ns, nil
}

// LoadConfig returns a kubernetes client config based on global settings.
func LoadConfig() (*rest.Config, error) {
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

// NewDynamicClient returns a Dynamic client configured by the Kanister environment.
func NewDynamicClient() (dynamic.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// NewCRDClient returns a Dynamic client configured by the Kanister environment.
func NewCRDClient() (crdclient.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	return crdclient.NewForConfig(config)
}
