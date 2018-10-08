package objectstore

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"

	"github.com/kanisterio/kanister/pkg/param"
)

// GetOrCreateBucket is a helper function to access the package level getOrCreateBucket
func GetOrCreateBucket(ctx context.Context, p Provider, bucketName string, region string) (Directory, error) {
	return p.getOrCreateBucket(ctx, bucketName, region)
}

// IsS3Provider is a helper function to find out if a provider is an s3Provider
func IsS3Provider(p Provider) bool {
	if _, ok := p.(*s3Provider); ok {
		return true
	}
	return false
}
func GetBucket(ctx context.Context, profile *param.Profile, osType ProviderType, bucketName string) (Bucket, error) {
	pc := ProviderConfig{Type: osType}
	if osType == ProviderTypeS3 {
		if profile.Credential.Type != param.CredentialTypeKeyPair {
			return nil, errors.New("Unsupported Credential type")
		}
	}
	secret, err := GetSecret(osType, profile)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get provider credentials")
	}
	provider, err := NewProvider(ctx, pc, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get new provider")
	}
	return provider.GetBucket(ctx, bucketName)
}

func PutData(ctx context.Context, profile *param.Profile, osType ProviderType, bucketName, dirName, fileName string, data []byte) error {
	bucket, err := GetBucket(ctx, profile, osType, bucketName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get bucket")
	}
	directory, err := bucket.GetDirectory(ctx, dirName)
	if err != nil {
		directory, err = bucket.CreateDirectory(ctx, dirName)
		if err != nil {
			return errors.Wrapf(err, "Failed to create directory")
		}
	}
	return directory.PutBytes(ctx, fileName, data, nil)
}

func GetData(ctx context.Context, profile *param.Profile, osType ProviderType, bucketName, dirName, fileName string) ([]byte, error) {
	bucket, err := GetBucket(ctx, profile, osType, bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get bucket")
	}
	directory, err := bucket.GetDirectory(ctx, dirName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get directory")
	}
	data, _, err := directory.GetBytes(ctx, fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get data")
	}
	return data, nil
}

func DeleteData(ctx context.Context, profile *param.Profile, osType ProviderType, bucketName, dirName string) error {
	bucket, err := GetBucket(ctx, profile, osType, bucketName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get bucket")
	}
	directory, err := bucket.GetDirectory(ctx, dirName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get directory")
	}
	return errors.Wrapf(directory.DeleteDirectory(ctx), "Failed to delete data")
}

func GetSecret(osType ProviderType, profile *param.Profile) (*Secret, error) {
	secret := &Secret{}
	switch osType {
	case ProviderTypeS3:
		secret.Type = SecretTypeAwsAccessKey
		secret.Aws = &SecretAws{
			AccessKeyID:     profile.Credential.KeyPair.ID,
			SecretAccessKey: profile.Credential.KeyPair.Secret,
		}
	case ProviderTypeGCS:
		creds, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope)
		if err != nil {
			return nil, errors.New("Could not get GCS credentials")
		}
		secret.Type = SecretTypeGcpServiceAccountKey
		secret.Gcp = &SecretGcp{
			ServiceKey: string(creds.JSON),
			ProjectID:  creds.ProjectID,
		}
	case ProviderTypeAzure:
		secret.Type = SecretTypeAzStorageAccount
		secret.Azure = &SecretAzure{
			StorageAccount: os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"),
			StorageKey:     os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
		}
	default:
		return nil, errors.New("Unsupported provider " + string(osType))
	}
	return secret, nil
}
