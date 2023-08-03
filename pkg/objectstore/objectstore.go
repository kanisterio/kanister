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
	CreateBucket(context.Context, string) (Bucket, error)

	// GetBucket access a specific bucket
	GetBucket(context.Context, string) (Bucket, error)

	// DeleteBucket deletes the bucket
	DeleteBucket(context.Context, string) error

	// ListBuckets returns all buckets and their Directory handle
	ListBuckets(context.Context) (map[string]Bucket, error)

	// getOrCreateBucket creates bucket if it does not already exist
	getOrCreateBucket(ctx context.Context, bucketName string) (Bucket, error)
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

	// DeleteAllWithPrefix deletes all directories and objects with a provided prefix
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
	config.Endpoint = providerEndpoint(config)
	p := &provider{
		config: config,
		secret: secret,
	}
	if p.config.Type == ProviderTypeS3 {
		return &s3Provider{provider: p}, nil
	}
	return p, nil
}

func providerEndpoint(c ProviderConfig) string {
	if c.Type == ProviderTypeGCS {
		return googleGCSHost
	}
	return c.Endpoint
}

// Supported returns true if the object store type is supported
func Supported(t ProviderType) bool {
	return t == ProviderTypeS3 || t == ProviderTypeGCS || t == ProviderTypeAzure
}

func s3Config(ctx context.Context, config ProviderConfig, secret *Secret) (stowKind string, stowConfig stow.Config, err error) {
	if secret == nil {
		return "", nil, errors.New("Invalid Secret value: nil")
	}
	if secret.Type != SecretTypeAwsAccessKey {
		return "", nil, errors.Errorf("invalid secret type %s", secret.Type)
	}
	awsAccessKeyID := secret.Aws.AccessKeyID
	awsSecretAccessKey := secret.Aws.SecretAccessKey
	awsSessionToken := secret.Aws.SessionToken

	cm := stow.ConfigMap{
		stows3.ConfigAccessKeyID: awsAccessKeyID,
		stows3.ConfigSecretKey:   awsSecretAccessKey,
		stows3.ConfigToken:       awsSessionToken,
	}
	if config.Region != "" {
		cm[stows3.ConfigRegion] = config.Region
	}
	if config.Endpoint != "" {
		cm[stows3.ConfigEndpoint] = config.Endpoint
	}
	if config.SkipSSLVerify {
		cm[stows3.ConfigInsecureSkipSSLVerify] = "true"
	}
	return stows3.Kind, cm, nil
}

func gcsConfig(ctx context.Context, config ProviderConfig, secret *Secret) (stowKind string, stowConfig stow.Config, err error) {
	var configJSON string
	var projectID string
	cm := stow.ConfigMap{}
	if secret != nil {
		if secret.Type != SecretTypeGcpServiceAccountKey {
			return "", nil, errors.Errorf("invalid secret type %s", secret.Type)
		}
		configJSON = secret.Gcp.ServiceKey
		projectID = secret.Gcp.ProjectID
		if config.Region != "" {
			cm[stowgcs.ConfigLocation] = config.Region
			cm[stowgcs.ConfigStorageClass] = REGIONAL
		}
	} else {
		creds, err := google.FindDefaultCredentials(ctx, compute.ComputeScope)
		if err != nil {
			return "", nil, err
		}
		configJSON = string(creds.JSON)
		projectID = creds.ProjectID
	}
	cm[stowgcs.ConfigJSON] = configJSON
	cm[stowgcs.ConfigProjectId] = projectID
	cm[stowgcs.ConfigScopes] = ""
	return stowgcs.Kind, cm, nil
}

func azureConfig(ctx context.Context, secret *Secret) (stowKind string, stowConfig stow.Config, err error) {
	var azAccount, azStorageKey, azEnvName string
	if secret != nil {
		if secret.Type != SecretTypeAzStorageAccount {
			return "", nil, errors.Errorf("invalid secret type %s", secret.Type)
		}
		azAccount = secret.Azure.StorageAccount
		azStorageKey = secret.Azure.StorageKey
		azEnvName = secret.Azure.EnvironmentName
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
		azEnvName, _ = os.LookupEnv("AZURE_ENV_NAME") // not required to be set.
	}
	return stowaz.Kind, stow.ConfigMap{
		stowaz.ConfigAccount: azAccount,
		stowaz.ConfigKey:     azStorageKey,
		stowaz.ConfigEnvName: azEnvName,
	}, nil
}

func getConfig(ctx context.Context, config ProviderConfig, secret *Secret) (stowKind string, stowConfig stow.Config, err error) {
	switch config.Type {
	case ProviderTypeS3:
		return s3Config(ctx, config, secret)
	case ProviderTypeGCS:
		return gcsConfig(ctx, config, secret)
	case ProviderTypeAzure:
		return azureConfig(ctx, secret)
	default:
		return "", nil, errors.Errorf("unknown or unimplemented object store type %s", config.Type)
	}
}

func getStowLocation(ctx context.Context, config ProviderConfig, secret *Secret) (stow.Location, error) {
	kind, stowConfig, err := getConfig(ctx, config, secret)
	if err != nil {
		return nil, err
	}
	location, err := stow.Dial(kind, stowConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create store provider %+v", config)
	}
	return location, nil
}
