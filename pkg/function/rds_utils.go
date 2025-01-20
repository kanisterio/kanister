package function

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	// CredentialsSourceArg provides a location of AWS credentials to access RDS
	CredentialsSourceArg = "credentialsSource"
	// CredentialsSecretArg provides a secret name to get AWS credentials from
	CredentialsSecretArg = "credentialsSecret"
	// RegionArg specifies (or overrides) an AWS region
	RegionArg = "region"

	CredentialSourceProfile        awsCredentialSourceType = "profile"
	CredentialSourceSecret         awsCredentialSourceType = "secret"
	CredentialSourceServiceAccount awsCredentialSourceType = "eks_service_account"
)

type awsCredentialSourceType string

type awsCredentialsSource struct {
	sourceType awsCredentialSourceType
	secret     string
	region     string
}

func parseCredentialsSource(args map[string]interface{}) (*awsCredentialsSource, error) {
	var sourceType awsCredentialSourceType
	var secret string
	var region string
	if err := OptArg(args, CredentialsSourceArg, &sourceType, CredentialSourceProfile); err != nil {
		return nil, err
	}

	if err := OptArg(args, CredentialsSecretArg, &secret, ""); err != nil {
		return nil, err
	}

	if sourceType == CredentialSourceSecret && secret == "" {
		return nil, errkit.New("credentialsSecret is required if credentialsSource is set to 'secret'")
	}

	if err := OptArg(args, RegionArg, &region, ""); err != nil {
		return nil, err
	}

	return &awsCredentialsSource{sourceType: sourceType, secret: secret, region: region}, nil
}

func getAwsConfig(ctx context.Context, credentialsSource awsCredentialsSource, tp param.TemplateParams) (*awssdk.Config, string, error) {
	switch credentialsSource.sourceType {
	case CredentialSourceProfile:
		profile := tp.Profile
		// Validate profile
		if err := ValidateProfile(profile); err != nil {
			return nil, "", errkit.Wrap(err, "Profile Validation failed")
		}
		// Get aws config from profile
		return getAWSConfigFromProfile(ctx, profile, credentialsSource.region)
	case CredentialSourceSecret:
		if secret, ok := tp.Secrets[credentialsSource.secret]; ok {
			return getAWSConfigFromSecret(ctx, secret, credentialsSource.region)
		}
		return nil, "", errkit.New("Cannot find secret in actionset secrets", "secret", credentialsSource.secret)
	case CredentialSourceServiceAccount:
		return aws.GetConfig(ctx, make(map[string]string))
	default:
		return nil, "", errkit.New("Invalid awsCredentials type", "type", credentialsSource.sourceType)
	}
}

func getAWSConfigFromProfile(ctx context.Context, profile *param.Profile, region string) (*awssdk.Config, string, error) {
	// Validate profile secret
	config := make(map[string]string)
	if profile.Credential.Type == param.CredentialTypeKeyPair {
		config[aws.AccessKeyID] = profile.Credential.KeyPair.ID
		config[aws.SecretAccessKey] = profile.Credential.KeyPair.Secret
	} else if profile.Credential.Type == param.CredentialTypeSecret {
		config[aws.AccessKeyID] = string(profile.Credential.Secret.Data[secrets.AWSAccessKeyID])
		config[aws.SecretAccessKey] = string(profile.Credential.Secret.Data[secrets.AWSSecretAccessKey])
		config[aws.ConfigRole] = string(profile.Credential.Secret.Data[secrets.ConfigRole])
		config[aws.SessionToken] = string(profile.Credential.Secret.Data[secrets.AWSSessionToken])
	}
	if region == "" {
		config[aws.ConfigRegion] = profile.Location.Region
	} else {
		config[aws.ConfigRegion] = region
	}
	return aws.GetConfig(ctx, config)
}

func getAWSConfigFromSecret(ctx context.Context, secret corev1.Secret, region string) (*awssdk.Config, string, error) {
	err := secrets.ValidateAWSCredentials(&secret)
	if err != nil {
		return nil, "", errkit.Wrap(err, "Invalid AWS credential")
	}

	config := map[string]string{
		aws.AccessKeyID:     string(secret.Data[secrets.AWSAccessKeyID]),
		aws.SecretAccessKey: string(secret.Data[secrets.AWSSecretAccessKey]),
		aws.ConfigRole:      string(secret.Data[secrets.ConfigRole]),
		aws.SessionToken:    string(secret.Data[secrets.AWSSessionToken]),
	}

	if region != "" {
		config[aws.ConfigRegion] = region
	}
	return aws.GetConfig(ctx, config)
}

// findSecurityGroups return list of security group IDs associated with the RDS instance
func findSecurityGroups(ctx context.Context, rdsCli *rds.RDS, instanceID string) ([]string, error) {
	desc, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	var sgIDs []string
	for _, vpc := range desc.DBInstances[0].VpcSecurityGroups {
		sgIDs = append(sgIDs, *vpc.VpcSecurityGroupId)
	}
	return sgIDs, err
}

func findAuroraSecurityGroups(ctx context.Context, rdsCli *rds.RDS, instanceID string) ([]string, error) {
	desc, err := rdsCli.DescribeDBClusters(ctx, instanceID)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return nil, err
			}
			return nil, nil
		}
	}

	var sgIDs []string
	for _, vpc := range desc.DBClusters[0].VpcSecurityGroups {
		sgIDs = append(sgIDs, *vpc.VpcSecurityGroupId)
	}
	return sgIDs, nil
}

// findRDSEndpoint returns endpoint to access RDS instance
func findRDSEndpoint(ctx context.Context, rdsCli *rds.RDS, instanceID string) (string, error) {
	// Find host of the instance
	dbInstance, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return "", err
	}

	if (len(dbInstance.DBInstances) == 0) || (dbInstance.DBInstances[0].Endpoint == nil) {
		return "", errkit.New("Received nil endpoint")
	}
	return *dbInstance.DBInstances[0].Endpoint.Address, nil
}

// rdsDBEngineVersion returns the database engine version
func rdsDBEngineVersion(ctx context.Context, rdsCli *rds.RDS, instanceID string) (string, error) {
	dbInstance, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return "", err
	}

	if (len(dbInstance.DBInstances) == 0) || (dbInstance.DBInstances[0].EngineVersion == nil) {
		return "", errkit.New("DB Instance's Engine version is nil")
	}

	return *dbInstance.DBInstances[0].EngineVersion, nil
}

func GetRDSDBSubnetGroup(ctx context.Context, rdsCli *rds.RDS, instanceID string) (*string, error) {
	result, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if len(result.DBInstances) == 0 {
		return nil, errkit.New(fmt.Sprintf("Could not get DBInstance with the instanceID %s", instanceID))
	}
	return result.DBInstances[0].DBSubnetGroup.DBSubnetGroupName, nil
}

func GetRDSAuroraDBSubnetGroup(ctx context.Context, rdsCli *rds.RDS, instanceID string) (*string, error) {
	desc, err := rdsCli.DescribeDBClusters(ctx, instanceID)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return nil, err
			}
			return nil, nil
		}
	}
	if len(desc.DBClusters) == 0 {
		return nil, errkit.New(fmt.Sprintf("Could not get DBCluster with the instanceID %s", instanceID))
	}
	return desc.DBClusters[0].DBSubnetGroup, nil
}
