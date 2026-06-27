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

// Package rds provides utilities for managing AWS RDS resources, such as creating, deleting,
// and describing RDS instances, clusters, snapshots, and related operations.
package rds

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/kanisterio/errkit"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	maxRetries               = 10
	rdsReadyTimeout          = 20 * time.Minute
	statusDBClusterAvailable = "available"
)

// RDS is a wrapper around the AWS RDS client
type RDS struct {
	client *rds.Client
}

// NewClient returns an RDS client struct.
func NewClient(ctx context.Context, awsConfig aws.Config, region string) (*RDS, error) {
	awsConfig.Region = region
	client := rds.NewFromConfig(awsConfig, func(o *rds.Options) {
		o.RetryMaxAttempts = maxRetries
	})
	return &RDS{client: client}, nil
}

// CreateDBInstance creates an RDS DB instance with context
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
		if storage != nil {
			s := int32(*storage)
			dbi.AllocatedStorage = &s
		}
		dbi.VpcSecurityGroupIds = sgIDs
		dbi.MasterUsername = aws.String(username)
		dbi.MasterUserPassword = aws.String(password)
		dbi.PubliclyAccessible = publicAccess
	}
	return r.client.CreateDBInstance(ctx, dbi)
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
		VpcSecurityGroupIds: sgIDs,
	}
	return r.client.CreateDBCluster(ctx, dbi)
}

func (r RDS) WaitUntilDBInstanceAvailable(ctx context.Context, instanceID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	waiter := rds.NewDBInstanceAvailableWaiter(r.client)
	return waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}, rdsReadyTimeout)
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
	descCluster, err := r.client.DescribeDBClusters(ctx, dci)
	if err != nil {
		return err
	}

	if aws.ToString(descCluster.DBClusters[0].Status) == status {
		return nil
	}
	return errkit.New(fmt.Sprintf("DBCluster is not in %s state", status))
}

func (r RDS) WaitUntilDBClusterDeleted(ctx context.Context, dbClusterID string) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer waitCancel()
	return poll.Wait(timeoutCtx, func(c context.Context) (bool, error) {
		dci := &rds.DescribeDBClustersInput{
			DBClusterIdentifier: &dbClusterID,
		}
		if _, err := r.client.DescribeDBClusters(ctx, dci); err != nil {
			var notFound *rdstypes.DBClusterNotFoundFault
			if errors.As(err, &notFound) {
				return true, nil
			}
			return false, nil
		}

		return false, nil
	})
}

func (r RDS) WaitUntilDBInstanceDeleted(ctx context.Context, instanceID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	waiter := rds.NewDBInstanceDeletedWaiter(r.client)
	return waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}, rdsReadyTimeout)
}

func (r RDS) DescribeDBInstances(ctx context.Context, instanceID string) (*rds.DescribeDBInstancesOutput, error) {
	dbi := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceID,
	}
	return r.client.DescribeDBInstances(ctx, dbi)
}

func (r RDS) DescribeDBClusters(ctx context.Context, instanceID string) (*rds.DescribeDBClustersOutput, error) {
	dci := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: &instanceID,
	}
	return r.client.DescribeDBClusters(ctx, dci)
}

func (r RDS) DescribeDBClustersSnapshot(ctx context.Context, snapshotID string) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	i := &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.client.DescribeDBClusterSnapshots(ctx, i)
}

func (r RDS) DeleteDBInstance(ctx context.Context, instanceID string) (*rds.DeleteDBInstanceOutput, error) {
	skipSnapshot := true
	dbi := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: &instanceID,
		SkipFinalSnapshot:    &skipSnapshot,
	}
	return r.client.DeleteDBInstance(ctx, dbi)
}

func (r RDS) DeleteDBCluster(ctx context.Context, instanceID string) (*rds.DeleteDBClusterOutput, error) {
	skipSnapshot := true
	ddbc := &rds.DeleteDBClusterInput{
		DBClusterIdentifier: &instanceID,
		SkipFinalSnapshot:   &skipSnapshot,
	}
	return r.client.DeleteDBCluster(ctx, ddbc)
}

func (r RDS) CreateDBSubnetGroup(ctx context.Context, dbSubnetGroupName, dbSubnetGroupDescription string, subnetIDs []string) (*rds.CreateDBSubnetGroupOutput, error) {
	dbsgi := &rds.CreateDBSubnetGroupInput{
		DBSubnetGroupName:        aws.String(dbSubnetGroupName),
		DBSubnetGroupDescription: aws.String(dbSubnetGroupDescription),
		SubnetIds:                subnetIDs,
	}
	return r.client.CreateDBSubnetGroup(ctx, dbsgi)
}

