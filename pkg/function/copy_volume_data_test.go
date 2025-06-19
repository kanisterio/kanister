// Copyright 2019 The Kanister Authors.
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

package function

import (
	"strings"

	"gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

type CopyVolumeDataTestSuite struct{}

var _ = check.Suite(&CopyVolumeDataTestSuite{})

func (s *CopyVolumeDataTestSuite) TestBackupCommandConstruction(c *check.C) {
	// Test that backup command uses relative path "." instead of absolute mount point
	profile := &param.Profile{
		Location: crv1alpha1.Location{
			Type:     crv1alpha1.LocationTypeS3Compliant,
			Bucket:   "test-bucket",
			Endpoint: "test-endpoint",
			Prefix:   "test-prefix",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     "test-id",
				Secret: "test-secret",
			},
		},
	}
	targetPath := "/tmp/test-backup"
	backupTag := "test-tag"
	encryptionKey := "test-key"
	insecureTLS := false

	// Test the backup command generation with relative path
	backupCmd, err := restic.BackupCommandByTag(profile, targetPath, backupTag, ".", encryptionKey, insecureTLS)
	c.Assert(err, check.IsNil)

	// Verify the command contains relative path "." instead of absolute path
	cmdStr := strings.Join(backupCmd, " ")
	c.Assert(strings.Contains(cmdStr, "backup --tag test-tag ."), check.Equals, true)
	c.Assert(strings.Contains(cmdStr, "/mnt/vol_data"), check.Equals, false)
}
