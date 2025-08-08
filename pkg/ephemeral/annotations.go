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

	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/validate"
)

func AnnotationsFromEnvVar(name string) ApplierSet {
	return ApplierSet{
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			if val, present := os.LookupEnv(name); present {
				var annotations map[string]string
				if err := json.Unmarshal([]byte(val), &annotations); err != nil {
					return err
				}

				if err := validate.ValidateAnnotations(annotations); err != nil {
					return err
				}

				if options.Annotations == nil {
					options.Annotations = make(map[string]string)
				}

				maps.Insert(options.Annotations, maps.All(annotations))
			}

			return nil
		}),
		Pod: ApplierFunc[corev1.Pod](func(options *corev1.Pod) error {
			if val, present := os.LookupEnv(name); present {
				var annotations map[string]string
				if err := json.Unmarshal([]byte(val), &annotations); err != nil {
					return err
				}

				if err := validate.ValidateAnnotations(annotations); err != nil {
					return err
				}

				if options.Annotations == nil {
					options.Annotations = make(map[string]string)
				}

				maps.Insert(options.Annotations, maps.All(annotations))
			}

			return nil
		}),
	}
}
