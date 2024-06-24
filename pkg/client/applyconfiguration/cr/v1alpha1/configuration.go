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

// ConfigurationApplyConfiguration represents an declarative configuration of the Configuration type for use
// with apply.
type ConfigurationApplyConfiguration struct {
	CacheDirectory *string `json:"cacheDirectory,omitempty"`
	LogDirectory   *string `json:"logDirectory,omitempty"`
	ConfigFilePath *string `json:"configFilePath,omitempty"`
}

// ConfigurationApplyConfiguration constructs an declarative configuration of the Configuration type for use with
// apply.
func Configuration() *ConfigurationApplyConfiguration {
	return &ConfigurationApplyConfiguration{}
}

// WithCacheDirectory sets the CacheDirectory field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CacheDirectory field is set to the value of the last call.
func (b *ConfigurationApplyConfiguration) WithCacheDirectory(value string) *ConfigurationApplyConfiguration {
	b.CacheDirectory = &value
	return b
}

// WithLogDirectory sets the LogDirectory field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LogDirectory field is set to the value of the last call.
func (b *ConfigurationApplyConfiguration) WithLogDirectory(value string) *ConfigurationApplyConfiguration {
	b.LogDirectory = &value
	return b
}

// WithConfigFilePath sets the ConfigFilePath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ConfigFilePath field is set to the value of the last call.
func (b *ConfigurationApplyConfiguration) WithConfigFilePath(value string) *ConfigurationApplyConfiguration {
	b.ConfigFilePath = &value
	return b
}
