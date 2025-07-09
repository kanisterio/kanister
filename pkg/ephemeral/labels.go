package ephemeral

import (
	"encoding/json"
	"maps"
	"os"

	"github.com/kanisterio/kanister/pkg/kube"
	corev1 "k8s.io/api/core/v1"
)

func LabelsFromEnvVar(name string) ApplierSet {
	return ApplierSet{
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			if val, present := os.LookupEnv(name); present {
				var labels map[string]string
				if err := json.Unmarshal([]byte(val), labels); err != nil {
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
				if err := json.Unmarshal([]byte(val), labels); err != nil {
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
