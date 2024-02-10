package server

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestRepositoryFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "TLSGenerateCert(false) should not generate a flag",
		Flag: TLSGenerateCert(false),
	},
	{
		Name:        "TLSGenerateCert(true)should generate a flag",
		Flag:        TLSGenerateCert(true),
		ExpectedCLI: []string{"--tls-generate-cert"},
	},
	{
		Name: "Empty ServerAddress should not generate a flag",
		Flag: ServerAddress(""),
	},
	{
		Name:        "ServerAddress with value should generate a flag with given server address",
		Flag:        ServerAddress("server-address"),
		ExpectedCLI: []string{"--address=server-address"},
	},
	{
		Name: "Empty ServerUsername should not generate a flag",
		Flag: ServerUsername(""),
	},
	{
		Name:        "ServerUsername with value should generate a flag with given username",
		Flag:        ServerUsername("server-username"),
		ExpectedCLI: []string{"--server-username=server-username"},
	},
	{
		Name: "Empty ServerControlUsername should not generate a flag",
		Flag: ServerControlUsername(""),
	},
	{
		Name:        "ServerControlUsername with value should generate a control flag with given username",
		Flag:        ServerControlUsername("server-username"),
		ExpectedCLI: []string{"--server-control-username=server-username"},
	},
	{
		Name: "Empty ServerPassword should not generate a flag",
		Flag: ServerPassword(""),
	},
	{
		Name:        "Non-empty ServerPassword should generate a flag with given password and redact it for logs",
		Flag:        ServerPassword("server-password"),
		ExpectedCLI: []string{"--server-password=server-password"},
		ExpectedLog: "--server-password=<****>",
	},
	{
		Name: "Empty ServerPassword should not generate a flag",
		Flag: ServerControlPassword(""),
	},
	{
		Name:        "Non-empty ServerControlPasswordshould generate a control flag with given password and redact it for logs",
		Flag:        ServerControlPassword("server-password"),
		ExpectedCLI: []string{"--server-control-password=server-password"},
		ExpectedLog: "--server-control-password=<****>",
	},
	{
		Name: "Empty TLSCertFile should not generate a flag",
		Flag: TLSCertFile(""),
	},
	{
		Name:        "TLSCertFile with value should generate a flag with given tls-cert-file",
		Flag:        TLSCertFile("tls-cert-file"),
		ExpectedCLI: []string{"--tls-cert-file=tls-cert-file"},
	},
	{
		Name: "Empty TLSKeyFile should not generate a flag",
		Flag: TLSKeyFile(""),
	},
	{
		Name:        "TLSKeyFile with value should generate a flag with given tls-key-file",
		Flag:        TLSKeyFile("tls-key-file"),
		ExpectedCLI: []string{"--tls-key-file=tls-key-file"},
	},
	{
		Name: "Empty Background should not generate a flag",
		Flag: Background(false),
	},
	{
		Name:        "Background(true) should generate redirect to dev null and background flag",
		Flag:        Background(true),
		ExpectedCLI: []string{shellRedirectToDevNull, shellRunInBackground},
	},
	{
		Name: "Empty ServerCertFingerprint should not generate a flag",
		Flag: ServerCertFingerprint(""),
	},
	{
		Name:        "ServerCertFingerprint with value should generate a flag with given fingerprint and redact fingerprint for logs",
		Flag:        ServerCertFingerprint("server-cert-fingerprint"),
		ExpectedCLI: []string{"--server-cert-fingerprint=server-cert-fingerprint"},
		ExpectedLog: "--server-cert-fingerprint=<****>",
	},
	{
		Name:        "Empty Username should generate an error",
		Flag:        Username(""),
		ExpectedErr: cli.ErrInvalidFlag,
	},
	{
		Name:        "Username with value should generate a flag with given username",
		Flag:        Username("username"),
		ExpectedCLI: []string{"username"},
	},
	{
		Name: "Empty UserPassword should not generate a flag",
		Flag: UserPassword(""),
	},
	{
		Name:        "UserPassword with value should generate a flag with given password and redact it for logs",
		Flag:        UserPassword("password"),
		ExpectedCLI: []string{"--user-password=password"},
		ExpectedLog: "--user-password=<****>",
	},
}))
