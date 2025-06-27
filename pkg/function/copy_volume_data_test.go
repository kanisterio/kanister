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
	mountPoint := "/mnt/vol_data/test-pvc"
	encryptionKey := "test-key"
	insecureTLS := false

	// Test the new backup command with CD functionality
	backupCmd, err := restic.BackupCommandByTagWithCD(profile, targetPath, backupTag, mountPoint, encryptionKey, insecureTLS)
	c.Assert(err, check.IsNil)

	// Verify the full command structure matches shCommand format
	c.Assert(len(backupCmd), check.Equals, 7)
	c.Assert(backupCmd[0], check.Equals, "bash")
	c.Assert(backupCmd[1], check.Equals, "-o")
	c.Assert(backupCmd[2], check.Equals, "errexit")
	c.Assert(backupCmd[3], check.Equals, "-o")
	c.Assert(backupCmd[4], check.Equals, "pipefail")
	c.Assert(backupCmd[5], check.Equals, "-c")

	// Verify the exact command structure and order of operations
	actualCmd := backupCmd[6] // Get the actual command string (last element)
	expectedCmd := "export AWS_ACCESS_KEY_ID=test-id\n export AWS_SECRET_ACCESS_KEY=test-secret\n " +
		"export RESTIC_REPOSITORY=s3:test-endpoint//tmp/test-backup\n export RESTIC_PASSWORD=test-key\n " +
		"cd /mnt/vol_data/test-pvc\n restic backup --tag test-tag ."

	c.Assert(actualCmd, check.Equals, expectedCmd,
		check.Commentf("Expected exact command match:\nExpected: %s\nActual:   %s", expectedCmd, actualCmd))
}
