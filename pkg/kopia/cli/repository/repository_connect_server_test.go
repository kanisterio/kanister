package repository

import (
	"testing"

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

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
			"--no-grpc",
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
}))
