// Copyright 2023 The Kanister Authors.
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

package datamover

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/util/rand"

	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type RepositoryServerSuite struct {
	ctx                context.Context
	address            string
	kopiaCacheDir      string
	kopiaLogDir        string
	kopiaConfigDir     string
	tlsDir             string
	serverHost         string
	serverUsername     string
	serverPassword     string
	repositoryUser     string
	testUsername       string
	testUserPassword   string
	repositoryPassword string
	repoPathPrefix     string
	fingerprint        string
}

const (
	TestRepositoryEncryptionKey = "TEST_REPOSITORY_ENCRYPTION_KEY"
)

var _ = Suite(&RepositoryServerSuite{})

func (rss *RepositoryServerSuite) SetUpSuite(c *C) {
	// Check if kopia binary exists in PATH
	if !CommandExists("kopia") {
		c.Skip("Skipping repository server datamover unit test. Couldn't find kopia binary in the path.")
	}

	// Setting Up Repository Server Address
	rss.address = fmt.Sprintf("%s:%s", "https://0.0.0.0", strconv.Itoa(rand.IntnRange(50000, 60000)))

	// Setting Up Repository Server User Access
	rss.serverUsername = "user@localhost"
	rss.serverPassword = "testPassword"
	rss.serverHost = "localhost"
	rss.testUsername = fmt.Sprintf("%s-%s", "testuser", rand.String(5))
	rss.testUserPassword = rand.String(8)

	// Setting Up Repository Access
	rss.repositoryUser = "repositoryUser"
	rss.repositoryPassword = testutil.GetEnvOrSkip(c, TestRepositoryEncryptionKey)
	rss.repoPathPrefix = path.Join("kopia-int", "repository-server-datamover-test")

	rss.ctx = context.Background()

	// Setting Up Kopia Cache, Log and Config Dir
	rss.kopiaCacheDir = kopiacmd.DefaultCacheDirectory
	rss.kopiaLogDir = kopiacmd.DefaultLogDirectory
	rss.kopiaConfigDir = kopiacmd.DefaultConfigDirectory

	// Setting Up TLS Dir
	temp := c.MkDir()
	rss.tlsDir = filepath.Join(temp, "tls-"+rand.String(5))
}

func (rss *RepositoryServerSuite) setupKopiaRepositoryServer(c *C) {
	// Setting Up Kopia Repository
	contentCacheMB, metadataCacheMB := kopiacmd.GetGeneralCacheSizeSettings()
	repoCommandArgs := kopiacmd.RepositoryCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   rss.repositoryPassword,
			ConfigFilePath: rss.kopiaConfigDir,
			LogDirectory:   rss.kopiaLogDir,
		},
		CacheDirectory: rss.kopiaCacheDir,
		Hostname:       rss.serverHost,
		CacheArgs: kopiacmd.CacheArgs{
			ContentCacheLimitMB:  contentCacheMB,
			MetadataCacheLimitMB: metadataCacheMB,
		},
		RepoPathPrefix: rss.repoPathPrefix,
		Username:       rss.repositoryUser,
		Location:       testutil.GetDefaultS3CompliantStorageLocation(),
	}
	// First try to connect with Kopia Repository
	c.Log("Connecting with Kopia Repository...")
	repoConnectCmd, err := kopiacmd.RepositoryConnectCommand(repoCommandArgs)
	c.Assert(err, IsNil)
	_, err = ExecCommand(c, repoConnectCmd...)
	if err != nil && strings.Contains(err.Error(), "error connecting to repository") {
		// If connection fails, create Kopia Repository
		c.Log("Creating Kopia Repository...")
		repoCreateCmd, err := kopiacmd.RepositoryCreateCommand(repoCommandArgs)
		c.Assert(err, IsNil)
		_, err = ExecCommand(c, repoCreateCmd...)
		c.Assert(err, IsNil)
	}

	// Setting Up Kopia Repository Server
	tlsCertFile := rss.tlsDir + ".cert"
	tlsKeyFile := rss.tlsDir + ".key"
	serverStartCommandArgs := kopiacmd.ServerStartCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   "",
			ConfigFilePath: rss.kopiaConfigDir,
			LogDirectory:   rss.kopiaLogDir,
		},
		ServerAddress:    rss.address,
		TLSCertFile:      tlsCertFile,
		TLSKeyFile:       tlsKeyFile,
		ServerUsername:   rss.serverUsername,
		ServerPassword:   rss.serverPassword,
		AutoGenerateCert: true,
		Background:       true,
	}
	serverStartCmd := kopiacmd.ServerStart(serverStartCommandArgs)
	_, err = ExecCommand(c, serverStartCmd...)
	c.Assert(err, IsNil)

	// Adding Users to Kopia Repository Server
	serverAddUserCommandArgs := kopiacmd.ServerAddUserCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   rss.repositoryPassword,
			ConfigFilePath: rss.kopiaConfigDir,
			LogDirectory:   rss.kopiaLogDir,
		},
		NewUsername:  fmt.Sprintf("%s@%s", rss.testUsername, rss.serverHost),
		UserPassword: rss.testUserPassword,
	}
	serverAddUserCmd := kopiacmd.ServerAddUser(serverAddUserCommandArgs)
	_, err = ExecCommand(c, serverAddUserCmd...)
	c.Assert(err, IsNil)

	// Getting Fingerprint of Kopia Repository Server
	rss.fingerprint = fingerprintFromTLSCert(c, tlsCertFile)
	c.Assert(rss.fingerprint, Not(Equals), "")

	// Refreshing Kopia Repository Server
	serverRefreshCommandArgs := kopiacmd.ServerRefreshCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   rss.repositoryPassword,
			ConfigFilePath: rss.kopiaConfigDir,
			LogDirectory:   rss.kopiaLogDir,
		},
		ServerAddress:  rss.address,
		ServerUsername: rss.serverUsername,
		ServerPassword: rss.serverPassword,
		Fingerprint:    rss.fingerprint,
	}
	serverRefreshCmd := kopiacmd.ServerRefresh(serverRefreshCommandArgs)
	_, err = ExecCommand(c, serverRefreshCmd...)
	c.Assert(err, IsNil)

	// Check Server Status
	serverStatusCommandArgs := kopiacmd.ServerStatusCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   rss.repositoryPassword,
			ConfigFilePath: rss.kopiaConfigDir,
			LogDirectory:   rss.kopiaLogDir,
		},
		ServerAddress:  rss.address,
		ServerUsername: rss.serverUsername,
		ServerPassword: rss.serverPassword,
		Fingerprint:    rss.fingerprint,
	}
	serverStatusCmd := kopiacmd.ServerStatus(serverStatusCommandArgs)
	out, err := ExecCommand(c, serverStatusCmd...)
	if !strings.Contains(out, "IDLE") && out != "" {
		c.Fail()
	}
	c.Assert(err, IsNil)
}

