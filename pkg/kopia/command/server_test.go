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
	"strings"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
)

type KopiaServerTestSuite struct{}

var _ = Suite(&KopiaServerTestSuite{})

func (kServer *KopiaServerTestSuite) TestServerCommands(c *C) {
	commandArgs := &CommandArgs{
		RepoPassword:   "encr-key",
		ConfigFilePath: "path/kopia.config",
		LogDirectory:   "cache/log",
	}
	cacheArgs := CacheArgs{
		ContentCacheLimitMB:  500,
		MetadataCacheLimitMB: 500,
	}

	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := ServerStartCommandArgs{
					CommandArgs:          commandArgs,
					CacheArgs:            cacheArgs,
					CacheDirectory:       "cache/dir",
					ServerAddress:        "a-server-address",
					TLSCertFile:          "/path/to/cert/tls.crt",
					TLSKeyFile:           "/path/to/key/tls.key",
					ServerUsername:       "a-username@a-hostname",
					ServerPassword:       "a-user-password",
					AutoGenerateCert:     true,
					Background:           true,
					EnablePprof:          true,
					MetricsListenAddress: "a-server-address:51516",
				}
				return ServerStart(args)
			},
			expectedLog: "bash -o errexit -c kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=a-user-password --server-control-username=a-username@a-hostname --server-control-password=a-user-password --cache-directory=cache/dir --content-cache-size-limit-mb=500 --metadata-cache-size-limit-mb=500 --enable-pprof --metrics-listen-addr=a-server-address:51516 > /dev/null 2>&1 &",
		},
		{
			f: func() []string {
				args := ServerStartCommandArgs{
					CommandArgs:      commandArgs,
					CacheArgs:        cacheArgs,
					CacheDirectory:   "cache/dir",
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: true,
					Background:       true,
				}
				return ServerStart(args)
			},
			expectedLog: "bash -o errexit -c kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=a-user-password --server-control-username=a-username@a-hostname --server-control-password=a-user-password --cache-directory=cache/dir --content-cache-size-limit-mb=500 --metadata-cache-size-limit-mb=500 > /dev/null 2>&1 &",
		},
		{
			f: func() []string {
				args := ServerStartCommandArgs{
					CommandArgs:      commandArgs,
					CacheArgs:        cacheArgs,
					CacheDirectory:   "cache/dir",
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: true,
					Background:       false,
				}
				return ServerStart(args)
			},
			expectedLog: "bash -o errexit -c kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=a-user-password --server-control-username=a-username@a-hostname --server-control-password=a-user-password --cache-directory=cache/dir --content-cache-size-limit-mb=500 --metadata-cache-size-limit-mb=500",
		},
		{
			f: func() []string {
				args := ServerStartCommandArgs{
					CommandArgs:      commandArgs,
					CacheArgs:        cacheArgs,
					CacheDirectory:   "cache/dir",
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: false,
					Background:       true,
				}
				return ServerStart(args)
			},
			expectedLog: "bash -o errexit -c kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=a-user-password --server-control-username=a-username@a-hostname --server-control-password=a-user-password --cache-directory=cache/dir --content-cache-size-limit-mb=500 --metadata-cache-size-limit-mb=500 > /dev/null 2>&1 &",
		},
		{
			f: func() []string {
				args := ServerStartCommandArgs{
					CommandArgs:      commandArgs,
					CacheArgs:        cacheArgs,
					CacheDirectory:   "cache/dir",
					ServerAddress:    "a-server-address",
					TLSCertFile:      "/path/to/cert/tls.crt",
					TLSKeyFile:       "/path/to/key/tls.key",
					ServerUsername:   "a-username@a-hostname",
					ServerPassword:   "a-user-password",
					AutoGenerateCert: false,
					Background:       true,
					ReadOnly:         true,
					HtpasswdFilePath: "/path/htpasswd",
				}
				return ServerStart(args)
			},
			expectedLog: "bash -o errexit -c kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --htpasswd-file=/path/htpasswd --server-username=a-username@a-hostname --server-password=a-user-password --server-control-username=a-username@a-hostname --server-control-password=a-user-password --cache-directory=cache/dir --content-cache-size-limit-mb=500 --metadata-cache-size-limit-mb=500 --readonly > /dev/null 2>&1 &",
		},
		{
			f: func() []string {
				args := ServerStatusCommandArgs{
					CommandArgs:    commandArgs,
					ServerAddress:  "a-server-address",
					ServerUsername: "a-username@a-hostname",
					ServerPassword: "a-user-password",
					Fingerprint:    "a-fingerprint",
				}
				return ServerStatus(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server status --address=a-server-address --server-cert-fingerprint=a-fingerprint --server-username=a-username@a-hostname --server-password=a-user-password",
		},
		{
			f: func() []string {
				args := ServerAddUserCommandArgs{
					CommandArgs:  commandArgs,
					NewUsername:  "a-username@a-hostname",
					UserPassword: "a-user-password",
				}
				return ServerAddUser(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key server user add a-username@a-hostname --user-password=a-user-password",
		},
		{
			f: func() []string {
				flags := args.UserAddSet
				args.UserAddSet = args.Args{}
				args.UserAddSet.Set("--testflag", "testvalue")
				defer func() { args.UserAddSet = flags }()
				args := ServerAddUserCommandArgs{
					CommandArgs:  commandArgs,
					NewUsername:  "a-username@a-hostname",
					UserPassword: "a-user-password",
				}
				return ServerAddUser(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key server user add a-username@a-hostname --user-password=a-user-password --testflag=testvalue",
		},
		{
			f: func() []string {
				args := ServerSetUserCommandArgs{
					CommandArgs:  commandArgs,
					NewUsername:  "a-username@a-hostname",
					UserPassword: "a-user-password",
				}
				return ServerSetUser(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key server user set a-username@a-hostname --user-password=a-user-password",
		},
		{
			f: func() []string {
				flags := args.UserAddSet
				args.UserAddSet = args.Args{}
				args.UserAddSet.Set("--testflag", "testvalue")
				defer func() { args.UserAddSet = flags }()
				args := ServerSetUserCommandArgs{
					CommandArgs:  commandArgs,
					NewUsername:  "a-username@a-hostname",
					UserPassword: "a-user-password",
				}
				return ServerSetUser(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key server user set a-username@a-hostname --user-password=a-user-password --testflag=testvalue",
		},
		{
			f: func() []string {
				args := ServerListUserCommmandArgs{
					CommandArgs: commandArgs,
				}
				return ServerListUser(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key server user list --json",
		},
		{
			f: func() []string {
				args := ServerRefreshCommandArgs{
					CommandArgs:    commandArgs,
					ServerAddress:  "a-server-address",
					ServerUsername: "a-username@a-hostname",
					ServerPassword: "a-user-password",
					Fingerprint:    "a-fingerprint",
				}
				return ServerRefresh(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key server refresh --server-cert-fingerprint=a-fingerprint --address=a-server-address --server-username=a-username@a-hostname --server-password=a-user-password",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Check(cmd, Equals, tc.expectedLog)
	}
}