func (r RDS) DeleteDBSubnetGroup(ctx context.Context, dbSubnetGroupName string) (*rds.DeleteDBSubnetGroupOutput, error) {
	dbsgi := &rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: aws.String(dbSubnetGroupName),
	}
	return r.client.DeleteDBSubnetGroup(ctx, dbsgi)
}

func (r RDS) DescribeDBSnapshot(ctx context.Context, snapshotID string) (*rds.DescribeDBSnapshotsOutput, error) {
	dsi := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.client.DescribeDBSnapshots(ctx, dsi)
}

func (r RDS) CreateDBSnapshot(ctx context.Context, instanceID, snapshotID string) (*rds.CreateDBSnapshotOutput, error) {
	sni := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: &instanceID,
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.client.CreateDBSnapshot(ctx, sni)
}

func (r RDS) CreateDBClusterSnapshot(ctx context.Context, clusterID, snapshotID string) (*rds.CreateDBClusterSnapshotOutput, error) {
	csni := &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         &clusterID,
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.client.CreateDBClusterSnapshot(ctx, csni)
}

func (r RDS) WaitUntilDBSnapshotAvailable(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	waiter := rds.NewDBSnapshotAvailableWaiter(r.client)
	return waiter.Wait(ctx, &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}, rdsReadyTimeout)
}

func (r RDS) WaitUntilDBClusterSnapshotAvailable(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	waiter := rds.NewDBClusterSnapshotAvailableWaiter(r.client)
	return waiter.Wait(ctx, &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}, rdsReadyTimeout)
}

func (r RDS) DeleteDBSnapshot(ctx context.Context, snapshotID string) (*rds.DeleteDBSnapshotOutput, error) {
	sni := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: &snapshotID,
	}
	return r.client.DeleteDBSnapshot(ctx, sni)
}

func (r RDS) DeleteDBClusterSnapshot(ctx context.Context, snapshotID string) (*rds.DeleteDBClusterSnapshotOutput, error) {
	dci := &rds.DeleteDBClusterSnapshotInput{
		DBClusterSnapshotIdentifier: &snapshotID,
	}
	return r.client.DeleteDBClusterSnapshot(ctx, dci)
}

func (r RDS) WaitUntilDBSnapshotDeleted(ctx context.Context, snapshotID string) error {
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	waiter := rds.NewDBSnapshotDeletedWaiter(r.client)
	return waiter.Wait(ctx, &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: &snapshotID,
	}, rdsReadyTimeout)
}

func (r RDS) WaitUntilDBClusterSnapshotDeleted(ctx context.Context, snapshotID string) error {
	// No pre-built DBClusterSnapshotDeleted waiter in v2; poll manually.
	ctx, cancel := context.WithTimeout(ctx, rdsReadyTimeout)
	defer cancel()
	return poll.Wait(ctx, func(c context.Context) (bool, error) {
		_, err := r.client.DescribeDBClusterSnapshots(c, &rds.DescribeDBClusterSnapshotsInput{
			DBClusterSnapshotIdentifier: &snapshotID,
		})
		if err != nil {
			var notFound *rdstypes.DBClusterSnapshotNotFoundFault
			if errors.As(err, &notFound) {
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func (r RDS) RestoreDBInstanceFromDBSnapshot(ctx context.Context, instanceID, subnetGroupName, snapshotID string, sgIDs []string) (*rds.RestoreDBInstanceFromDBSnapshotOutput, error) {
	rdbi := &rds.RestoreDBInstanceFromDBSnapshotInput{
		DBInstanceIdentifier: &instanceID,
		DBSnapshotIdentifier: &snapshotID,
		DBSubnetGroupName:    &subnetGroupName,
		VpcSecurityGroupIds:  sgIDs,
	}
	return r.client.RestoreDBInstanceFromDBSnapshot(ctx, rdbi)
}

func (r RDS) RestoreDBClusterFromDBSnapshot(ctx context.Context, instanceID, dbSubnetGroup, snapshotID, dbEngine, version string, sgIDs []string) (*rds.RestoreDBClusterFromSnapshotOutput, error) {
	rdi := &rds.RestoreDBClusterFromSnapshotInput{
		Engine:              &dbEngine,
		EngineVersion:       &version,
		DBClusterIdentifier: &instanceID,
		SnapshotIdentifier:  &snapshotID,
		DBSubnetGroupName:   &dbSubnetGroup,
	}
	if sgIDs != nil {
		rdi.VpcSecurityGroupIds = sgIDs
	}
	return r.client.RestoreDBClusterFromSnapshot(ctx, rdi)
}
