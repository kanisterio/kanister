package location

import (
	"context"
	"io"
	"path/filepath"

	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	AWSAccessKeyID     = "AWS_ACCESS_KEY_ID"
	AWSSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	GoogleCloudCreds   = "GOOGLE_APPLICATION_CREDENTIALS"
)

// Write pipes data from `in` into the location specified by `profile` and `suffix`.
func Write(ctx context.Context, in io.Reader, profile param.Profile, suffix string) error {
	osType, err := getProviderType(profile.Location.Type)
	if err != nil {
		return err
	}
	path := filepath.Join(
		profile.Location.Prefix,
		suffix,
	)
	return writeData(ctx, osType, profile, in, path)
}

// Read pipes data from `in` into the location specified by `profile` and `suffix`.
func Read(ctx context.Context, out io.Writer, profile param.Profile, suffix string) error {
	osType, err := getProviderType(profile.Location.Type)
	if err != nil {
		return err
	}
	path := filepath.Join(
		profile.Location.Prefix,
		suffix,
	)
	return readData(ctx, osType, profile, out, path)
}

//Delete data from location specified by `profile` and `suffix`.
func Delete(ctx context.Context, profile param.Profile, suffix string) error {
	osType, err := getProviderType(profile.Location.Type)
	if err != nil {
		return err
	}
	path := filepath.Join(
		profile.Location.Prefix,
		suffix,
	)
	return deleteData(ctx, osType, profile, path)
}

func readData(ctx context.Context, pType objectstore.ProviderType, profile param.Profile, out io.Writer, path string) error {
	bucket, err := getBucket(ctx, pType, profile)
	if err != nil {
		return err
	}

	r, _, err := bucket.Get(ctx, path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		return err
	}
	return nil
}

func writeData(ctx context.Context, pType objectstore.ProviderType, profile param.Profile, in io.Reader, path string) error {
	bucket, err := getBucket(ctx, pType, profile)
	if err != nil {
		return err
	}
	if err := bucket.Put(ctx, path, in, 0, nil); err != nil {
		return errors.Errorf("failed to write contents to bucket '%s'", profile.Location.Bucket)
	}
	return nil
}

func deleteData(ctx context.Context, pType objectstore.ProviderType, profile param.Profile, path string) error {
	bucket, err := getBucket(ctx, pType, profile)
	if err != nil {
		return err
	}
	return bucket.DeleteAllWithPrefix(ctx, path)
}

func getProviderType(lType crv1alpha1.LocationType) (objectstore.ProviderType, error) {
	switch lType {
	case crv1alpha1.LocationTypeS3Compliant:
		return objectstore.ProviderTypeS3, nil
	case crv1alpha1.LocationTypeGCS:
		return objectstore.ProviderTypeGCS, nil
	default:
		return "", errors.Errorf("Unsupported Location type: %s", lType)
	}
}

func getBucket(ctx context.Context, pType objectstore.ProviderType, profile param.Profile) (objectstore.Bucket, error) {
	pc := objectstore.ProviderConfig{
		Type: pType,
	}
	secret, err := getOSSecret(pType, profile.Credential)
	if err != nil {
		return nil, err
	}
	provider, err := objectstore.NewProvider(ctx, pc, secret)
	if err != nil {
		return nil, err
	}
	return provider.GetBucket(ctx, profile.Location.Bucket)
}

func getOSSecret(pType objectstore.ProviderType, cred param.Credential) (*objectstore.Secret, error) {
	secret := &objectstore.Secret{}
	switch pType {
	case objectstore.ProviderTypeS3:
		secret.Type = objectstore.SecretTypeAwsAccessKey
		secret.Aws = &objectstore.SecretAws{
			AccessKeyID:     cred.KeyPair.ID,
			SecretAccessKey: cred.KeyPair.Secret,
		}
	case objectstore.ProviderTypeGCS:
		secret.Type = objectstore.SecretTypeGcpServiceAccountKey
		secret.Gcp = &objectstore.SecretGcp{
			ProjectID:  cred.KeyPair.ID,
			ServiceKey: cred.KeyPair.Secret,
		}
	default:
		return nil, errors.Errorf("unknown or unsupported provider type '%s'", pType)
	}
	return secret, nil
}
