/*
Copyright 2019 The Kanister Authors.

Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kanisterio/kanister/pkg/consts"
	customresource "github.com/kanisterio/kanister/pkg/customresource"
)

// These variables are exported to help hook into this package's schemes.
var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// ActionSetResource is a CRD for actionsets.
var ActionSetResource = customresource.CustomResource{
	Name:    consts.ActionSetResourceName,
	Plural:  consts.ActionSetResourceNamePlural,
	Group:   ResourceGroup,
	Version: SchemeVersion,
	Scope:   apiextensionsv1.NamespaceScoped,
	Kind:    reflect.TypeOf(ActionSet{}).Name(),
}

// BlueprintResource is a CRD for blueprints.
var BlueprintResource = customresource.CustomResource{
	Name:    consts.BlueprintResourceName,
	Plural:  consts.BlueprintResourceNamePlural,
	Group:   ResourceGroup,
	Version: SchemeVersion,
	Scope:   apiextensionsv1.NamespaceScoped,
	Kind:    reflect.TypeOf(Blueprint{}).Name(),
}

// ProfileResource is a CRD for blueprints.
var ProfileResource = customresource.CustomResource{
	Name:    consts.ProfileResourceName,
	Plural:  consts.ProfileResourceNamePlural,
	Group:   ResourceGroup,
	Version: SchemeVersion,
	Scope:   apiextensionsv1.NamespaceScoped,
	Kind:    reflect.TypeOf(Profile{}).Name(),
}

// RepositoryServerResource is a CRD for blueprints.
var RepositoryServerResource = customresource.CustomResource{
	Name:    consts.RepositoryServerResourceName,
	Plural:  consts.RepositoryServerResourceNamePlural,
	Group:   ResourceGroup,
	Version: SchemeVersion,
	Scope:   apiextensionsv1.NamespaceScoped,
	Kind:    reflect.TypeOf(RepositoryServer{}).Name(),
}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ActionSet{},
		&ActionSetList{},
		&Blueprint{},
		&BlueprintList{},
		&Profile{},
		&ProfileList{},
		&RepositoryServer{},
		&RepositoryServerList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
