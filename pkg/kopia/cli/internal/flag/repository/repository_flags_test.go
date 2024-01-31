package repository

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestRepositoryFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty Hostname should not generate a flag",
		Flag: Hostname(""),
	},
	{
		Name: "Hostname with value should generate a flag with given value",
		Flag: Hostname("hostname"),
		ExpectedCLI: []string{
			"--override-hostname=hostname",
		},
	},
	{
		Name: "Empty Username should not generate a flag",
		Flag: Username(""),
	},
	{
		Name: "Username with value should generate a flag with given value",
		Flag: Username("username"),
		ExpectedCLI: []string{
			"--override-username=username",
		},
	},
	{
		Name: "Empty BlobRetention should not generate a flag",
		Flag: BlobRetention("", time.Duration(0)),
	},
	{
		Name: "BlobRetention with values should generate multiple flags with given values",
		Flag: BlobRetention("mode", 24*time.Hour),
		ExpectedCLI: []string{
			"--retention-mode=mode",
			"--retention-period=24h0m0s",
		},
	},
	{
		Name: "BlobRetention with RetentionMode only should generate mode flag with zero period",
		Flag: BlobRetention("mode", 0),
		ExpectedCLI: []string{
			"--retention-mode=mode",
			"--retention-period=0s",
		},
	},
	{
		Name: "Empty PIT should not generate a flag",
		Flag: PIT(strfmt.DateTime{}),
	},
	{
		Name: "PIT with value should generate a flag with given value",
		Flag: PIT(func() strfmt.DateTime {
			dt, _ := strfmt.ParseDateTime("2024-01-02T03:04:05.678Z")
			return dt
		}()),
		ExpectedCLI: []string{
			"--point-in-time=2024-01-02T03:04:05.678Z",
		},
	},
	{
		Name: "Empty ServerURL should not generate a flag",
		Flag: ServerURL(""),
	},
	{
		Name: "ServerURL with value should generate a flag with given value",
		Flag: ServerURL("ServerURL"),
		ExpectedCLI: []string{
			"--url=ServerURL",
		},
	},
	{
		Name: "Empty ServerCertFingerprint should not generate a flag",
		Flag: ServerCertFingerprint(""),
	},
	{
		Name: "ServerCertFingerprint with value should generate a flag with given value and redact fingerprint for logs",
		Flag: ServerCertFingerprint("ServerCertFingerprint"),
		ExpectedCLI: []string{
			"--server-cert-fingerprint=ServerCertFingerprint",
		},
		ExpectedLog: "--server-cert-fingerprint=<****>",
	},
}))