func (rss *RepositoryServerSuite) connectWithTestKopiaRepositoryServer(c *C) error {
	// Connect With Kopia Repository Server
	tlsCertFile := rss.tlsDir + ".cert"
	tlsCertStr := readTLSCert(c, tlsCertFile)
	c.Assert(tlsCertStr, Not(Equals), "")
	contentCacheMB, metadataCacheMB := kopiacmd.GetGeneralCacheSizeSettings()
	return repository.ConnectToAPIServer(
		rss.ctx,
		tlsCertStr,
		rss.testUserPassword,
		rss.serverHost,
		rss.address,
		rss.testUsername,
		contentCacheMB,
		metadataCacheMB,
	)
}

func (rss *RepositoryServerSuite) TestLocationOperationsForRepositoryServerDataMover(c *C) {
	// Setup Kopia Repository Server
	rss.setupKopiaRepositoryServer(c)

	// Setup Test Data
	sourceDir := c.MkDir()
	filePath := filepath.Join(sourceDir, "test.txt")

	cmd := exec.Command("touch", filePath)
	_, err := cmd.Output()
	c.Assert(err, IsNil)

	targetDir := c.MkDir()

	// Connect with Kopia Repository Server
	err = rss.connectWithTestKopiaRepositoryServer(c)
	c.Assert(err, IsNil)

	// Test Kopia Repository Server Location Push
	snapInfo, err := kopiaLocationPush(rss.ctx, rss.repoPathPrefix, "kandoOutput", sourceDir, rss.testUserPassword)
	c.Assert(err, IsNil)

	// Test Kopia Repository Server Location Pull
	err = kopiaLocationPull(rss.ctx, snapInfo.ID, rss.repoPathPrefix, targetDir, rss.testUserPassword)
	c.Assert(err, IsNil)

	// TODO : Verify Data is Pulled from the Location (Issue #2230)

	// Test Kopia Repository Location Delete
	err = kopiaLocationDelete(rss.ctx, snapInfo.ID, rss.repoPathPrefix, rss.testUserPassword)
	c.Assert(err, IsNil)

	// Verify Data is Deleted from the Location
	// Expect an Error while Pulling Data
	err = kopiaLocationPull(rss.ctx, snapInfo.ID, rss.repoPathPrefix, targetDir, rss.testUserPassword)
	c.Assert(err, NotNil)
}
