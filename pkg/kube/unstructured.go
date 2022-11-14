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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// FetchUnstructuredObject returns the referenced API object as a map[string]interface{}
func FetchUnstructuredObject(ctx context.Context, resource schema.GroupVersionResource, namespace, name string) (runtime.Unstructured, error) {
	cli, err := client()
	if err != nil {
		return nil, err
	}
	return FetchUnstructuredObjectWithCli(ctx, cli, resource, namespace, name)
}

// FetchUnstructuredObjectWithCli returns the referenced API object as a map[string]interface{} using the specified CLI
// TODO: deprecate `FetchUnstructuredObject`
func FetchUnstructuredObjectWithCli(ctx context.Context, cli dynamic.Interface, resource schema.GroupVersionResource, namespace, name string) (runtime.Unstructured, error) {
	if namespace == "" {
		_, _ = cli.Resource(resource).Get(ctx, name, metav1.GetOptions{})
	}
	return cli.Resource(resource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListUnstructuredObject returns the referenced API objects as a map[string]interface{}
func ListUnstructuredObject(resource schema.GroupVersionResource, namespace string) (runtime.Unstructured, error) {
	cli, err := client()
	if err != nil {
		return nil, err
	}
	return ListUnstructuredObjectWithCli(cli, resource, namespace)
}

// ListUnstructuredObjectWithCli returns the referenced API objects as a map[string]interface{} using the specified CLI
// TODO: deprecate `ListUnstructuredObject`
func ListUnstructuredObjectWithCli(cli dynamic.Interface, resource schema.GroupVersionResource, namespace string) (runtime.Unstructured, error) {
	ctx := context.Background()
	if namespace == "" {
		return cli.Resource(resource).List(ctx, metav1.ListOptions{})
	}
	return cli.Resource(resource).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

func client() (dynamic.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}
