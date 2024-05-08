package ephemeral

import (
	"os"

	"github.com/kanisterio/kanister/pkg/kube"
	corev1 "k8s.io/api/core/v1"
)

// OSEnvVar creates an ApplierSet to set an environment variable if its present
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
