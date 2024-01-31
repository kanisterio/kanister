package test

import (
	"testing"

	"gopkg.in/check.v1"
)

func TestMaintenanceCommands(t *testing.T) { check.TestingT(t) }

type RedactSuite struct{}

var _ = check.Suite(&RedactSuite{})

func (s *RedactSuite) TestRedactCLI(c *check.C) {
	cli := []string{
		"--password=secret",
		"--user-password=123456",
		"--server-password=pass123",
		"--server-control-password=abc123",
		"--server-cert-fingerprint=abcd1234",
		"--other-flag=value",
	}

	expected := "--password=<****> --user-password=<****> --server-password=<****> --server-control-password=<****> --server-cert-fingerprint=<****> --other-flag=value"

	result := RedactCLI(cli)
	c.Assert(result, check.Equals, expected)
}
