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

package objectstore

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/graymeta/stow"
	stowaz "github.com/graymeta/stow/azure"
	stowgcs "github.com/graymeta/stow/google"
	stows3 "github.com/graymeta/stow/s3"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// Provider abstracts actions on cloud provider bucket
type Provider interface {
	// CreateBucket creates a new bucket. Bucket name must
	// honors provider naming restrictions
	CreateBucket(ctx context.Context, bucketName, region string) (Bucket, error)

	// GetBucket access a specific bucket
	GetBucket(context.Context, string) (Bucket, error)

	// DeleteBucket deletes the bucket
	DeleteBucket(context.Context, string) error

	// ListBuckets returns all buckets and their Directory handle
	ListBuckets(context.Context) (map[string]Bucket, error)

	// getOrCreateBucket creates bucket if it does not already exist
	getOrCreateBucket(ctx context.Context, bucketName, region string) (Bucket, error)
}

// Bucket abstracts the object store of different cloud providers
type Bucket interface {
	Directory
}

// Directory operations
type Directory interface {
	// CreateDirectory creates a sub directory
	CreateDirectory(context.Context, string) (Directory, error)

	// GetDirectory gets the sub directory
	GetDirectory(context.Context, string) (Directory, error)

	// DeleteDirectory deletes the current directory
	DeleteDirectory(context.Context) error

	// DeleteAllWithPrefix deletes all directorys and objects with a provided prefix
	DeleteAllWithPrefix(context.Context, string) error

	// ListDirectories lists all the directories rooted in
	// the current directory and their handle
	ListDirectories(context.Context) (map[string]Directory, error)

	// ListObjects lists all the objects rooted in the current directory
	ListObjects(context.Context) ([]string, error)

	// Get returns the io interface to read object data
	Get(context.Context, string) (io.ReadCloser, map[string]string, error)

	// Get returns bytes in the named object
	GetBytes(context.Context, string) ([]byte, map[string]string, error)

	// Put persists data from the Reader interface in the named object
	Put(context.Context, string, io.Reader, int64, map[string]string) error

	// Put persists bytes in the named object
	PutBytes(context.Context, string, []byte, map[string]string) error

	// Delete removes the object
	Delete(context.Context, string) error

	// Serialize directory
	String() string
}

// NewProvider creates a new Provider
func NewProvider(ctx context.Context, config ProviderConfig, secret *Secret) (Provider, error) {
	p := &provider{
		hostEndPoint: getHostURI(config),
		config:       config,
		secret:       secret,
	}
	if p.config.Type == ProviderTypeS3 {
		return &s3Provider{provider: p}, nil
	}
	return p, nil
}

// Supported returns true if the object store type is supported
func Supported(t ProviderType) bool {
	return t == ProviderTypeS3 || t == ProviderTypeGCS || t == ProviderTypeAzure
}

func s3Config(config ProviderConfig, secret *Secret, region string) (stowKind string, stowConfig stow.Config, err error) {
	var awsAccessKeyID, awsSecretAccessKey string
	if secret != nil {
		if secret.Type != SecretTypeAwsAccessKey {
			return "", nil, errors.Errorf("invalid secret type %s", secret.Type)
		}
		awsAccessKeyID = secret.Aws.AccessKeyID
		awsSecretAccessKey = secret.Aws.SecretAccessKey
	} else {
		var ok bool
		if awsAccessKeyID, ok = os.LookupEnv("AWS_ACCESS_KEY_ID"); !ok {
			return "", nil, errors.New("AWS_ACCESS_KEY environment not set")
		}
		if awsSecretAccessKey, ok = os.LookupEnv("AWS_SECRET_ACCESS_KEY"); !ok {
			return "", nil, errors.New("AWS_SECRET_ACCESS_KEY environment not set")
		}
	}
	cm := stow.ConfigMap{
		stows3.ConfigAccessKeyID: awsAccessKeyID,
		stows3.ConfigSecretKey:   awsSecretAccessKey,
	}
	if config.Role != "" {
		// Switch role and replace credentials.
		creds, err := switchRole(awsAccessKeyID, awsSecretAccessKey, config.Role)
		if err != nil {
			return "", cm, errors.New("Failed to switch role")
		}
		cm = stow.ConfigMap{
			stows3.ConfigAccessKeyID: creds.accessKeyID,
			stows3.ConfigSecretKey:   creds.secretAccessKey,
			stows3.ConfigToken:       creds.token,
		}
	}
	if region != "" {
		cm[stows3.ConfigRegion] = region
	}
	if config.Endpoint != "" {
		cm[stows3.ConfigEndpoint] = config.Endpoint
	}
	if config.SkipSSLVerify {
		cm[stows3.ConfigInsecureSkipSSLVerify] = "true"
	}
	return stows3.Kind, cm, nil
}

