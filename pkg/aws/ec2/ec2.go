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

// Package ec2 provides a wrapper around the AWS EC2 SDK to simplify interactions
// with EC2 resources such as security groups, subnets, and VPCs.
package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	maxRetries = 10
)

// EC2 is a wrapper around the AWS EC2 client
type EC2 struct {
	client *ec2.Client
	DryRun bool
}

// NewClient returns an EC2 client struct.
func NewClient(ctx context.Context, awsConfig aws.Config, region string) (*EC2, error) {
	awsConfig.Region = region
	client := ec2.NewFromConfig(awsConfig, func(o *ec2.Options) {
		o.RetryMaxAttempts = maxRetries
	})
	return &EC2{client: client, DryRun: false}, nil
}

func (e EC2) DescribeSecurityGroup(ctx context.Context, groupName string) (*ec2.DescribeSecurityGroupsOutput, error) {
	sgi := &ec2.DescribeSecurityGroupsInput{
		DryRun:     aws.Bool(e.DryRun),
		GroupNames: []string{groupName},
	}
	return e.client.DescribeSecurityGroups(ctx, sgi)
}

func (e EC2) CreateSecurityGroup(ctx context.Context, groupName, description, vpcID string) (*ec2.CreateSecurityGroupOutput, error) {
	sgi := &ec2.CreateSecurityGroupInput{
		DryRun:      aws.Bool(e.DryRun),
		Description: &description,
		GroupName:   &groupName,
		VpcId:       aws.String(vpcID),
	}
	return e.client.CreateSecurityGroup(ctx, sgi)
}

func (e EC2) AuthorizeSecurityGroupIngress(ctx context.Context, groupID, cidr, protocol string, port int64) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	p := int32(port)
	sgi := &ec2.AuthorizeSecurityGroupIngressInput{
		DryRun:     aws.Bool(e.DryRun),
		GroupId:    &groupID,
		CidrIp:     &cidr,
		IpProtocol: &protocol,
		ToPort:     &p,
		FromPort:   &p,
	}
	return e.client.AuthorizeSecurityGroupIngress(ctx, sgi)
}

func (e EC2) DeleteSecurityGroup(ctx context.Context, groupID string) (*ec2.DeleteSecurityGroupOutput, error) {
	sgi := &ec2.DeleteSecurityGroupInput{
		DryRun:  aws.Bool(e.DryRun),
		GroupId: aws.String(groupID),
	}
	return e.client.DeleteSecurityGroup(ctx, sgi)
}

func (e EC2) DescribeSubnets(ctx context.Context, vpcID string) (*ec2.DescribeSubnetsOutput, error) {
	paramsEC2 := &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	}
	return e.client.DescribeSubnets(ctx, paramsEC2)
}

func (e EC2) DescribeDefaultVpc(ctx context.Context) (*ec2.DescribeVpcsOutput, error) {
	vpci := &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []string{"true"},
			},
		},
	}
	return e.client.DescribeVpcs(ctx, vpci)
}
