package ephemeral_test

import (
	"os"
	"testing"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type EphemeralSuite struct{}

var _ = Suite(&EphemeralSuite{})

func (s *EphemeralSuite) TestRegisteringOSEnvVarKubePodOptions(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_OS_ENVVAR",
			Value: "1",
		},
	}

	var options kube.PodOptions

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList
	registeredAppliers.Register(ephemeral.OSEnvVar(expected[0].Name))
	registeredAppliers.Apply(&options)

	c.Assert(options.EnvironmentVariables, DeepEquals, []corev1.EnvVar(nil))

	// OS environment variable set
	os.Setenv(expected[0].Name, expected[0].Value)
	defer os.Unsetenv(expected[0].Name)

	registeredAppliers = ephemeral.ApplierList{}
	registeredAppliers.Register(ephemeral.OSEnvVar(expected[0].Name))
	registeredAppliers.Apply(&options)

	c.Assert(options.EnvironmentVariables, DeepEquals, expected)
}

func (s *EphemeralSuite) TestRegisteringOSEnvVarCoreV1Container(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_OS_ENVVAR",
			Value: "1",
		},
	}

	var options corev1.Container

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList
	registeredAppliers.Register(ephemeral.OSEnvVar(expected[0].Name))
	registeredAppliers.Apply(&options)

	c.Assert(options.Env, DeepEquals, []corev1.EnvVar(nil))

	// OS environment variable set
	os.Setenv(expected[0].Name, expected[0].Value)
	defer os.Unsetenv(expected[0].Name)

	registeredAppliers = ephemeral.ApplierList{}
	registeredAppliers.Register(ephemeral.OSEnvVar(expected[0].Name))
	registeredAppliers.Apply(&options)

	c.Assert(options.Env, DeepEquals, expected)
}

func (s *EphemeralSuite) TestRegisteringStaticEnvVarKubePodOptions(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_STATIC_ENVVAR",
			Value: "1",
		},
	}

	var options kube.PodOptions

	var registeredAppliers ephemeral.ApplierList
	registeredAppliers.Register(ephemeral.StaticEnvVar(expected[0].Name, expected[0].Value))
	registeredAppliers.Apply(&options)

	c.Assert(options.EnvironmentVariables, DeepEquals, expected)
}

func (s *EphemeralSuite) TestRegisteringStaticEnvVarCoreV1Container(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_STATIC_ENVVAR",
			Value: "1",
		},
	}

	var options corev1.Container

	var registeredAppliers ephemeral.ApplierList
	registeredAppliers.Register(ephemeral.StaticEnvVar(expected[0].Name, expected[0].Value))
	registeredAppliers.Apply(&options)

	c.Assert(options.Env, DeepEquals, expected)
}
