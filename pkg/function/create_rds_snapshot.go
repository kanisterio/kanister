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
	"time"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&createRDSSnapshotFunc{})
}

var (
	_ kanister.Func = (*createRDSSnapshotFunc)(nil)
)

const (
	// CreateVolumeFromSnapshotFuncName gives the name of the function
	CreateRDSSnapshotFuncName = "CreateRDSSnapshot"
	// CreateRDSSnapshotInstanceIDArg provides rds instance ID
	CreateRDSSnapshotInstanceIDArg = "instanceID"
	// CreateRDSSnapshotSecurityGroupIDArg provides RDS instance security group ID
	CreateRDSSnapshotSecurityGroupIDArg = "securityGroupID"
	// RDSSnapshotID provides RDS snapshot ID
	CreateRDSSnapshotSnapshotIDArg = "snapshotID"

	rdsReadyTimeout = 20 * time.Minute
)

type createRDSSnapshotFunc struct{}

func (*createRDSSnapshotFunc) Name() string {
	return CreateRDSSnapshotFuncName
}

func createRDSSnapshot(ctx context.Context, instanceID, sgID, snapshotID string, profile *param.Profile) (map[string]interface{}, error) {
	// Validate profile
	if err := ValidateProfile(profile); err != nil {
		return nil, errors.Wrapf(err, "Profile Validation failed")
	}

	awsConfig, region, err := aws.GetConfigFromProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return nil, err
	}

	// Create Snapshot
	log.Print("Creating RDS snapshot", field.M{"SnapshotID": snapshotID})
	_, err = rdsCli.CreateDBSnapshot(ctx, instanceID, snapshotID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot")
	}
	// Wait until snapshot becomes available
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	log.Print("Waiting for RDS snapshot to be available", field.M{"SnapshotID": snapshotID})
	if err := rdsCli.WaitUntilDBSnapshotAvailable(ctx, snapshotID); err != nil {
		return nil, err
	}

	output := map[string]interface{}{
		CreateRDSSnapshotSnapshotIDArg:      snapshotID,
		CreateRDSSnapshotInstanceIDArg:      instanceID,
		CreateRDSSnapshotSecurityGroupIDArg: sgID,
	}
	return output, nil
}

func (crs *createRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var instanceID, sgID, snapshotID string
	if err := Arg(args, CreateRDSSnapshotInstanceIDArg, &instanceID); err != nil {
		return nil, err
	}
	if err := Arg(args, CreateRDSSnapshotSecurityGroupIDArg, &sgID); err != nil {
		return nil, err
	}
	if err := Arg(args, CreateRDSSnapshotSnapshotIDArg, &snapshotID); err != nil {
		return nil, err
	}
	return createRDSSnapshot(ctx, instanceID, sgID, snapshotID, tp.Profile)
}

func (*createRDSSnapshotFunc) RequiredArgs() []string {
	return []string{CreateRDSSnapshotInstanceIDArg, CreateRDSSnapshotSecurityGroupIDArg, CreateRDSSnapshotSnapshotIDArg}
}
