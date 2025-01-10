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

package v1alpha1

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// PhaseApplyConfiguration represents a declarative configuration of the Phase type for use
// with apply.
type PhaseApplyConfiguration struct {
	Name     *string                          `json:"name,omitempty"`
	State    *v1alpha1.State                  `json:"state,omitempty"`
	Output   map[string]interface{}           `json:"output,omitempty"`
	Progress *PhaseProgressApplyConfiguration `json:"progress,omitempty"`
}

// PhaseApplyConfiguration constructs a declarative configuration of the Phase type for use with
// apply.
func Phase() *PhaseApplyConfiguration {
	return &PhaseApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *PhaseApplyConfiguration) WithName(value string) *PhaseApplyConfiguration {
	b.Name = &value
	return b
}

// WithState sets the State field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the State field is set to the value of the last call.
func (b *PhaseApplyConfiguration) WithState(value v1alpha1.State) *PhaseApplyConfiguration {
	b.State = &value
	return b
}

// WithOutput puts the entries into the Output field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Output field,
// overwriting an existing map entries in Output field with the same key.
func (b *PhaseApplyConfiguration) WithOutput(entries map[string]interface{}) *PhaseApplyConfiguration {
	if b.Output == nil && len(entries) > 0 {
		b.Output = make(map[string]interface{}, len(entries))
	}
	for k, v := range entries {
		b.Output[k] = v
	}
	return b
}

// WithProgress sets the Progress field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Progress field is set to the value of the last call.
func (b *PhaseApplyConfiguration) WithProgress(value *PhaseProgressApplyConfiguration) *PhaseApplyConfiguration {
	b.Progress = value
	return b
}
