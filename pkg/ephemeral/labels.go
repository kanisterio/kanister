package ephemeral

import (
	"encoding/json"
	"maps"
	"os"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/utils"
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

				if err := utils.ValidateLabels(labels); err != nil {
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

				if err := utils.ValidateLabels(labels); err != nil {
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

// StaticLabels creates an ApplierSet to set a static labels.
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
