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
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	kanister "github.com/kanisterio/kanister/pkg"
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
	// CreateRDSSnapshotSnapshotID to set snapshotID in output artifact
	CreateRDSSnapshotSnapshotID = "snapshotID"
	// CreateRDSSnapshotSecurityGroupID to set securityGroupIDs in output artifact
	CreateRDSSnapshotSecurityGroupID = "securityGroupID"
)

type createRDSSnapshotFunc struct{}

func (*createRDSSnapshotFunc) Name() string {
	return CreateRDSSnapshotFuncName
}

func createRDSSnapshot(ctx context.Context, instanceID string, profile *param.Profile) (map[string]interface{}, error) {
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

	// Create Snapshot
	snapshotID := fmt.Sprintf("%s-%s", instanceID, rand.String(10))

	log.Print("Creating RDS snapshot", field.M{"SnapshotID": snapshotID})
	if _, err := rdsCli.CreateDBSnapshot(ctx, instanceID, snapshotID); err != nil {
		return nil, errors.Wrap(err, "Failed to create snapshot")
	}

	// Wait until snapshot becomes available
	log.Print("Waiting for RDS snapshot to be available", field.M{"SnapshotID": snapshotID})
	if err := rdsCli.WaitUntilDBSnapshotAvailable(ctx, snapshotID); err != nil {
		return nil, errors.Wrap(err, "Error while waiting snapshot to be available")
	}

	// Find security group ids
	sgIDs, err := findSecurityGroups(ctx, rdsCli, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
	}

	output := map[string]interface{}{
		CreateRDSSnapshotSnapshotID:      snapshotID,
		CreateRDSSnapshotInstanceIDArg:   instanceID,
		CreateRDSSnapshotSecurityGroupID: sgIDs,
	}
	return output, nil
}

func (crs *createRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var instanceID string
	if err := Arg(args, CreateRDSSnapshotInstanceIDArg, &instanceID); err != nil {
		return nil, err
	}
	return createRDSSnapshot(ctx, instanceID, tp.Profile)
}

func (*createRDSSnapshotFunc) RequiredArgs() []string {
	return []string{CreateRDSSnapshotInstanceIDArg}
}
