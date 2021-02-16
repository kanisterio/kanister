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
	maxRetries      = 10
	rdsReadyTimeout = 20 * time.Minute
)

// RDS is a wrapper around ec2.RDS structs
type RDS struct {
	*rds.RDS
}

// NewRDSClient returns ec2 client struct.
func NewClient(ctx context.Context, awsConfig *aws.Config, region string) (*RDS, error) {
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	return &RDS{RDS: rds.New(s, awsConfig.WithMaxRetries(maxRetries).WithRegion(region).WithCredentials(awsConfig.Credentials))}, nil
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

func (r RDS) CreateDBInstanceInCluster(ctx context.Context, restoredClusterID string) (*rds.CreateDBInstanceOutput, error) {
	dbi := &rds.CreateDBInstanceInput{
		DBClusterIdentifier: &restoredClusterID,
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

func (r RDS) DescribeDBClusters(ctx context.Context, instanceID string) (*rds.DescribeDBClustersOutput, error) {
	dci := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: &instanceID,
	}
	return r.DescribeDBClustersWithContext(ctx, dci)
}

func (r RDS) DeleteDBInstance(ctx context.Context, instanceID string) (*rds.DeleteDBInstanceOutput, error) {
	skipSnapshot := true
	dbi := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: &instanceID,
		SkipFinalSnapshot:    &skipSnapshot,
	}
	return r.DeleteDBInstanceWithContext(ctx, dbi)
}

func (r RDS) DeleteDBCluster(ctx context.Context, instanceID string) (*rds.DeleteDBClusterOutput, error) {
	skipSnapshot := true
	ddbc := &rds.DeleteDBClusterInput{
		DBClusterIdentifier: &instanceID,
		SkipFinalSnapshot:   &skipSnapshot,
	}
	return r.DeleteDBClusterWithContext(ctx, ddbc)
}

func (r RDS) CreateDBSnapshot(ctx context.Context, instanceID, snapshotID string) (*rds.CreateDBSnapshotOutput, error) {
	sni := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: &instanceID,
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.CreateDBSnapshotWithContext(ctx, sni)
}

func (r RDS) CreateDBClusterSnapshot(ctx context.Context, clusterID, snapshotID string) (*rds.CreateDBClusterSnapshotOutput, error) {
	csni := &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         &clusterID,
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.CreateDBClusterSnapshotWithContext(ctx, csni)
}

func (r RDS) WaitUntilDBSnapshotAvailable(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	sni := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.WaitUntilDBSnapshotAvailableWithContext(ctx, sni)
}

func (r RDS) WaitUntilDBClusterSnapshotAvailable(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	dsni := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.WaitUntilDBClusterSnapshotAvailableWithContext(ctx, dsni)
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

func (r RDS) RestoreDBClusterFromDBSnapshot(ctx context.Context, instanceID, snapshotID string, sgIDs []string) (*rds.RestoreDBClusterFromSnapshotOutput, error) {
	engine := "aurora"
	rdi := &rds.RestoreDBClusterFromSnapshotInput{
		Engine:              &engine,
		DBClusterIdentifier: &instanceID,
		SnapshotIdentifier:  &snapshotID,
		VpcSecurityGroupIds: convertSGIDs(sgIDs),
	}
	return r.RestoreDBClusterFromSnapshotWithContext(ctx, rdi)
}

func convertSGIDs(sgIDs []string) []*string {
	var refSGIDs []*string
	for _, ID := range sgIDs {
		refSGIDs = append(refSGIDs, &ID)
	}
	return refSGIDs
}
