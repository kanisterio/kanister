package kancontroller

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMetricsEnabled(t *testing.T) {
	// Test case 1: Environment variable is not set
	os.Unsetenv(kanisterMetricsEnv)
	assert.False(t, metricsEnabled(), "Expected metricsEnabled() to return false when environment variable is not set")

	// Test case 2: Environment variable is set but has invalid value
	os.Setenv(kanisterMetricsEnv, "invalid")
	assert.False(t, metricsEnabled(), "Expected metricsEnabled() to return false when environment variable is set to an invalid boolean")

	// Test case 3: Environment variable is set to "true"
	os.Setenv(kanisterMetricsEnv, "true")
	assert.True(t, metricsEnabled(), "Expected metricsEnabled() to return true when environment variable is set to 'true'")

	// Test case 4: Environment variable is set to "false"
	os.Setenv(kanisterMetricsEnv, "false")
	assert.False(t, metricsEnabled(), "Expected metricsEnabled() to return false when environment variable is set to 'false'")
}
