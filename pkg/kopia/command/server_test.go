// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

func TestServerCommandsLogging(t *testing.T) {
	c := qt.New(t)

	for _, tc := range []struct {
		f           func() logsafe.Cmd
		expectedLog string
	}{
		{
			f: func() logsafe.Cmd {
				args := ServerStartCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: true,
					Background:       true,
				}
				return serverStartCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=<****> --server-control-username=a-username@a-hostname --server-control-password=<****> --no-grpc > /dev/null 2>&1 &",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerStartCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: true,
					Background:       false,
				}
				return serverStartCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=<****> --server-control-username=a-username@a-hostname --server-control-password=<****> --no-grpc",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerStartCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: false,
					Background:       true,
				}
				return serverStartCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=<****> --server-control-username=a-username@a-hostname --server-control-password=<****> --no-grpc > /dev/null 2>&1 &",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerStatusCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					ServerAddress:  "a-server-address",
					ServerUsername: "a-username@a-hostname",
					ServerPassword: "a-user-password",
					Fingerprint:    "a-fingerprint",
				}
				return serverStatusCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server status --address=a-server-address --server-cert-fingerprint=<****> --server-username=a-username@a-hostname --server-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerAddUserCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					NewUsername:  "a-username@a-hostname",
					UserPassword: "a-user-password",
				}
				return serverAddUserCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server user add a-username@a-hostname --user-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerSetUserCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					NewUsername:  "a-username@a-hostname",
					UserPassword: "a-user-password",
				}
				return serverSetUserCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server user set a-username@a-hostname --user-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerListUserCommmandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
				}
				return serverListUserCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server user list --json",
		},
		{
			f: func() logsafe.Cmd {
				args := ServerRefreshCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					ServerAddress:  "a-server-address",
					ServerUsername: "a-username@a-hostname",
					ServerPassword: "a-user-password",
					Fingerprint:    "a-fingerprint",
				}
				return serverRefreshCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server refresh --server-cert-fingerprint=<****> --address=a-server-address --server-username=a-username@a-hostname --server-password=<****>",
		},
	} {
		cmd := tc.f()
		c.Check(cmd.String(), qt.Equals, tc.expectedLog)
	}
}
