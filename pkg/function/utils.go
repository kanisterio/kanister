package function

import (
	"bytes"
	"context"
	"os"
	"path"
	"strings"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	// FunctionOutputVersion returns version
	FunctionOutputVersion     = "version"
	kanisterToolsImage        = "ghcr.io/kanisterio/kanister-tools:0.50.0"
	kanisterToolsImageEnvName = "KANISTER_TOOLS"
)

func getKanisterToolsImage() string {
	if val, ok := os.LookupEnv(kanisterToolsImageEnvName); ok {
		return val
	}
	return kanisterToolsImage
}

// ValidateCredentials verifies if the given credentials have appropriate values set
func ValidateCredentials(creds *param.Credential) error {
	if creds == nil {
		return errors.New("Empty credentials")
	}
	switch creds.Type {
	case param.CredentialTypeKeyPair:
		if creds.KeyPair == nil {
			return errors.New("Empty KeyPair field")
		}
		if len(creds.KeyPair.ID) == 0 {
			return errors.New("Access key ID is not set")
		}
		if len(creds.KeyPair.Secret) == 0 {
			return errors.New("Secret access key is not set")
		}
		return nil
	case param.CredentialTypeSecret:
		return secrets.ValidateCredentials(creds.Secret)
	default:
		return errors.Errorf("Unsupported type '%s' for credentials", creds.Type)
	}
}

// ValidateProfile verifies if the given profile has valid creds and location type
func ValidateProfile(profile *param.Profile) error {
	if profile == nil {
		return errors.New("Profile must be non-nil")
	}
	if err := ValidateCredentials(&profile.Credential); err != nil {
		return err
	}
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
	case crv1alpha1.LocationTypeGCS:
	case crv1alpha1.LocationTypeAzure:
	default:
		return errors.New("Location type not supported")
	}
	return nil
}

// GetPodWriter creates a file with Google credentials if the given profile points to a GCS location
func GetPodWriter(cli kubernetes.Interface, ctx context.Context, namespace, podName, containerName string, profile *param.Profile) (*kube.PodWriter, error) {
	if profile.Location.Type == crv1alpha1.LocationTypeGCS {
		pw := kube.NewPodWriter(cli, consts.GoogleCloudCredsFilePath, bytes.NewBufferString(profile.Credential.KeyPair.Secret))
		if err := pw.Write(ctx, namespace, podName, containerName); err != nil {
			return nil, err
		}
		return pw, nil
	}
	return nil, nil
}

// CleanUpCredsFile is used to remove the file created by the given PodWriter
func CleanUpCredsFile(ctx context.Context, pw *kube.PodWriter, namespace, podName, containerName string) {
	if pw != nil {
		if err := pw.Remove(ctx, namespace, podName, containerName); err != nil {
			log.Error().WithContext(ctx).Print("Could not delete the temp file")
		}
	}
}

// FetchPodVolumes returns a map of PVCName->MountPath for a given pod
func FetchPodVolumes(pod string, tp param.TemplateParams) (map[string]string, error) {
	switch {
	case tp.Deployment != nil:
		if pvcToMountPath, ok := tp.Deployment.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	case tp.StatefulSet != nil:
		if pvcToMountPath, ok := tp.StatefulSet.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	default:
		return nil, errors.New("Invalid Template Params")
	}
}

// ResolveArtifactPrefix appends the bucket name as a suffix to the given artifact path if not already present
func ResolveArtifactPrefix(artifactPrefix string, profile *param.Profile) string {
	ps := strings.Split(artifactPrefix, "/")
	if ps[0] == profile.Location.Bucket {
		return artifactPrefix
	}
	return path.Join(profile.Location.Bucket, artifactPrefix)
}

func getAWSConfigFromProfile(ctx context.Context, profile *param.Profile) (*awssdk.Config, string, error) {
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
	config[aws.ConfigRegion] = profile.Location.Region
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
		return "", errors.Errorf("Received nil endpoint")
	}
	return *dbInstance.DBInstances[0].Endpoint.Address, nil
}

func createPostgresSecret(cli kubernetes.Interface, name, namespace, username, password string) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		},
	}
	_, err := cli.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	return err
}

func deletePostgresSecret(cli kubernetes.Interface, name, namespace string) error {
	return cli.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func isAuroraCluster(engine string) bool {
	for _, v := range []string{string(DBEngineAurora), string(DBEngineAuroraMySQL), string(DBEngineAuroraPostgreSQL)} {
		if engine == v {
			return true
		}
	}
	return false
}
