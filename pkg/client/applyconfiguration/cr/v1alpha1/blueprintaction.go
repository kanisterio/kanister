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

// BlueprintActionApplyConfiguration represents an declarative configuration of the BlueprintAction type for use
// with apply.
type BlueprintActionApplyConfiguration struct {
	Name               *string                               `json:"name,omitempty"`
	Kind               *string                               `json:"kind,omitempty"`
	ConfigMapNames     []string                              `json:"configMapNames,omitempty"`
	SecretNames        []string                              `json:"secretNames,omitempty"`
	InputArtifactNames []string                              `json:"inputArtifactNames,omitempty"`
	OutputArtifacts    map[string]ArtifactApplyConfiguration `json:"outputArtifacts,omitempty"`
	Phases             []BlueprintPhaseApplyConfiguration    `json:"phases,omitempty"`
	DeferPhase         *BlueprintPhaseApplyConfiguration     `json:"deferPhase,omitempty"`
}

// BlueprintActionApplyConfiguration constructs an declarative configuration of the BlueprintAction type for use with
// apply.
func BlueprintAction() *BlueprintActionApplyConfiguration {
	return &BlueprintActionApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *BlueprintActionApplyConfiguration) WithName(value string) *BlueprintActionApplyConfiguration {
	b.Name = &value
	return b
}

// WithKind sets the Kind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kind field is set to the value of the last call.
func (b *BlueprintActionApplyConfiguration) WithKind(value string) *BlueprintActionApplyConfiguration {
	b.Kind = &value
	return b
}

// WithConfigMapNames adds the given value to the ConfigMapNames field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ConfigMapNames field.
func (b *BlueprintActionApplyConfiguration) WithConfigMapNames(values ...string) *BlueprintActionApplyConfiguration {
	for i := range values {
		b.ConfigMapNames = append(b.ConfigMapNames, values[i])
	}
	return b
}

// WithSecretNames adds the given value to the SecretNames field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the SecretNames field.
func (b *BlueprintActionApplyConfiguration) WithSecretNames(values ...string) *BlueprintActionApplyConfiguration {
	for i := range values {
		b.SecretNames = append(b.SecretNames, values[i])
	}
	return b
}

// WithInputArtifactNames adds the given value to the InputArtifactNames field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the InputArtifactNames field.
func (b *BlueprintActionApplyConfiguration) WithInputArtifactNames(values ...string) *BlueprintActionApplyConfiguration {
	for i := range values {
		b.InputArtifactNames = append(b.InputArtifactNames, values[i])
	}
	return b
}

// WithOutputArtifacts puts the entries into the OutputArtifacts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the OutputArtifacts field,
// overwriting an existing map entries in OutputArtifacts field with the same key.
func (b *BlueprintActionApplyConfiguration) WithOutputArtifacts(entries map[string]ArtifactApplyConfiguration) *BlueprintActionApplyConfiguration {
	if b.OutputArtifacts == nil && len(entries) > 0 {
		b.OutputArtifacts = make(map[string]ArtifactApplyConfiguration, len(entries))
	}
	for k, v := range entries {
		b.OutputArtifacts[k] = v
	}
	return b
}

// WithPhases adds the given value to the Phases field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Phases field.
func (b *BlueprintActionApplyConfiguration) WithPhases(values ...*BlueprintPhaseApplyConfiguration) *BlueprintActionApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithPhases")
		}
		b.Phases = append(b.Phases, *values[i])
	}
	return b
}

// WithDeferPhase sets the DeferPhase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeferPhase field is set to the value of the last call.
func (b *BlueprintActionApplyConfiguration) WithDeferPhase(value *BlueprintPhaseApplyConfiguration) *BlueprintActionApplyConfiguration {
	b.DeferPhase = value
	return b
}
