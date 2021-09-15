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
	"io"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type Operation string

const (
	CreateOperation Operation = "create"
)

// KubectlOperation implements methods to perform kubectl operations
type KubectlOperation struct {
	factory   cmdutil.Factory
	specs     io.Reader
	namespace string
}

// NewKubectlOperations returns new KubectlOperations object
func NewKubectlOperations(specsString, namespace string) *KubectlOperation {
	return &KubectlOperation{
		factory:   cmdutil.NewFactory(genericclioptions.NewConfigFlags(false)),
		specs:     strings.NewReader(specsString),
		namespace: namespace,
	}
}

// Execute executes kubectl operation
func (k *KubectlOperation) Execute(op Operation) (*crv1alpha1.ObjectReference, error) {
	switch op {
	case CreateOperation:
		return k.create()
	default:
		return nil, errors.New("not implemented")
	}
}

func (k *KubectlOperation) create() (*crv1alpha1.ObjectReference, error) {
	// TODO: Create namespace if doesn't exists before creating an resource
	result := k.factory.NewBuilder().
		Unstructured().
		NamespaceParam(k.namespace).
		Stream(k.specs, "resource").
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
		namespace := k.namespace
		// Override namespace if the namespace is set in resource specs
		if info.Namespace != "" {
			namespace = info.Namespace
		}
		obj, err := resource.
			NewHelper(info.Client, info.Mapping).
			WithFieldManager("kanister-create").
			Create(namespace, true, info.Object)
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
