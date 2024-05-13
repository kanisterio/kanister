// Copyright 2024 The Kanister Authors.
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

package ephemeral

import (
	"os"

	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/kube"
)

// OSEnvVar creates an ApplierSet to set an environment variable if it's present
// in the current environment.
func OSEnvVar(name string) ApplierSet {
	return ApplierSet{
		Container: ApplierFunc[corev1.Container](func(container *corev1.Container) {
			if val, present := os.LookupEnv(name); present {
				container.Env = append(
					container.Env,
					corev1.EnvVar{
						Name:  name,
						Value: val,
					},
				)
			}
		}),
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) {
			if val, present := os.LookupEnv(name); present {
				options.EnvironmentVariables = append(
					options.EnvironmentVariables,
					corev1.EnvVar{
						Name:  name,
						Value: val,
					},
				)
			}
		}),
	}
}

// StaticEnvVar creates an ApplierSet to set a static environment variable.
func StaticEnvVar(name, value string) ApplierSet {
	return ApplierSet{
		Container: ApplierFunc[corev1.Container](func(container *corev1.Container) {
			container.Env = append(
				container.Env,
				corev1.EnvVar{
					Name:  name,
					Value: value,
				},
			)
		}),
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) {
			options.EnvironmentVariables = append(
				options.EnvironmentVariables,
				corev1.EnvVar{
					Name:  name,
					Value: value,
				},
			)
		}),
	}
}
