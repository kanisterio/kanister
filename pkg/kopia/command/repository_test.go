package command

import (
	"strings"
	"testing"

	"github.com/go-openapi/strfmt"
	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type RepositoryUtilsSuite struct{}

var _ = check.Suite(&RepositoryUtilsSuite{})

func (s *RepositoryUtilsSuite) TestRepositoryCreateUtil(c *check.C) {
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
				LocationSecret:  &v1.Secret{},
				CredsSecret:     &v1.Secret{},
				CacheDirectory:  "/tmp/cache.dir",
				Hostname:        "test-hostname",
				ContentCacheMB:  0,
				MetadataCacheMB: 0,
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
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
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						"prefix": "test-prefix",
						"type":   "filestore",
					},
				},
				CredsSecret:     &v1.Secret{},
				CacheDirectory:  "/tmp/cache.dir",
				Hostname:        "test-hostname",
				ContentCacheMB:  0,
				MetadataCacheMB: 0,
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
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
				"--content-cache-size-mb=0",
				"--metadata-cache-size-mb=0",
				"--override-hostname=test-hostname",
				"--override-username=test-username",
				"filesystem",
				"--path=/mnt/data/test-prefix/test-path/prefix/",
			},
		},
	} {
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
				LocationSecret:  &v1.Secret{},
				CredsSecret:     &v1.Secret{},
				CacheDirectory:  "/tmp/cache.dir",
				Hostname:        "test-hostname",
				ContentCacheMB:  0,
				MetadataCacheMB: 0,
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
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
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						"prefix": "test-prefix",
						"type":   "filestore",
					},
				},
				CredsSecret:     &v1.Secret{},
				CacheDirectory:  "/tmp/cache.dir",
				ContentCacheMB:  0,
				MetadataCacheMB: 0,
				RepoPathPrefix:  "test-path/prefix",
				PITFlag:         pit,
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
				"--cache-directory=/tmp/cache.dir",
				"--content-cache-size-mb=0",
				"--metadata-cache-size-mb=0",
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
	cmd := RepositoryConnectServerCommand(RepositoryServerCommandArgs{
		UserPassword:    "testpass123",
		ConfigFilePath:  "/tmp/config.file",
		LogDirectory:    "/tmp/log.dir",
		CacheDirectory:  "/tmp/cache.dir",
		Hostname:        "test-hostname",
		Username:        "test-username",
		ServerURL:       "https://127.0.0.1:51515",
		Fingerprint:     "test-fingerprint",
		ContentCacheMB:  0,
		MetadataCacheMB: 0,
	})
	c.Assert(cmd, check.DeepEquals, []string{"kopia",
		"--log-level=error",
		"--config-file=/tmp/config.file",
		"--log-dir=/tmp/log.dir",
		"--password=testpass123",
		"repository",
		"connect",
		"server",
		"--no-check-for-updates",
		"--no-grpc",
		"--cache-directory=/tmp/cache.dir",
		"--content-cache-size-mb=0",
		"--metadata-cache-size-mb=0",
		"--override-hostname=test-hostname",
		"--override-username=test-username",
		"--url=https://127.0.0.1:51515",
		"--server-cert-fingerprint=test-fingerprint",
	})
}
