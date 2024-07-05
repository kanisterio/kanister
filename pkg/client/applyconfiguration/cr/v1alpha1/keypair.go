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

// KeyPairApplyConfiguration represents an declarative configuration of the KeyPair type for use
// with apply.
type KeyPairApplyConfiguration struct {
	IDField     *string                            `json:"idField,omitempty"`
	SecretField *string                            `json:"secretField,omitempty"`
	Secret      *ObjectReferenceApplyConfiguration `json:"secret,omitempty"`
}

// KeyPairApplyConfiguration constructs an declarative configuration of the KeyPair type for use with
// apply.
func KeyPair() *KeyPairApplyConfiguration {
	return &KeyPairApplyConfiguration{}
}

// WithIDField sets the IDField field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IDField field is set to the value of the last call.
func (b *KeyPairApplyConfiguration) WithIDField(value string) *KeyPairApplyConfiguration {
	b.IDField = &value
	return b
}

// WithSecretField sets the SecretField field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SecretField field is set to the value of the last call.
func (b *KeyPairApplyConfiguration) WithSecretField(value string) *KeyPairApplyConfiguration {
	b.SecretField = &value
	return b
}

// WithSecret sets the Secret field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Secret field is set to the value of the last call.
func (b *KeyPairApplyConfiguration) WithSecret(value *ObjectReferenceApplyConfiguration) *KeyPairApplyConfiguration {
	b.Secret = value
	return b
}
