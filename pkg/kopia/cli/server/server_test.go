package server

import (
	"testing"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestServerCommands(t *testing.T) { check.TestingT(t) }

// Test Refresh command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "server refresh",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RefreshArgs{
				CommonArgs:     test.CommonArgs,
				ServerAddress:  "a-server-address",
				ServerUsername: "a-username@a-hostname",
				ServerPassword: "a-user-password",
				Fingerprint:    "a-fingerprint",
			}
			return Refresh(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"server",
			"refresh",
			"--address=a-server-address",
			"--server-username=a-username@a-hostname",
			"--server-password=a-user-password",
			"--server-cert-fingerprint=a-fingerprint",
		},
	},
}))

// Test Start command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "server start with enabled Background",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StartArgs{
				CommonArgs:       test.CommonArgs,
				ServerAddress:    "a-server-address",
				TLSCertFile:      "/path/to/cert/tls.crt",
				TLSKeyFile:       "/path/to/key/tls.key",
				ServerUsername:   "a-username@a-hostname",
				ServerPassword:   "a-user-password",
				AutoGenerateCert: true,
				Background:       true,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"bash", "-o", "errexit", "-c",
			"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"server",
			"start",
			"--tls-generate-cert",
			"--tls-cert-file=/path/to/cert/tls.crt",
			"--tls-key-file=/path/to/key/tls.key",
			"--address=a-server-address",
			"--server-username=a-username@a-hostname",
			"--server-password=a-user-password",
			"--server-control-username=a-username@a-hostname",
			"--server-control-password=a-user-password",
			"--no-grpc",
			"> /dev/null 2>&1",
			"&",
		},
	},
	{
		Name: "server start with disabled Background",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StartArgs{
				CommonArgs:       test.CommonArgs,
				ServerAddress:    "a-server-address",
				TLSCertFile:      "/path/to/cert/tls.crt",
				TLSKeyFile:       "/path/to/key/tls.key",
				ServerUsername:   "a-username@a-hostname",
				ServerPassword:   "a-user-password",
				AutoGenerateCert: true,
				Background:       false,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"bash", "-o", "errexit", "-c",
			"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"server",
			"start",
			"--tls-generate-cert",
			"--tls-cert-file=/path/to/cert/tls.crt",
			"--tls-key-file=/path/to/key/tls.key",
			"--address=a-server-address",
			"--server-username=a-username@a-hostname",
			"--server-password=a-user-password",
			"--server-control-username=a-username@a-hostname",
			"--server-control-password=a-user-password",
			"--no-grpc",
		},
	},
	{
		Name: "server start with disabled Background and BashWrapper",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StartArgs{
				CommonArgs:         test.CommonArgs,
				ServerAddress:      "a-server-address",
				TLSCertFile:        "/path/to/cert/tls.crt",
				TLSKeyFile:         "/path/to/key/tls.key",
				ServerUsername:     "a-username@a-hostname",
				ServerPassword:     "a-user-password",
				AutoGenerateCert:   true,
				Background:         false,
				DisableBashWrapper: true,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"server",
			"start",
			"--tls-generate-cert",
			"--tls-cert-file=/path/to/cert/tls.crt",
			"--tls-key-file=/path/to/key/tls.key",
			"--address=a-server-address",
			"--server-username=a-username@a-hostname",
			"--server-password=a-user-password",
			"--server-control-username=a-username@a-hostname",
			"--server-control-password=a-user-password",
			"--no-grpc",
		},
	},
	{
		Name: "server start with disabled AutoGenerateCert",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StartArgs{
				CommonArgs:       test.CommonArgs,
				ServerAddress:    "a-server-address",
				TLSCertFile:      "/path/to/cert/tls.crt",
				TLSKeyFile:       "/path/to/key/tls.key",
				ServerUsername:   "a-username@a-hostname",
				ServerPassword:   "a-user-password",
				AutoGenerateCert: false,
				Background:       true,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"bash", "-o", "errexit", "-c",
			"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"server",
			"start",
			"--tls-cert-file=/path/to/cert/tls.crt",
			"--tls-key-file=/path/to/key/tls.key",
			"--address=a-server-address",
			"--server-username=a-username@a-hostname",
			"--server-password=a-user-password",
			"--server-control-username=a-username@a-hostname",
			"--server-control-password=a-user-password",
			"--no-grpc",
			"> /dev/null 2>&1",
			"&",
		},
	},
}))

// Test Status command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "server status with Fingerprint",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StatusArgs{
				CommonArgs:     test.CommonArgs,
				ServerAddress:  "a-server-address",
				ServerUsername: "a-username@a-hostname",
				ServerPassword: "a-user-password",
				Fingerprint:    "a-fingerprint",
			}
			return Status(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"server",
			"status",
			"--address=a-server-address",
			"--server-username=a-username@a-hostname",
			"--server-password=a-user-password",
			"--server-cert-fingerprint=a-fingerprint",
		},
	},
}))

// Test Server User commands
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "server user add",
		CLI: func() (safecli.CommandBuilder, error) {
			args := UserAddArgs{
				CommonArgs:   test.CommonArgs,
				Username:     "a-username@a-hostname",
				UserPassword: "a-user-password",
			}
			return UserAdd(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"server",
			"user",
			"add",
			"a-username@a-hostname",
			"--user-password=a-user-password",
		},
	},
	{
		Name: "server user set",
		CLI: func() (safecli.CommandBuilder, error) {
			args := UserSetArgs{
				CommonArgs:   test.CommonArgs,
				Username:     "a-username@a-hostname",
				UserPassword: "a-user-password",
			}
			return UserSet(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"server",
			"user",
			"set",
			"a-username@a-hostname",
			"--user-password=a-user-password",
		},
	},
	{
		Name: "server user list",
		CLI: func() (safecli.CommandBuilder, error) {
			args := UserListArgs{
				CommonArgs: test.CommonArgs,
			}
			return UserList(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"server",
			"user",
			"list",
			"--json",
		},
	},
}))
