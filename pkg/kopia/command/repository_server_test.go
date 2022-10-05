package command

import "gopkg.in/check.v1"

type RepositoryServerUtilsTestSuite struct{}

var _ = check.Suite(&RepositoryServerUtilsTestSuite{})

func (s *RepositoryServerUtilsTestSuite) TestRepositoryConnectServerUtil(c *check.C) {
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
	c.Assert(cmd, check.DeepEquals, []string{"kopia", "--log-level=error", "--config-file=/tmp/config.file", "--log-dir=/tmp/log.dir", "--password=testpass123", "repository", "connect", "server", "--no-check-for-updates", "--no-grpc", "--cache-directory=/tmp/cache.dir", "--content-cache-size-mb=0", "--metadata-cache-size-mb=0", "--override-hostname=test-hostname", "--override-username=test-username", "--url=https://127.0.0.1:51515", "--server-cert-fingerprint=test-fingerprint"})
}
