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

package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
)

const (
	maxRetries = 10
)

// EC2 is a wrapper around ec2.EC2 structs
type EC2 struct {
	*ec2.EC2
	DryRun bool
}

// NewClient returns ec2 client struct.
func NewClient(ctx context.Context, awsConfig *aws.Config, region string) (*EC2, error) {
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	return &EC2{EC2: ec2.New(s, awsConfig.WithMaxRetries(maxRetries).WithRegion(region).WithCredentials(awsConfig.Credentials))}, nil
}

func (e EC2) DescribeSecurityGroup(ctx context.Context, groupName string) (*ec2.DescribeSecurityGroupsOutput, error) {
	sgi := &ec2.DescribeSecurityGroupsInput{
		DryRun:     &e.DryRun,
		GroupNames: []*string{&groupName},
	}
	return e.DescribeSecurityGroupsWithContext(ctx, sgi)
}

func (e EC2) CreateSecurityGroup(ctx context.Context, groupName, description, vpcID string) (*ec2.CreateSecurityGroupOutput, error) {
	sgi := &ec2.CreateSecurityGroupInput{
		DryRun:      &e.DryRun,
		Description: &description,
		GroupName:   &groupName,
		VpcId:       aws.String(vpcID),
	}
	return e.CreateSecurityGroupWithContext(ctx, sgi)
}

func (e EC2) AuthorizeSecurityGroupIngress(ctx context.Context, groupID, cidr, protocol string, port int64) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	sgi := &ec2.AuthorizeSecurityGroupIngressInput{
		DryRun:     &e.DryRun,
		GroupId:    &groupID,
		CidrIp:     &cidr,
		IpProtocol: &protocol,
		ToPort:     &port,
		FromPort:   &port,
	}
	return e.AuthorizeSecurityGroupIngressWithContext(ctx, sgi)
}

func (e EC2) DeleteSecurityGroup(ctx context.Context, groupID string) (*ec2.DeleteSecurityGroupOutput, error) {
	sgi := &ec2.DeleteSecurityGroupInput{
		DryRun:  &e.DryRun,
		GroupId: aws.String(groupID),
	}
	return e.DeleteSecurityGroupWithContext(ctx, sgi)
}

func (e EC2) DescribeSubnets(ctx context.Context, vpcID string) (*ec2.DescribeSubnetsOutput, error) {
	paramsEC2 := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
		},
	}
	return e.DescribeSubnetsWithContext(ctx, paramsEC2)
}

func (e EC2) DescribeDefaultVpc(ctx context.Context) (*ec2.DescribeVpcsOutput, error) {
	vpci := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("isDefault"),
				Values: []*string{
					aws.String("true"),
				},
			},
		},
	}
	return e.DescribeVpcsWithContext(ctx, vpci)
}
