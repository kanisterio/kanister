package storage

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/secrets"
)

type locType string

const (
	locTypeS3        locType = "s3"
	locTypeGCS       locType = "gcs"
	locTypeAzure     locType = "azure"
	locTypeFilestore locType = "filestore"
)

const (
	bucketKey        = "bucket"
	endpointKey      = "endpoint"
	prefixKey        = "prefix"
	regionKey        = "region"
	skipSSLVerifyKey = "skipSSLVerify"
	typeKey          = "type"
)

func bucketName(m map[string]string) string {
	return m[bucketKey]
}

func endpoint(m map[string]string) string {
	return m[endpointKey]
}

func prefix(m map[string]string) string {
	return m[prefixKey]
}

func region(m map[string]string) string {
	return m[regionKey]
}

func skipSSLVerify(m map[string]string) bool {
	v := m[skipSSLVerifyKey]
	return v == "true"
}

func locationType(m map[string]string) locType {
	return locType(m[typeKey])
}

func SkipCredentialSecretMount(m map[string]string) bool {
	return locType(m[typeKey]) == locTypeFilestore
}

func GenerateEnvSpecFromCredentialSecret(s *v1.Secret) ([]v1.EnvVar, error) {
	if s == nil {
		return nil, errors.New("Secret cannot be nil")
	}
	secType := string(s.Type)
	switch secType {
	case secrets.AWSSecretType:
		return getEnvSpecForAWSCredentialSecret(s)
	case secrets.AzureSecretType:
		return getEnvSpecForAzureCredentialSecret(s)
	}
	// We only need to set the environment variables in cases where
	// secret type is AWS or Azure.
	return nil, nil
}

func getEnvSpecForAWSCredentialSecret(s *v1.Secret) ([]v1.EnvVar, error) {
	var duration time.Duration
	var err error
	if val, ok := s.Data["assume_duration"]; ok {
		duration, err = time.ParseDuration(string(val))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse AWS Assume Role duration")
		}

	}
	creds, err := secrets.ExtractAWSCredentials(context.Background(), s, duration)
	if err != nil {
		return nil, err
	}
	envVars := []v1.EnvVar{}
	envVars = append(
		envVars,
		getEnvVarWithSecretRef(aws.AccessKeyID, s.Name, secrets.AWSAccessKeyID),
		getEnvVarWithSecretRef(aws.SecretAccessKey, s.Name, secrets.AWSSecretAccessKey),
	)
	if creds.SessionToken != "" {
		envVars = append(envVars, getEnvVarWithSecretRef(aws.SessionToken, s.Name, secrets.AWSSessionToken))
	}
	return envVars, nil
}

func getEnvSpecForAzureCredentialSecret(s *v1.Secret) ([]v1.EnvVar, error) {
	azureSecret, err := secrets.ExtractAzureCredentials(s)
	if err != nil {
		return nil, err
	}
	envVars := []v1.EnvVar{}
	envVars = append(
		envVars,
		getEnvVarWithSecretRef("AZURE_STORAGE_ACCOUNT", s.Name, secrets.AzureStorageAccountID),
		getEnvVarWithSecretRef("AZURE_STORAGE_KEY", s.Name, secrets.AzureStorageAccountKey),
	)
	storageEnv := azureSecret.EnvironmentName
	if storageEnv != "" {
		env, err := azure.EnvironmentFromName(storageEnv)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get azure environment from name: %s", storageEnv)
		}
		blobDomain := "blob." + env.StorageEndpointSuffix
		// TODO : Check how we can set this env to use value from secret
		envVars = append(envVars, getEnvVar("AZURE_STORAGE_DOMAIN", blobDomain))
	}
	return envVars, nil
}

func getEnvVarWithSecretRef(varName, secretName, secretKey string) v1.EnvVar {
	return v1.EnvVar{
		Name: varName,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				Key: secretKey,
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
			},
		},
	}
}

func getEnvVar(varName, value string) v1.EnvVar {
	return v1.EnvVar{
		Name:  varName,
		Value: value,
	}
}
