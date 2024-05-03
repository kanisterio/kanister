package envvar

import (
	"errors"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
)

var (
	ErrNameNotAvailable = errors.New("environment variable name is not available")
)

// EnvVar is the interface which registers environment variables to be used in the
// ephemeral pod.
type EnvVar interface {
	Enabled() bool

	Name() string
	Value() string
}

// StaticEnvVar implements the EnvVar interface and is used when specifying an
// environment variable that does not come directly from the parent pod environment
// variables.
type StaticEnvVar struct {
	EnvName  string
	EnvValue string
}

var _ EnvVar = StaticEnvVar{}

func (v StaticEnvVar) Enabled() bool { return true }
func (v StaticEnvVar) Name() string  { return v.EnvName }
func (v StaticEnvVar) Value() string { return v.EnvValue }

// ConditionalEnvVar implements the EnvVar interface and is used when mapping an
// parent pods environment variable into the ephemeral pod.
type ConditionalEnvVar struct {
	EnvName string

	EnabledFunc  func() bool
	EnvValueFunc func() string
}

var _ EnvVar = ConditionalEnvVar{}

func (v ConditionalEnvVar) Enabled() bool {
	if v.EnabledFunc != nil {
		return v.EnabledFunc()
	}

	return true
}

func (v ConditionalEnvVar) Name() string {
	return v.EnvName
}

func (v ConditionalEnvVar) Value() string {
	if v.EnvValueFunc != nil {
		return v.EnvValueFunc()
	}

	return ""
}

var (
	// EnvironmentVariables is the map holding all the registered environment
	// variables.
	EnvironmentVariables = map[string]EnvVar{}
)

// OSEnvVar is a helper function which creates a ConditionalVar and uses the
// OS's environment variable to compare against the input variable to see whether
// the EnvVar is enabled.
func OSEnvVar(envName string) ConditionalEnvVar {
	return ConditionalEnvVar{
		EnvName: envName,
		EnabledFunc: func() bool {
			_, present := os.LookupEnv(envName)
			return present
		},
		EnvValueFunc: func() string {
			return os.Getenv(envName)
		},
	}
}

// Register adds support for constructing k8s environment variables with the
// given name.
func Register(envVar EnvVar) {
	if _, present := EnvironmentVariables[envVar.Name()]; present {
		panic(fmt.Sprintf("environment variables named %s already registered", envVar.Name()))
	}

	EnvironmentVariables[envVar.Name()] = envVar
}

// EnabledEnvVars constructs a slice of k8s environment variables if they've
// been enabled.
func EnabledEnvVars() []corev1.EnvVar {
	var envvars []corev1.EnvVar
	for _, envvar := range EnvironmentVariables {
		if envvar.Enabled() {
			envvars = append(envvars, corev1.EnvVar{
				Name:  envvar.Name(),
				Value: envvar.Value(),
			})
		}
	}

	return envvars
}
