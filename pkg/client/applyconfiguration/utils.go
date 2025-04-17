/*
Copyright 2024 The Kanister Authors.

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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

// Package applyconfiguration provides utility functions and types for working with
// apply configurations in the Kanister project.
package applyconfiguration

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/client/applyconfiguration/cr/v1alpha1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=cr.kanister.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithKind("ActionProgress"):
		return &crv1alpha1.ActionProgressApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ActionSet"):
		return &crv1alpha1.ActionSetApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ActionSetSpec"):
		return &crv1alpha1.ActionSetSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ActionSetStatus"):
		return &crv1alpha1.ActionSetStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ActionSpec"):
		return &crv1alpha1.ActionSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ActionStatus"):
		return &crv1alpha1.ActionStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Artifact"):
		return &crv1alpha1.ArtifactApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Blueprint"):
		return &crv1alpha1.BlueprintApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("BlueprintAction"):
		return &crv1alpha1.BlueprintActionApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("BlueprintPhase"):
		return &crv1alpha1.BlueprintPhaseApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CacheSizeSettings"):
		return &crv1alpha1.CacheSizeSettingsApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Configuration"):
		return &crv1alpha1.ConfigurationApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Credential"):
		return &crv1alpha1.CredentialApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Error"):
		return &crv1alpha1.ErrorApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("KeyPair"):
		return &crv1alpha1.KeyPairApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("KopiaServerSecret"):
		return &crv1alpha1.KopiaServerSecretApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("KopiaServerSecretRef"):
		return &crv1alpha1.KopiaServerSecretRefApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Location"):
		return &crv1alpha1.LocationApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ObjectReference"):
		return &crv1alpha1.ObjectReferenceApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Phase"):
		return &crv1alpha1.PhaseApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("PhaseProgress"):
		return &crv1alpha1.PhaseProgressApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Profile"):
		return &crv1alpha1.ProfileApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Repository"):
		return &crv1alpha1.RepositoryApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RepositoryServer"):
		return &crv1alpha1.RepositoryServerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RepositoryServerSpec"):
		return &crv1alpha1.RepositoryServerSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RepositoryServerStatus"):
		return &crv1alpha1.RepositoryServerStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Server"):
		return &crv1alpha1.ServerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ServerInfo"):
		return &crv1alpha1.ServerInfoApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Storage"):
		return &crv1alpha1.StorageApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("UserAccess"):
		return &crv1alpha1.UserAccessApplyConfiguration{}

	}
	return nil
}
