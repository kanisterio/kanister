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
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	maxRetries               = 10
	rdsReadyTimeout          = 20 * time.Minute
	statusDBClusterAvailable = "available"
)

// RDS is a wrapper around ec2.RDS structs
type RDS struct {
	*rds.RDS
}

// NewClient returns ec2 client struct.
func NewClient(ctx context.Context, awsConfig *aws.Config, region string) (*RDS, error) {
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	return &RDS{RDS: rds.New(s, awsConfig.WithMaxRetries(maxRetries).WithRegion(region).WithCredentials(awsConfig.Credentials))}, nil
}

// CreateDBInstance return DBInstance with context
func (r RDS) CreateDBInstance(
	ctx context.Context,
	storage *int64,
	instanceClass,
	instanceID,
	engine,
	username,
	password string,
	sgIDs []string,
	publicAccess *bool,
	restoredClusterID *string,
	dbSubnetGroup string,
) (*rds.CreateDBInstanceOutput, error) {
	dbi := &rds.CreateDBInstanceInput{
		DBInstanceClass:      &instanceClass,
		DBInstanceIdentifier: &instanceID,
		Engine:               &engine,
		DBSubnetGroupName:    aws.String(dbSubnetGroup),
	}

	// check if the instance is being restored from an existing cluster
	switch {
	case restoredClusterID != nil && publicAccess != nil:
		dbi.DBClusterIdentifier = restoredClusterID
		dbi.PubliclyAccessible = publicAccess
	case restoredClusterID != nil && publicAccess == nil:
		dbi.DBClusterIdentifier = restoredClusterID
	default:
		// if not restoring from an existing cluster, create a new instance input
		dbi.AllocatedStorage = storage
		dbi.VpcSecurityGroupIds = convertSGIDs(sgIDs)
		dbi.MasterUsername = aws.String(username)
		dbi.MasterUserPassword = aws.String(password)
		dbi.PubliclyAccessible = publicAccess
	}
	return r.CreateDBInstanceWithContext(ctx, dbi)
}

func (r RDS) CreateDBCluster(
	ctx context.Context,
	storage int64,
	instanceClass,
	instanceID,
	dbSubnetGroup,
	engine,
	dbName,
	username,
	password string,
	sgIDs []string,
) (*rds.CreateDBClusterOutput, error) {
	dbi := &rds.CreateDBClusterInput{
		DBClusterIdentifier: &instanceID,
		DatabaseName:        &dbName,
		DBSubnetGroupName:   aws.String(dbSubnetGroup),
		Engine:              &engine,
		MasterUsername:      &username,
		MasterUserPassword:  &password,
		VpcSecurityGroupIds: convertSGIDs(sgIDs),
	}
	return r.CreateDBClusterWithContext(ctx, dbi)
}

func (r RDS) WaitUntilDBInstanceAvailable(ctx context.Context, instanceID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	dba := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}
	return r.WaitUntilDBInstanceAvailableWithContext(ctx, dba)
}

func (r RDS) WaitUntilDBClusterAvailable(ctx context.Context, dbClusterID string) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer waitCancel()
	return poll.Wait(timeoutCtx, func(c context.Context) (bool, error) {
		err := r.WaitOnDBCluster(ctx, dbClusterID, statusDBClusterAvailable)
		return err == nil, nil
	})
}

// WaitOnDBCluster waits for DB cluster with instanceID
func (r RDS) WaitOnDBCluster(ctx context.Context, dbClusterID, status string) error {
	// describe the cluster return err if status is !Available
	dci := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: &dbClusterID,
	}
	descCluster, err := r.DescribeDBClustersWithContext(ctx, dci)
	if err != nil {
		return err
	}

	if *descCluster.DBClusters[0].Status == status {
		return nil
	}
	return errors.New(fmt.Sprintf("DBCluster is not in %s state", status))
}

func (r RDS) WaitUntilDBClusterDeleted(ctx context.Context, dbClusterID string) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer waitCancel()
	return poll.Wait(timeoutCtx, func(c context.Context) (bool, error) {
		dci := &rds.DescribeDBClustersInput{
			DBClusterIdentifier: &dbClusterID,
		}
		if _, err := r.DescribeDBClustersWithContext(ctx, dci); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == rds.ErrCodeDBClusterNotFoundFault {
					return true, nil
				}
				return false, nil
			}
		}

		return false, nil
	})
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

