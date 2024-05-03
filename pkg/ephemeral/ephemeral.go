package ephemeral

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/ephemeral/internal/envvar"
)

// NewEnvironmentVariables creates a new slice of k8s environment variables
// based on the current registered configuration.
func GlobalEnvVars() []corev1.EnvVar {
	return envvar.EnabledEnvVars()
}

// RegisterOSEnvVar registers an environment variable to be injected into
// ephemeral pods conditionally if already present on the OS.
func RegisterOSEnvVar(envName string) {
	envvar.Register(envvar.OSEnvVar(envName))
}

// RegisterStaticEnvVar registers an environment variable and value to be
// injected into ephemeral pods.
func RegisterStaticEnvVar(envName, envValue string) {
	envvar.Register(envvar.StaticEnvVar{
		EnvName:  envName,
		EnvValue: envValue,
	})
}
