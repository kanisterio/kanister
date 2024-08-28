// Copyright 2024 The Kanister Authors.
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

package repository

import (
	"testing"

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryConnectServerCommand(t *testing.T) { check.TestingT(t) }

// Test Repository Connect Server command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "repository connect server",
		Command: func() (*safecli.Builder, error) {
			args := ConnectServerArgs{
				Common:      common,
				Cache:       cache,
				Hostname:    "test-hostname",
				Username:    "test-username",
				ServerURL:   "http://test-server",
				Fingerprint: "test-fingerprint",
				ReadOnly:    true,
			}
			return ConnectServer(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"connect",
			"server",
			"--no-check-for-updates",
			"--readonly",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"--url=http://test-server",
			"--server-cert-fingerprint=test-fingerprint",
		},
	},
	{
		Name: "repository connect server, with additional args",
		Command: func() (*safecli.Builder, error) {
			arguments := ConnectServerArgs{
				Common:      common,
				Cache:       cache,
				Hostname:    "test-hostname",
				Username:    "test-username",
				ServerURL:   "http://test-server",
				Fingerprint: "test-fingerprint",
				ReadOnly:    true,
			}

			flags := args.RepositoryConnectServer
			args.RepositoryConnectServer = args.Args{}
			args.RepositoryConnectServer.Set("--testflag", "testvalue")
			defer func() { args.RepositoryConnectServer = flags }()

			return ConnectServer(arguments)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"connect",
			"server",
			"--no-check-for-updates",
			"--readonly",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"--url=http://test-server",
			"--server-cert-fingerprint=test-fingerprint",
			"--testflag=testvalue",
		},
	},
}))
