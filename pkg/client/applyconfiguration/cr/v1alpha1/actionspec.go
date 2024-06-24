/*
Copyright 2023 The Kanister Authors.

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

package v1alpha1

import (
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// ActionSpecApplyConfiguration represents an declarative configuration of the ActionSpec type for use
// with apply.
type ActionSpecApplyConfiguration struct {
	Name             *string                                      `json:"name,omitempty"`
	Object           *ObjectReferenceApplyConfiguration           `json:"object,omitempty"`
	Blueprint        *string                                      `json:"blueprint,omitempty"`
	Artifacts        map[string]ArtifactApplyConfiguration        `json:"artifacts,omitempty"`
	ConfigMaps       map[string]ObjectReferenceApplyConfiguration `json:"configMaps,omitempty"`
	Secrets          map[string]ObjectReferenceApplyConfiguration `json:"secrets,omitempty"`
	Profile          *ObjectReferenceApplyConfiguration           `json:"profile,omitempty"`
	RepositoryServer *ObjectReferenceApplyConfiguration           `json:"repositoryServer,omitempty"`
	PodOverride      *crv1alpha1.JSONMap                          `json:"podOverride,omitempty"`
	Options          map[string]string                            `json:"options,omitempty"`
	PreferredVersion *string                                      `json:"preferredVersion,omitempty"`
}

// ActionSpecApplyConfiguration constructs an declarative configuration of the ActionSpec type for use with
// apply.
func ActionSpec() *ActionSpecApplyConfiguration {
	return &ActionSpecApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithName(value string) *ActionSpecApplyConfiguration {
	b.Name = &value
	return b
}

// WithObject sets the Object field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Object field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithObject(value *ObjectReferenceApplyConfiguration) *ActionSpecApplyConfiguration {
	b.Object = value
	return b
}

// WithBlueprint sets the Blueprint field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Blueprint field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithBlueprint(value string) *ActionSpecApplyConfiguration {
	b.Blueprint = &value
	return b
}

// WithArtifacts puts the entries into the Artifacts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Artifacts field,
// overwriting an existing map entries in Artifacts field with the same key.
func (b *ActionSpecApplyConfiguration) WithArtifacts(entries map[string]ArtifactApplyConfiguration) *ActionSpecApplyConfiguration {
	if b.Artifacts == nil && len(entries) > 0 {
		b.Artifacts = make(map[string]ArtifactApplyConfiguration, len(entries))
	}
	for k, v := range entries {
		b.Artifacts[k] = v
	}
	return b
}

// WithConfigMaps puts the entries into the ConfigMaps field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the ConfigMaps field,
// overwriting an existing map entries in ConfigMaps field with the same key.
func (b *ActionSpecApplyConfiguration) WithConfigMaps(entries map[string]ObjectReferenceApplyConfiguration) *ActionSpecApplyConfiguration {
	if b.ConfigMaps == nil && len(entries) > 0 {
		b.ConfigMaps = make(map[string]ObjectReferenceApplyConfiguration, len(entries))
	}
	for k, v := range entries {
		b.ConfigMaps[k] = v
	}
	return b
}

// WithSecrets puts the entries into the Secrets field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Secrets field,
// overwriting an existing map entries in Secrets field with the same key.
func (b *ActionSpecApplyConfiguration) WithSecrets(entries map[string]ObjectReferenceApplyConfiguration) *ActionSpecApplyConfiguration {
	if b.Secrets == nil && len(entries) > 0 {
		b.Secrets = make(map[string]ObjectReferenceApplyConfiguration, len(entries))
	}
	for k, v := range entries {
		b.Secrets[k] = v
	}
	return b
}

// WithProfile sets the Profile field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Profile field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithProfile(value *ObjectReferenceApplyConfiguration) *ActionSpecApplyConfiguration {
	b.Profile = value
	return b
}

// WithRepositoryServer sets the RepositoryServer field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RepositoryServer field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithRepositoryServer(value *ObjectReferenceApplyConfiguration) *ActionSpecApplyConfiguration {
	b.RepositoryServer = value
	return b
}

// WithPodOverride sets the PodOverride field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodOverride field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithPodOverride(value crv1alpha1.JSONMap) *ActionSpecApplyConfiguration {
	b.PodOverride = &value
	return b
}

// WithOptions puts the entries into the Options field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Options field,
// overwriting an existing map entries in Options field with the same key.
func (b *ActionSpecApplyConfiguration) WithOptions(entries map[string]string) *ActionSpecApplyConfiguration {
	if b.Options == nil && len(entries) > 0 {
		b.Options = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.Options[k] = v
	}
	return b
}

// WithPreferredVersion sets the PreferredVersion field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PreferredVersion field is set to the value of the last call.
func (b *ActionSpecApplyConfiguration) WithPreferredVersion(value string) *ActionSpecApplyConfiguration {
	b.PreferredVersion = &value
	return b
}
