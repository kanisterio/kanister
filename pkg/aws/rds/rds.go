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

package rds

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
)

const (
	maxRetries                  = 10
	rdsReadyTimeout             = 20 * time.Minute
	dbAurora        RDSDBEngine = "Aurora"
)

// RDS is a wrapper around ec2.RDS structs
type RDS struct {
	*rds.RDS
	dbEngine RDSDBEngine
}

// NewRDSClient returns ec2 client struct.
func NewClient(ctx context.Context, awsConfig *aws.Config, region string, dbEngine RDSDBEngine) (*RDS, error) {
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	return &RDS{
		RDS:      rds.New(s, awsConfig.WithMaxRetries(maxRetries).WithRegion(region).WithCredentials(awsConfig.Credentials)),
		dbEngine: dbEngine,
	}, nil
}

// CreateDBInstanceWithContext
func (r RDS) CreateDBInstance(ctx context.Context, storage int64, instanceClass, instanceID, engine, username, password string, sgIDs []string) (*rds.CreateDBInstanceOutput, error) {
	dbi := &rds.CreateDBInstanceInput{
		AllocatedStorage:     &storage,
		DBInstanceIdentifier: &instanceID,
		VpcSecurityGroupIds:  convertSGIDs(sgIDs),
		DBInstanceClass:      &instanceClass,
		Engine:               &engine,
		MasterUsername:       &username,
		MasterUserPassword:   &password,
	}
	return r.CreateDBInstanceWithContext(ctx, dbi)
}

func (r RDS) WaitUntilDBInstanceAvailable(ctx context.Context, instanceID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	dba := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}
	return r.WaitUntilDBInstanceAvailableWithContext(ctx, dba)
}

func (r RDS) WaitUntilDBInstanceDeleted(ctx context.Context, instanceID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	dba := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}
	return r.WaitUntilDBInstanceDeletedWithContext(ctx, dba)
}

func (r RDS) DescribeDBInstances(ctx context.Context, instanceID string) (*rds.DescribeDBInstancesOutput, error) {
	dbi := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}
	return r.DescribeDBInstancesWithContext(ctx, dbi)
}

func (r RDS) DeleteDBInstance(ctx context.Context, instanceID string) (*rds.DeleteDBInstanceOutput, error) {
	skipSnapshot := true
	dbi := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: &instanceID,
		SkipFinalSnapshot:    &skipSnapshot,
	}
	return r.DeleteDBInstanceWithContext(ctx, dbi)
}

func (r RDS) CreateDBSnapshot(ctx context.Context, instanceID, snapshotID string) (*rds.CreateDBSnapshotOutput, error) {
	sni := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: &instanceID,
		DBSnapshotIdentifier: &snapshotID,
	}
	if r.dbEngine == dbAurora {
		return r.CreateDBClusterSnapshotWithContext(ctx, sni)
	}
	return r.CreateDBSnapshotWithContext(ctx, sni)
}

func (r RDS) WaitUntilDBSnapshotAvailable(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	sni := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.WaitUntilDBSnapshotAvailableWithContext(ctx, sni)
}

func (r RDS) DeleteDBSnapshot(ctx context.Context, snapshotID string) (*rds.DeleteDBSnapshotOutput, error) {
	sni := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.DeleteDBSnapshotWithContext(ctx, sni)
}

func (r RDS) WaitUntilDBSnapshotDeleted(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	sni := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.WaitUntilDBSnapshotDeletedWithContext(ctx, sni)
}

func (r RDS) RestoreDBInstanceFromDBSnapshot(ctx context.Context, instanceID, snapshotID string, sgIDs []string) (*rds.RestoreDBInstanceFromDBSnapshotOutput, error) {
	rdbi := &rds.RestoreDBInstanceFromDBSnapshotInput{
		DBInstanceIdentifier: &instanceID,
		DBSnapshotIdentifier: &snapshotID,
		VpcSecurityGroupIds:  convertSGIDs(sgIDs),
	}
	return r.RestoreDBInstanceFromDBSnapshotWithContext(ctx, rdbi)
}

func convertSGIDs(sgIDs []string) []*string {
	var refSGIDs []*string
	for _, ID := range sgIDs {
		refSGIDs = append(refSGIDs, &ID)
	}
	return refSGIDs
}
