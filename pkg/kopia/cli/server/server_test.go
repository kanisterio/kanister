package server

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
)

func TestServerCommands(t *testing.T) { check.TestingT(t) }

// Test Refresh command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "ServerRefresh",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerStart with enabled Background",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerStart with disabled Background",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerStart with disabled Background and BashWrapper",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerStart with disabled AutoGenerateCert",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerStatus with Fingerprint",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerUserAdd",
		CLI: func() (safecli.CommandBuilder, error) {
			args := UserAddArgs{
				CommonArgs:   test.CommonArgs,
				Username:     "a-username@a-hostname",
				UserPassword: "a-user-password",
			}
			return UserAdd(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerUserSet",
		CLI: func() (safecli.CommandBuilder, error) {
			args := UserSetArgs{
				CommonArgs:   test.CommonArgs,
				Username:     "a-username@a-hostname",
				UserPassword: "a-user-password",
			}
			return UserSet(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "ServerUserList",
		CLI: func() (safecli.CommandBuilder, error) {
			args := UserListArgs{
				CommonArgs: test.CommonArgs,
			}
			return UserList(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"server",
			"user",
			"list",
			"--json",
		},
	},
}))
