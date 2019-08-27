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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// FetchUnstructuredObject returns the referenced API object as a map[string]interface{}
func FetchUnstructuredObject(resource schema.GroupVersionResource, namespace, name string) (runtime.Unstructured, error) {
	cli, err := client()
	if err != nil {
		return nil, err
	}
	return fetchCR(cli, resource, namespace, name)
}

func fetchCR(cli dynamic.Interface, resource schema.GroupVersionResource, namespace, name string) (runtime.Unstructured, error) {
	return cli.Resource(resource).Namespace(namespace).Get(name, metav1.GetOptions{})
}

// ListUnstructuredObject returns the referenced API objects as a map[string]interface{}
func ListUnstructuredObject(resource schema.GroupVersionResource, namespace string) (runtime.Unstructured, error) {
	cli, err := client()
	if err != nil {
		return nil, err
	}
	return listCR(cli, resource, namespace)
}

func listCR(cli dynamic.Interface, resource schema.GroupVersionResource, namespace string) (runtime.Unstructured, error) {
	//return cli.Resource(resource).Namespace(namespace).List(metav1.ListOptions{})
	return cli.Resource(resource).List(metav1.ListOptions{})
}

func client() (dynamic.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}
