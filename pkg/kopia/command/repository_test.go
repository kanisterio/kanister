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
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
)

func Test(t *testing.T) { check.TestingT(t) }

type RepositoryUtilsSuite struct{}

var _ = check.Suite(&RepositoryUtilsSuite{})

func (s *RepositoryUtilsSuite) TestRepositoryCreateUtil(c *check.C) {
	retentionMode := "Locked"
	retentionPeriod := 10 * time.Second
	for _, tc := range []struct {
		cmdArg            RepositoryCommandArgs
		location          map[string]string
		addRegisteredArgs bool
		check.Checker
		expectedCmd   []string
		expectedError string
	}{
		{
			cmdArg: RepositoryCommandArgs{
				CommandArgs: &CommandArgs{
					RepoPassword:   "pass123",
					ConfigFilePath: "/tmp/config.file",
					LogDirectory:   "/tmp/log.dir",
				},
				CacheDirectory: "/tmp/cache.dir",
				Hostname:       "test-hostname",
				CacheArgs: CacheArgs{
					ContentCacheLimitMB:  0,
					MetadataCacheLimitMB: 0,
				},
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
			},
			location:      map[string]string{},
			Checker:       check.NotNil,
			expectedError: "Failed to generate storage args: unsupported type for the location",
		},
		{
			cmdArg: RepositoryCommandArgs{
				CommandArgs: &CommandArgs{
					RepoPassword:   "pass123",
					ConfigFilePath: "/tmp/config.file",
					LogDirectory:   "/tmp/log.dir",
				},
				CacheDirectory: "/tmp/cache.dir",
				Hostname:       "test-hostname",
				CacheArgs: CacheArgs{
					ContentCacheLimitMB:  0,
					MetadataCacheLimitMB: 0,
				},
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location: map[string][]byte{
					"prefix": []byte("test-prefix"),
					"type":   []byte("filestore"),
				},
			},
			Checker: check.IsNil,
			expectedCmd: []string{"kopia",
				"--log-level=error",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
				"repository",
				"create",
				"--no-check-for-updates",
				"--cache-directory=/tmp/cache.dir",
				"--content-cache-size-limit-mb=0",
				"--metadata-cache-size-limit-mb=0",
				"--override-hostname=test-hostname",
				"--override-username=test-username",
				"filesystem",
				"--path=/mnt/data/test-prefix/test-path/prefix/",
			},
		},
		{
			cmdArg: RepositoryCommandArgs{
				CommandArgs: &CommandArgs{
					RepoPassword:   "pass123",
					ConfigFilePath: "/tmp/config.file",
					LogDirectory:   "/tmp/log.dir",
				},
				CacheDirectory:  "/tmp/cache.dir",
				Hostname:        "test-hostname",
				ContentCacheMB:  0,
				MetadataCacheMB: 0,
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location: map[string][]byte{
					"prefix": []byte("test-prefix"),
					"type":   []byte("filestore"),
				},
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			},
			Checker: check.IsNil,
			expectedCmd: []string{"kopia",
				"--log-level=error",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
				"repository",
				"create",
				"--no-check-for-updates",
				"--cache-directory=/tmp/cache.dir",
				"--content-cache-size-limit-mb=0",
				"--metadata-cache-size-limit-mb=0",
				"--override-hostname=test-hostname",
				"--override-username=test-username",
				"--retention-mode=Locked",
				"--retention-period=10s",
				"filesystem",
				"--path=/mnt/data/test-prefix/test-path/prefix/",
			},
		},
		{
			cmdArg: RepositoryCommandArgs{
				CommandArgs: &CommandArgs{
					RepoPassword:   "pass123",
					ConfigFilePath: "/tmp/config.file",
					LogDirectory:   "/tmp/log.dir",
				},
				CacheDirectory:  "/tmp/cache.dir",
				Hostname:        "test-hostname",
				ContentCacheMB:  0,
				MetadataCacheMB: 0,
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location: map[string][]byte{
					"prefix": []byte("test-prefix"),
					"type":   []byte("filestore"),
				},
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			},
			Checker: check.IsNil,
			expectedCmd: []string{"kopia",
				"--log-level=error",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
				"repository",
				"create",
				"--no-check-for-updates",
				"--cache-directory=/tmp/cache.dir",
				"--content-cache-size-limit-mb=0",
				"--metadata-cache-size-limit-mb=0",
				"--override-hostname=test-hostname",
				"--override-username=test-username",
				"--retention-mode=Locked",
				"--retention-period=10s",
				"--testflag=testvalue",
				"filesystem",
				"--path=/mnt/data/test-prefix/test-path/prefix/",
			},
			addRegisteredArgs: true,
		},
	} {
		if tc.addRegisteredArgs {
			flags := args.RepositoryCreate
			args.RepositoryCreate = args.Args{}
			args.RepositoryCreate.Set("--testflag", "testvalue")
			defer func() { args.RepositoryCreate = flags }()
		}
		cmd, err := RepositoryCreateCommand(tc.cmdArg)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd, check.DeepEquals, tc.expectedCmd)
		} else {
			c.Assert(strings.Contains(err.Error(), tc.expectedError), check.Equals, true)
		}
	}
}

