package ephemeral_test

import (
	"os"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
)

type EnvVarSuite struct{}

var _ = Suite(&EnvVarSuite{})

func (s *EnvVarSuite) TestOSEnvVarKubePodOptions(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_OS_ENVVAR",
			Value: "1",
		},
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.OSEnvVar(expected[0].Name)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	registeredAppliers.Apply(&options)
	c.Assert(options.EnvironmentVariables, DeepEquals, []corev1.EnvVar(nil))

	// OS environment variable set
	os.Setenv(expected[0].Name, expected[0].Value)
	defer os.Unsetenv(expected[0].Name)

	registeredAppliers.Apply(&options)
	c.Assert(options.EnvironmentVariables, DeepEquals, expected)
}

func (s *EnvVarSuite) TestOSEnvVarCoreV1Container(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_OS_ENVVAR",
			Value: "1",
		},
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[corev1.Container]
	set := ephemeral.OSEnvVar(expected[0].Name)
	registeredAppliers.Register(set.Container)

	var options corev1.Container
	registeredAppliers.Apply(&options)
	c.Assert(options.Env, DeepEquals, []corev1.EnvVar(nil))

	// OS environment variable set
	os.Setenv(expected[0].Name, expected[0].Value)
	defer os.Unsetenv(expected[0].Name)

	registeredAppliers.Apply(&options)
	c.Assert(options.Env, DeepEquals, expected)
}

func (s *EnvVarSuite) TestStaticEnvVarKubePodOptions(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_STATIC_ENVVAR",
			Value: "1",
		},
	}

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.StaticEnvVar(expected[0].Name, expected[0].Value)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	registeredAppliers.Apply(&options)
	c.Assert(options.EnvironmentVariables, DeepEquals, expected)
}

func (s *EnvVarSuite) TestRegisteringStaticEnvVarCoreV1Container(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_STATIC_ENVVAR",
			Value: "1",
		},
	}

	var registeredAppliers ephemeral.ApplierList[corev1.Container]
	set := ephemeral.StaticEnvVar(expected[0].Name, expected[0].Value)
	registeredAppliers.Register(set.Container)

	var options corev1.Container
	registeredAppliers.Apply(&options)
	c.Assert(options.Env, DeepEquals, expected)
}
