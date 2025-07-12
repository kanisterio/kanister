package ephemeral

import (
	"encoding/json"
	"maps"
	"os"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func AnnotationsFromEnvVar(name string) ApplierSet {
	return ApplierSet{
		PodOptions: ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			if val, present := os.LookupEnv(name); present {
				var annotations map[string]string
				if err := json.Unmarshal([]byte(val), &annotations); err != nil {
					return err
				}

				if err := utils.ValidateAnnotations(annotations); err != nil {
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

				if err := utils.ValidateAnnotations(annotations); err != nil {
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
