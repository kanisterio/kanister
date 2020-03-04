package config

import (
	"fmt"
	"os"

	"gopkg.in/check.v1"
)

func GetEnvOrSkip(c *check.C, varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		reason := fmt.Sprintf("Test %s requires the environemnt variable '%s'", c.TestName(), varName)
		c.Skip(reason)
	}
	return v
}
