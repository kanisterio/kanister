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
