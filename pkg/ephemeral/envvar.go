package ephemeral

import (
	"os"

	"github.com/kanisterio/kanister/pkg/kube"
	corev1 "k8s.io/api/core/v1"
)

// OSEnvVar registers an environment variable if its present on the current OS to be applied to the PodOptions.
func OSEnvVar(name string) Applier {
	return ApplierFunc(func(options any) {
		if val, present := os.LookupEnv(name); present {
			switch v := options.(type) {
			case *kube.PodOptions:
				v.EnvironmentVariables = append(
					v.EnvironmentVariables,
					corev1.EnvVar{
						Name:  name,
						Value: val,
					},
				)
			case *corev1.Container:
				v.Env = append(
					v.Env,
					corev1.EnvVar{
						Name:  name,
						Value: val,
					},
				)
			}
		}
	})
}

// StaticEnvVar registers a static environment variable to be applied to the PodOptions.
func StaticEnvVar(name, value string) Applier {
	return ApplierFunc(func(options any) {
		switch v := options.(type) {
		case *kube.PodOptions:
			v.EnvironmentVariables = append(
				v.EnvironmentVariables,
				corev1.EnvVar{
					Name:  name,
					Value: value,
				},
			)
		case *corev1.Container:
			v.Env = append(
				v.Env,
				corev1.EnvVar{
					Name:  name,
					Value: value,
				},
			)
		}
	})
}