func gcsConfig(ctx context.Context, secret *Secret) (stowKind string, stowConfig stow.Config, err error) {
	var configJSON string
	var projectID string
	if secret != nil {
		if secret.Type != SecretTypeGcpServiceAccountKey {
			return "", nil, errors.Errorf("invalid secret type %s", secret.Type)
		}
		configJSON = secret.Gcp.ServiceKey
		projectID = secret.Gcp.ProjectID
	} else {
		creds, err := google.FindDefaultCredentials(ctx, compute.ComputeScope)
		if err != nil {
			return "", nil, err
		}
		configJSON = string(creds.JSON)
		projectID = creds.ProjectID
	}
	return stowgcs.Kind, stow.ConfigMap{
		stowgcs.ConfigJSON:      configJSON,
		stowgcs.ConfigProjectId: projectID,
		stowgcs.ConfigScopes:    "",
	}, nil
}

func azureConfig(ctx context.Context, secret *Secret) (stowKind string, stowConfig stow.Config, err error) {
	var azAccount, azStorageKey string
	if secret != nil {
		if secret.Type != SecretTypeAzStorageAccount {
			return "", nil, errors.Errorf("invalid secret type %s", secret.Type)
		}
		azAccount = secret.Azure.StorageAccount
		azStorageKey = secret.Azure.StorageKey
	} else {
		var ok bool
		azAccount, ok = os.LookupEnv("AZURE_STORAGE_ACCOUNT")
		if !ok {
			return "", nil, errors.New("AZURE_STORAGE_ACCOUNT environment not set")
		}

		azStorageKey, ok = os.LookupEnv("AZURE_STORAGE_KEY")
		if !ok {
			return "", nil, errors.New("AZURE_STORAGE_KEY environment not set")
		}
	}
	return stowaz.Kind, stow.ConfigMap{
		stowaz.ConfigAccount: azAccount,
		stowaz.ConfigKey:     azStorageKey,
	}, nil
}

func getConfig(ctx context.Context, config ProviderConfig, secret *Secret, region string) (stowKind string, stowConfig stow.Config, err error) {
	switch config.Type {
	case ProviderTypeS3:
		return s3Config(config, secret, region)
	case ProviderTypeGCS:
		return gcsConfig(ctx, secret)
	case ProviderTypeAzure:
		return azureConfig(ctx, secret)
	default:
		return "", nil, errors.Errorf("unknown or unimplemented object store type %s", config.Type)
	}
}

func getStowLocation(ctx context.Context, config ProviderConfig, secret *Secret, region string) (stow.Location, error) {
	kind, stowConfig, err := getConfig(ctx, config, secret, region)
	if err != nil {
		return nil, err
	}
	location, err := stow.Dial(kind, stowConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create store provider %+v", config)
	}
	return location, nil
}

func getHostURI(config ProviderConfig) string {
	switch config.Type {
	case ProviderTypeGCS:
		return googleGCSHost
	default:
		return config.Endpoint
	}
}

func awsS3Endpoint(region string) string {
	return fmt.Sprintf(awsS3HostFmt, region)
}
