/*
Copyright 2025 by contributors to the Kanister project.

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

package ephemeral

import (
	"encoding/json"
	"maps"
	"os"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/validate"
	corev1 "k8s.io/api/core/v1"
)

func LabelsFromEnvVar(name string) ApplierSet {
	return ApplierSet{
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			if val, present := os.LookupEnv(name); present {
				var labels map[string]string
				if err := json.Unmarshal([]byte(val), &labels); err != nil {
					return err
				}

				if err := validate.ValidateLabels(labels); err != nil {
					return err
				}

				if options.Labels == nil {
					options.Labels = make(map[string]string)
				}

				maps.Insert(options.Labels, maps.All(labels))
			}

			return nil
		}),
		Pod: ApplierFunc[corev1.Pod](func(options *corev1.Pod) error {
			if val, present := os.LookupEnv(name); present {
				var labels map[string]string
				if err := json.Unmarshal([]byte(val), &labels); err != nil {
					return err
				}

				if err := validate.ValidateLabels(labels); err != nil {
					return err
				}

				if options.Labels == nil {
					options.Labels = make(map[string]string)
				}

				maps.Insert(options.Labels, maps.All(labels))
			}

			return nil
		}),
	}
}

// StaticLabels creates an ApplierSet that applies the passed labels
// to PodOptions and Pods.
func StaticLabels(labels map[string]string) ApplierSet {
	return ApplierSet{
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			if options.Labels == nil {
				options.Labels = make(map[string]string, len(labels))
			}
			maps.Copy(options.Labels, labels)
			return nil
		}),
		Pod: ApplierFunc[corev1.Pod](func(options *corev1.Pod) error {
			if options.Labels == nil {
				options.Labels = make(map[string]string, len(labels))
			}
			maps.Copy(options.Labels, labels)
			return nil
		}),
	}
}

// StaticLabelsOSEnvVar creates an ApplierSet that applies a fixed set of labels
// to PodOptions and Pods if the specified environment variables are set.
func StaticLabelsOSEnvVar(labels map[string]string, envVars ...string) ApplierSet {
	return ApplierSet{
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			for _, envVar := range envVars {
				if _, ok := os.LookupEnv(envVar); !ok {
					return nil
				}
			}
			if options.Labels == nil {
				options.Labels = make(map[string]string, len(labels))
			}
			maps.Copy(options.Labels, labels)
			return nil
		}),
		Pod: ApplierFunc[corev1.Pod](func(options *corev1.Pod) error {
			for _, envVar := range envVars {
				if _, ok := os.LookupEnv(envVar); !ok {
					return nil
				}
			}
			if options.Labels == nil {
				options.Labels = make(map[string]string, len(labels))
			}
			maps.Copy(options.Labels, labels)
			return nil
		}),
	}
}
