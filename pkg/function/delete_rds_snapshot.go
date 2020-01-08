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
	"context"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&deleteRDSSnapshotFunc{})
}

var (
	_ kanister.Func = (*deleteRDSSnapshotFunc)(nil)
)

const (
	// DeleteRDSSnapshotFuncName gives the name of the function
	DeleteRDSSnapshotFuncName      = "DeleteRDSSnapshot"
	DeleteRDSSnapshotSnapshotIDArg = "snapshotID"
)

type deleteRDSSnapshotFunc struct{}

func (*deleteRDSSnapshotFunc) Name() string {
	return DeleteRDSSnapshotFuncName
}

func deleteRDSSnapshot(ctx context.Context, snapshotID string, profile *param.Profile) (map[string]interface{}, error) {
	// Validate profile
	if err := ValidateProfile(profile); err != nil {
		return nil, errors.Wrap(err, "Profile Validation failed")
	}

	// Get aws config from profile
	awsConfig, region, err := getAWSConfigFromProfile(ctx, profile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AWS creds from profile")
	}

	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RDS client")

	}

	// Delete Snapshot
	log.Print("Deleting RDS snapshot", field.M{"SnapshotID": snapshotID})
	if _, err := rdsCli.DeleteDBSnapshot(ctx, snapshotID); err != nil {
		return nil, errors.Wrap(err, "Failed to delete snapshot")
	}

	// Wait until snapshot is deleted
	log.Print("Waiting for RDS snapshot to be deleted", field.M{"SnapshotID": snapshotID})
	err = rdsCli.WaitUntilDBSnapshotDeleted(ctx, snapshotID)
	return nil, errors.Wrap(err, "Error while waiting snapshot to be deleted")
}

func (crs *deleteRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var snapshotID string
	if err := Arg(args, DeleteRDSSnapshotSnapshotIDArg, &snapshotID); err != nil {
		return nil, err
	}
	return deleteRDSSnapshot(ctx, snapshotID, tp.Profile)
}

func (*deleteRDSSnapshotFunc) RequiredArgs() []string {
	return []string{DeleteRDSSnapshotSnapshotIDArg}
}