func (r RDS) DescribeDBClustersSnapshot(ctx context.Context, snapshotID string) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	i := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.DescribeDBClusterSnapshotsWithContext(ctx, i)
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

func (r RDS) CreateDBSubnetGroup(ctx context.Context, dbSubnetGroupName, dbSubnetGroupDescription string, subnetIDs []string) (*rds.CreateDBSubnetGroupOutput, error) {
	var subnetIds []*string
	for _, ID := range subnetIDs {
		subnetIds = append(subnetIds, aws.String(ID))
	}
	dbsgi := &rds.CreateDBSubnetGroupInput{
		DBSubnetGroupName:        aws.String(dbSubnetGroupName),
		DBSubnetGroupDescription: aws.String(dbSubnetGroupDescription),
		SubnetIds:                subnetIds,
	}
	return r.CreateDBSubnetGroupWithContext(ctx, dbsgi)
}

func (r RDS) DeleteDBSubnetGroup(ctx context.Context, dbSubnetGroupName string) (*rds.DeleteDBSubnetGroupOutput, error) {
	dbsgi := &rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: aws.String(dbSubnetGroupName),
	}
	return r.DeleteDBSubnetGroupWithContext(ctx, dbsgi)
}

func (r RDS) DescribeDBSnapshot(ctx context.Context, snapshotID string) (*rds.DescribeDBSnapshotsOutput, error) {
	dsi := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.DescribeDBSnapshotsWithContext(ctx, dsi)
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

func (r RDS) DeleteDBClusterSnapshot(ctx context.Context, snapshotID string) (*rds.DeleteDBClusterSnapshotOutput, error) {
	dci := &rds.DeleteDBClusterSnapshotInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.DeleteDBClusterSnapshotWithContext(ctx, dci)
}

func (r RDS) WaitUntilDBSnapshotDeleted(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	sni := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.WaitUntilDBSnapshotDeletedWithContext(ctx, sni)
}

func (r RDS) WaitUntilDBClusterSnapshotDeleted(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	sdi := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.WaitUntilDBClusterSnapshotDeletedWithContext(ctx, sdi)
}

func (r RDS) RestoreDBInstanceFromDBSnapshot(ctx context.Context, instanceID, subnetGroupName, snapshotID string, sgIDs []string) (*rds.RestoreDBInstanceFromDBSnapshotOutput, error) {
	rdbi := &rds.RestoreDBInstanceFromDBSnapshotInput{
		DBInstanceIdentifier: &instanceID,
		DBSnapshotIdentifier: &snapshotID,
		DBSubnetGroupName:    &subnetGroupName,
		VpcSecurityGroupIds:  convertSGIDs(sgIDs),
	}
	return r.RestoreDBInstanceFromDBSnapshotWithContext(ctx, rdbi)
}

func (r RDS) RestoreDBClusterFromDBSnapshot(ctx context.Context, instanceID, dbSubnetGroup, snapshotID, dbEngine, version string, sgIDs []string) (*rds.RestoreDBClusterFromSnapshotOutput, error) {
	var rdi *rds.RestoreDBClusterFromSnapshotInput
	if sgIDs == nil {
		rdi = &rds.RestoreDBClusterFromSnapshotInput{
			Engine:              &dbEngine,
			EngineVersion:       &version,
			DBClusterIdentifier: &instanceID,
			SnapshotIdentifier:  &snapshotID,
			DBSubnetGroupName:   &dbSubnetGroup,
		}
	} else {
		rdi = &rds.RestoreDBClusterFromSnapshotInput{
			Engine:              &dbEngine,
			EngineVersion:       &version,
			DBClusterIdentifier: &instanceID,
			SnapshotIdentifier:  &snapshotID,
			VpcSecurityGroupIds: convertSGIDs(sgIDs),
			DBSubnetGroupName:   &dbSubnetGroup,
		}
	}
	return r.RestoreDBClusterFromSnapshotWithContext(ctx, rdi)
}

func convertSGIDs(sgIDs []string) []*string {
	var refSGIDs []*string
	for _, ID := range sgIDs {
		idPtr := &ID
		refSGIDs = append(refSGIDs, idPtr)
	}
	return refSGIDs
}
