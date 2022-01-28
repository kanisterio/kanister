// Copyright 2021 The Kanister Authors.
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
	"io"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/poll"
)

// Operation represents kubectl operation
type Operation string

const (
	// CreateOperation represents kubectl create operation
	CreateOperation Operation = "create"
	// DeleteOperation represents kubectl delete operation
	DeleteOperation Operation = "delete"
)

// KubectlOperation implements methods to perform kubectl operations
type KubectlOperation struct {
	dynCli  dynamic.Interface
	factory cmdutil.Factory
}

// NewKubectlOperations returns new KubectlOperations object
func NewKubectlOperations(dynCli dynamic.Interface) *KubectlOperation {
	return &KubectlOperation{
		dynCli:  dynCli,
		factory: cmdutil.NewFactory(genericclioptions.NewConfigFlags(false)),
	}
}

// Create k8s resource from spec manifest
func (k *KubectlOperation) Create(spec io.Reader, namespace string) (*crv1alpha1.ObjectReference, error) {
	// TODO: Create namespace if doesn't exist before creating an resource
	result := k.factory.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).
		Stream(spec, "resource").
		Flatten().
		Do()
	err := result.Err()
	if err != nil {
		return nil, err
	}
	var objRef *crv1alpha1.ObjectReference
	err = result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		// Override namespace if the namespace is set in resource spec
		if info.Namespace != "" {
			namespace = info.Namespace
		}
		obj, err := resource.
			NewHelper(info.Client, info.Mapping).
			WithFieldManager("kanister-create").
			Create(namespace, true, info.Object)
		if err != nil {
			return err
		}
		// convert the runtime.Object to unstructured.Unstructured
		unstructObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}
		us := unstructured.Unstructured{Object: unstructObj}
		objRef = &crv1alpha1.ObjectReference{
			APIVersion: info.Mapping.Resource.Version,
			Group:      info.Mapping.Resource.Group,
			Resource:   info.Mapping.Resource.Resource,
			Name:       us.GetName(),
			Namespace:  us.GetNamespace(),
		}
		return err
	})
	return objRef, err
}

// Delete k8s resource referred by objectReference. Waits for the resource to be deleted
func (k *KubectlOperation) Delete(ctx context.Context, objRef crv1alpha1.ObjectReference, namespace string) (*crv1alpha1.ObjectReference, error) {
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	if objRef.Namespace != "" {
		namespace = objRef.Namespace
	}
	err := k.dynCli.Resource(schema.GroupVersionResource{Group: objRef.Group, Version: objRef.APIVersion, Resource: objRef.Resource}).Namespace(namespace).Delete(ctx, objRef.Name, metav1.DeleteOptions{})
	if err != nil {
		return &objRef, err
	}
	return waitForResourceDeletion(ctx, k, objRef, namespace)
}

// waitForResourceDeletion repeatedly checks for NotFound error after fetching the resource
func waitForResourceDeletion(ctx context.Context, k *KubectlOperation, objRef crv1alpha1.ObjectReference, namespace string) (*crv1alpha1.ObjectReference, error) {
	err := poll.Wait(ctx, func(context.Context) (done bool, err error) {
		_, err = k.dynCli.Resource(schema.GroupVersionResource{Group: objRef.Group, Version: objRef.APIVersion, Resource: objRef.Resource}).Namespace(namespace).Get(ctx, objRef.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	return &objRef, err
}
