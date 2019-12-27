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

// EC2 is a wrapper around ec2.EC2 structs
type EC2 struct {
	*ec2.EC2
	DryRun bool
}

// NewEC2Client returns ec2 client struct.
func NewClient(ctx context.Context, awsConfig *aws.Config) (*EC2, error) {
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	return &EC2{EC2: ec2.New(s, awsConfig)}, nil
}

func (e EC2) DescribeSecurityGroup(ctx context.Context, groupName string) (*ec2.DescribeSecurityGroupsOutput, error) {
	sgi := &ec2.DescribeSecurityGroupsInput{
		DryRun:     &e.DryRun,
		GroupNames: []*string{&groupName},
	}
	return e.DescribeSecurityGroupsWithContext(ctx, sgi)
}

func (e EC2) CreateSecurityGroup(ctx context.Context, groupName, description string) (*ec2.CreateSecurityGroupOutput, error) {
	sgi := &ec2.CreateSecurityGroupInput{
		DryRun:      &e.DryRun,
		Description: &description,
		GroupName:   &groupName,
	}
	return e.CreateSecurityGroupWithContext(ctx, sgi)
}

func (e EC2) AuthorizeSecurityGroupIngress(ctx context.Context, groupName, cidr, protocol string, port int64) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	sgi := &ec2.AuthorizeSecurityGroupIngressInput{
		DryRun:     &e.DryRun,
		GroupName:  &groupName,
		CidrIp:     &cidr,
		IpProtocol: &protocol,
		ToPort:     &port,
		FromPort:   &port,
	}
	return e.AuthorizeSecurityGroupIngressWithContext(ctx, sgi)
}

func (e EC2) DeleteSecurityGroup(ctx context.Context, groupName string) (*ec2.DeleteSecurityGroupOutput, error) {
	sgi := &ec2.DeleteSecurityGroupInput{
		DryRun:    &e.DryRun,
		GroupName: &groupName,
	}
	return e.DeleteSecurityGroupWithContext(ctx, sgi)
}