func (s *RepositoryUtilsSuite) TestRepositoryConnectUtil(c *check.C) {
	pit := strfmt.NewDateTime()
	for _, tc := range []struct {
		cmdArg RepositoryCommandArgs
		check.Checker
		expectedCmd   []string
		expectedError string
	}{
		{
			cmdArg: RepositoryCommandArgs{
				CommandArgs: &CommandArgs{
					RepoPassword:   "pass123",
					ConfigFilePath: "/tmp/config.file",
					LogDirectory:   "/tmp/log.dir",
				},
				CacheDirectory: "/tmp/cache.dir",
				Hostname:       "test-hostname",
				CacheArgs: CacheArgs{
					ContentCacheLimitMB:  0,
					MetadataCacheLimitMB: 0,
				},

				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       map[string][]byte{},
			},
			Checker:       check.NotNil,
			expectedError: "Failed to generate storage args: unsupported type for the location",
		},
		{
			cmdArg: RepositoryCommandArgs{
				CommandArgs: &CommandArgs{
					RepoPassword:   "pass123",
					ConfigFilePath: "/tmp/config.file",
					LogDirectory:   "/tmp/log.dir",
				},
				CacheDirectory: "/tmp/cache.dir",
				CacheArgs: CacheArgs{
					ContentCacheLimitMB:  0,
					MetadataCacheLimitMB: 0,
				},
				RepoPathPrefix: "test-path/prefix",
				PITFlag:        pit,
				Location: map[string][]byte{
					"prefix": []byte("test-prefix"),
					"type":   []byte("filestore"),
				},
				ReadOnly: true,
			},
			Checker: check.IsNil,
			expectedCmd: []string{"kopia",
				"--log-level=error",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
				"repository",
				"connect",
				"--no-check-for-updates",
				"--readonly",
				"--cache-directory=/tmp/cache.dir",
				"--content-cache-size-limit-mb=0",
				"--metadata-cache-size-limit-mb=0",
				"filesystem",
				"--path=/mnt/data/test-prefix/test-path/prefix/",
				"--point-in-time=1970-01-01T00:00:00.000Z",
			},
		},
	} {
		cmd, err := RepositoryConnectCommand(tc.cmdArg)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd, check.DeepEquals, tc.expectedCmd)
		} else {
			c.Assert(strings.Contains(err.Error(), tc.expectedError), check.Equals, true)
		}
	}
}

func (s *RepositoryUtilsSuite) TestRepositoryConnectServerUtil(c *check.C) {
	cmd := func() []string {
		return RepositoryConnectServerCommand(RepositoryServerCommandArgs{
			UserPassword:   "testpass123",
			ConfigFilePath: "/tmp/config.file",
			LogDirectory:   "/tmp/log.dir",
			CacheDirectory: "/tmp/cache.dir",
			Hostname:       "test-hostname",
			Username:       "test-username",
			ServerURL:      "https://127.0.0.1:51515",
			Fingerprint:    "test-fingerprint",
			ReadOnly:       true,
			CacheArgs: CacheArgs{
				ContentCacheLimitMB:  0,
				MetadataCacheLimitMB: 0,
			},
		})
	}
	c.Assert(cmd(), check.DeepEquals, []string{"kopia",
		"--log-level=error",
		"--config-file=/tmp/config.file",
		"--log-dir=/tmp/log.dir",
		"--password=testpass123",
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
		"--url=https://127.0.0.1:51515",
		"--server-cert-fingerprint=test-fingerprint",
	})

	// register additional arguments
	flags := args.RepositoryConnectServer
	args.RepositoryConnectServer = args.Args{}
	args.RepositoryConnectServer.Set("--testflag", "testvalue")
	defer func() { args.RepositoryConnectServer = flags }()

	// ensure they are set
	c.Assert(cmd(), check.DeepEquals, []string{"kopia",
		"--log-level=error",
		"--config-file=/tmp/config.file",
		"--log-dir=/tmp/log.dir",
		"--password=testpass123",
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
		"--url=https://127.0.0.1:51515",
		"--testflag=testvalue", // additional flag is set
		"--server-cert-fingerprint=test-fingerprint",
	})
}

func (kRepoStatus *RepositoryUtilsSuite) TestRepositoryStatusCommand(c *check.C) {
	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := RepositoryStatusCommandArgs{
					CommandArgs: &CommandArgs{
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
				}
				return RepositoryStatusCommand(args)
			},
			expectedLog: "kopia --log-level=info --config-file=path/kopia.config --log-dir=cache/log repository status",
		},
		{
			f: func() []string {
				args := RepositoryStatusCommandArgs{
					CommandArgs: &CommandArgs{
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					GetJSONOutput: true,
				}
				return RepositoryStatusCommand(args)
			},
			expectedLog: "kopia --log-level=info --config-file=path/kopia.config --log-dir=cache/log repository status --json",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Check(cmd, check.Equals, tc.expectedLog)
	}
}

func (s *RepositoryUtilsSuite) TestRepositorySetParametersCommand(c *check.C) {
	retentionMode := "Locked"
	retentionPeriod := 10 * time.Second
	cmd := RepositorySetParametersCommand(RepositorySetParametersCommandArgs{
		CommandArgs: &CommandArgs{
			ConfigFilePath: "path/kopia.config",
			LogDirectory:   "cache/log",
		},
		RetentionMode:   retentionMode,
		RetentionPeriod: retentionPeriod,
	})
	c.Assert(cmd, check.DeepEquals, []string{"kopia",
		"--log-level=error",
		"--config-file=path/kopia.config",
		"--log-dir=cache/log",
		"repository",
		"set-parameters",
		"--retention-mode=Locked",
		"--retention-period=10s",
	})
}
