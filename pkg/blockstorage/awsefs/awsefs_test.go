// Copyright 2021 The Kanister Authors.
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

package awsefs

import (
	"os"

	. "gopkg.in/check.v1"
)

type AWSEFSTestSuite struct{}

var _ = Suite(&AWSEFSTestSuite{})

func (s *AWSEFSTestSuite) TestGetEnvAsStringOrDefault(c *C) {
	tempEnv := os.Getenv(efsBackupVaultNameEnv)
	os.Unsetenv(efsBackupVaultNameEnv)

	vaultName := getEnvAsStringOrDefault(efsBackupVaultNameEnv, defaultK10BackupVaultName)
	c.Assert(vaultName, Equals, defaultK10BackupVaultName)

	os.Setenv(efsBackupVaultNameEnv, "vaultname")
	vaultName = getEnvAsStringOrDefault(efsBackupVaultNameEnv, defaultK10BackupVaultName)
	c.Assert(vaultName, Equals, "vaultname")

	os.Setenv(efsBackupVaultNameEnv, "vaultname")
	vaultName = getEnvAsStringOrDefault("", defaultK10BackupVaultName)
	c.Assert(vaultName, Equals, defaultK10BackupVaultName)

	os.Setenv(efsBackupVaultNameEnv, "vaultname")
	vaultName = getEnvAsStringOrDefault("", "somethingbesidesdefault")
	c.Assert(vaultName, Equals, "somethingbesidesdefault")

	os.Setenv(efsBackupVaultNameEnv, tempEnv)
}
