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

package location

import (
	"bytes"
	"context"
	"io"
	"path/filepath"

	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	AWSAccessKeyID      = "AWS_ACCESS_KEY_ID"
	AWSSecretAccessKey  = "AWS_SECRET_ACCESS_KEY"
	AWSSessionToken     = "AWS_SESSION_TOKEN"
	GoogleCloudCreds    = "GOOGLE_APPLICATION_CREDENTIALS"
	GoogleProjectID     = "GOOGLE_PROJECT_ID"
	AzureStorageAccount = "AZURE_ACCOUNT_NAME"
	AzureStorageKey     = "AZURE_ACCOUNT_KEY"

	// buffSize is the maximum size of an object that can be Put to Azure container in a single request
	// https://github.com/kastenhq/stow/blob/v0.2.6-kasten/azure/container.go#L14
	buffSize      = 256 * 1024 * 1024
	buffSizeLimit = 1 * 1024 * 1024 * 1024
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

// Delete data from location specified by `profile` and `suffix`.
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
	var input io.Reader
	var size int64
	bucket, err := getBucket(ctx, pType, profile)
	if err != nil {
		return err
	}

	var r io.Reader
	var n int64
	if pType == objectstore.ProviderTypeAzure {
		// Switch to multipart upload based on data size
		r, n, err = readerSize(in, buffSize)
		if err != nil {
			return err
		}

		input = r
		size = int64(n)
	} else {
		input = in
		size = 0
	}

	if err := bucket.Put(ctx, path, input, size, nil); err != nil {
		return errors.Wrapf(err, "failed to write contents to bucket '%s'", profile.Location.Bucket)
	}

	return nil
}

// readerSize checks if data size is greater than buffSize i.e the max size of an object that can be Put to Azure container in a single request
// If size >= buffSize, return buffer size to enable multipart upload, otherwise return 0 buffersize.
func readerSize(in io.Reader, buffSize int64) (io.Reader, int64, error) {
	var n int64
	var err error
	var r io.Reader

	// Read first buffSize bytes into buffer
	buf := make([]byte, buffSize)
	m, err := in.Read(buf)
	if err != nil && err != io.EOF {
		return nil, 0, err
	}

	// If first buffSize bytes are read successfully, that means the data size >= buffSize
	if int64(m) == buffSize {
		r = io.MultiReader(bytes.NewReader(buf), in)
		n = buffSizeLimit
	} else {
		buf = buf[:m]
		r = bytes.NewReader(buf)
	}

	return r, n, nil
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
	case crv1alpha1.LocationTypeAzure:
		return objectstore.ProviderTypeAzure, nil
	default:
		return "", errors.Errorf("Unsupported Location type: %s", lType)
	}
}

func getBucket(ctx context.Context, pType objectstore.ProviderType, profile param.Profile) (objectstore.Bucket, error) {
	pc := objectstore.ProviderConfig{
		Type:          pType,
		Endpoint:      profile.Location.Endpoint,
		Region:        profile.Location.Region,
		SkipSSLVerify: profile.SkipSSLVerify,
	}
	secret, err := getOSSecret(ctx, pType, profile.Credential)
	if err != nil {
		return nil, err
	}
	provider, err := objectstore.NewProvider(ctx, pc, secret)
	if err != nil {
		return nil, err
	}
	return provider.GetBucket(ctx, profile.Location.Bucket)
}

func getOSSecret(ctx context.Context, pType objectstore.ProviderType, cred param.Credential) (*objectstore.Secret, error) {
	secret := &objectstore.Secret{}
	switch pType {
	case objectstore.ProviderTypeS3:
		return getAWSSecret(ctx, cred)
	case objectstore.ProviderTypeGCS:
		secret.Type = objectstore.SecretTypeGcpServiceAccountKey
		secret.Gcp = &objectstore.SecretGcp{
			ProjectID:  cred.KeyPair.ID,
			ServiceKey: cred.KeyPair.Secret,
		}
	case objectstore.ProviderTypeAzure:
		return getAzureSecret(cred)
	default:
		return nil, errors.Errorf("unknown or unsupported provider type '%s'", pType)
	}
	return secret, nil
}

func getAzureSecret(cred param.Credential) (*objectstore.Secret, error) {
	os := &objectstore.Secret{
		Type: objectstore.SecretTypeAzStorageAccount,
	}
	switch cred.Type {
	case param.CredentialTypeKeyPair:
		os.Azure = &objectstore.SecretAzure{
			StorageAccount: cred.KeyPair.ID,
			StorageKey:     cred.KeyPair.Secret,
		}

	case param.CredentialTypeSecret:
		azSecret, err := secrets.ExtractAzureCredentials(cred.Secret)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to extract azure credentials")
		}
		os.Azure = azSecret
	}
	return os, nil
}

func getAWSSecret(ctx context.Context, cred param.Credential) (*objectstore.Secret, error) {
	os := &objectstore.Secret{
		Type: objectstore.SecretTypeAwsAccessKey,
	}
	switch cred.Type {
	case param.CredentialTypeKeyPair:
		os.Aws = &objectstore.SecretAws{
			AccessKeyID:     cred.KeyPair.ID,
			SecretAccessKey: cred.KeyPair.Secret,
		}
		return os, nil
	case param.CredentialTypeSecret:
		creds, err := secrets.ExtractAWSCredentials(ctx, cred.Secret, aws.AssumeRoleDurationDefault)
		if err != nil {
			return nil, err
		}
		os.Aws = &objectstore.SecretAws{
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
		}
		return os, nil
	default:
		return nil, errors.Errorf("Unsupported type '%s' for credential", cred.Type)
	}
}
