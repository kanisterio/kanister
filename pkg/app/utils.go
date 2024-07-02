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

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/kanisterio/errkit"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/kanisterio/kanister/pkg/aws/ec2"
	"github.com/kanisterio/kanister/pkg/aws/rds"
)

const (
	dbTemplateURI = "https://raw.githubusercontent.com/openshift/origin/%s/examples/db-templates/%s-%s-template.json"
	// PersistentStorage can be used if we want to deploy database with Persistent Volumes
	PersistentStorage storage = "persistent" //nolint:varcheck

	// EphemeralStorage can be used if we don't want to deploy database with Persistent
	EphemeralStorage storage = "ephemeral"
	// TemplateVersionOCP3_11 stores version of db template 3.11
	TemplateVersionOCP3_11 DBTemplate = "release-3.11"
	// TemplateVersionOCP4_4 stores version of db template 4.4
	TemplateVersionOCP4_4 DBTemplate = "release-4.4"
	// TemplateVersionOCP4_5 stores version of db template 4.5
	TemplateVersionOCP4_5 DBTemplate = "release-4.5"
	// TemplateVersionOCP4_10 stores version of db template 4.10
	TemplateVersionOCP4_10 DBTemplate = "release-4.10"
	// TemplateVersionOCP4_11 stores version of db template 4.11
	TemplateVersionOCP4_11 DBTemplate = "release-4.11"
	// TemplateVersionOCP4_12 stores version of db template 4.12
	TemplateVersionOCP4_12 DBTemplate = "release-4.12"
	// TemplateVersionOCP4_13 stores version of db template 4.13
	TemplateVersionOCP4_13 DBTemplate = "release-4.13"
	// TemplateVersionOCP4_14 stores version of db template 4.14
	TemplateVersionOCP4_14 DBTemplate = "release-4.14"
)

type storage string

// DBTemplate is type of openshift db template version
type DBTemplate string

// appendRandString, appends a random string to the passed string value
func appendRandString(name string) string {
	return fmt.Sprintf("%s-%s", name, rand.String(5))
}

// getOpenShiftDBTemplate accepts the application name and returns the
// db template for that application
// https://github.com/openshift/origin/tree/master/examples/db-templates
func getOpenShiftDBTemplate(appName string, templateVersion DBTemplate, storageType storage) string {
	return fmt.Sprintf(dbTemplateURI, templateVersion, appName, storageType)
}

// getLabelOfApp returns label of the passed application this label can be
// used to delete all the resources that were created while deploying this application
func getLabelOfApp(appName string, storageType storage) string {
	return fmt.Sprintf("app=%s-%s", appName, storageType)
}

// bastionDebugWorkloadSpec creates Deployment Resource Manifest from which RDS database queries can be executed
func bastionDebugWorkloadSpec(ctx context.Context, name string, image string, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    name,
							Image:   image,
							Command: []string{"sh", "-c", "tail -f /dev/null"},
						},
					},
				},
			},
		},
	}
}

// vpcIdForRDSInstance gets the VPC ID from env var `VPC_ID` if set, or from the default VPC
func vpcIDForRDSInstance(ctx context.Context, ec2Cli *ec2.EC2) (string, error) {
	vpcID := os.Getenv("VPC_ID")

	// VPCId is not provided, use Default VPC
	if vpcID != "" {
		return vpcID, nil
	}
	defaultVpc, err := ec2Cli.DescribeDefaultVpc(ctx)
	if err != nil {
		return "", err
	}
	if len(defaultVpc.Vpcs) == 0 {
		return "", errkit.New("No default VPC found")
	}
	return *defaultVpc.Vpcs[0].VpcId, nil
}

// dbSubnetGroup gets the DBSubnetGroup based on VPC ID
func dbSubnetGroup(ctx context.Context, ec2Cli *ec2.EC2, rdsCli *rds.RDS, vpcID, name, subnetGroupDescription string) (string, error) {
	// describe subnets in the VPC
	resp, err := ec2Cli.DescribeSubnets(ctx, vpcID)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to describe subnets")
	}

	// Extract subnet IDs from the response
	var subnetIDs []string
	for _, subnet := range resp.Subnets {
		subnetIDs = append(subnetIDs, *subnet.SubnetId)
	}

	// create a subnetgroup with subnets in the VPC
	subnetGroup, err := rdsCli.CreateDBSubnetGroup(ctx, fmt.Sprintf("%s-subnetgroup", name), subnetGroupDescription, subnetIDs)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to create subnet group")
	}

	return *subnetGroup.DBSubnetGroup.DBSubnetGroupName, nil
}
