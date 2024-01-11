# Problem Statement
Currently we are using Kopia CLI to perform the repository and kopia repository server operations in kanister.
The repository controller creates a pod, executes commands through `kube.exec`` to perform operations. The commands include: 
- repo connect 
- start server 
- add users 
- refresh server 

The CLI commands executed using kube.exec can be flakey for long running commands.

The goal of the proposal is
- To replace kopia CLI with Kopia SDK wherever possible to gain more control over operations on kopia.

## Scope
1. Implement a library that provides wraps the SDK functions provided by kopia to connect to underlying storage providers - S3, Azure, GCP, Filestore
2. Build another pkg on top of the library implemented in #1 that can be used to perform repository operations


## High Level Design

### Kopia Library

#### Storage pkg

```go

package storage

import (
	"context"

	"github.com/kopia/kopia/repo/blob"
)

type StorageType string

const (
	TypeS3        StorageType = "S3"
	TypeAzure     StorageType = "Azure"
	TypeFileStore StorageType = "FileStore"
	TypeGCP       StorageType = "GCP"
)

type Storage interface {
	Connect() (blob.Storage, error)
	SetOptions(context.Context, map[string]string)
	WithCreate(bool)
}

func New(storageType StorageType) Storage {
	switch storageType {
	case TypeS3:
		return &s3Storage{}
	case TypeFileStore:
		return &fileSystem{}
	case TypeAzure:
		return &azureStorage{}
	case TypeGCP:
		return &gcpStorage{}
	default:
		return nil
	}
}

```


##### Example implementation of storage interface

```go

package storage

import (
	"context"

	"github.com/kanisterio/kanister/pkg/kopialib"
	"github.com/kanisterio/kanister/pkg/utils"
	"github.com/kopia/kopia/repo/blob"
	"github.com/kopia/kopia/repo/blob/s3"
)

type s3Storage struct {
	options *s3.Options
	create  bool
}

// This uses `github.com/kopia/kopia/repo/blob/s3` of kopia
// to connect to underlying storage
func (s *s3Storage) Connect() (blob.Storage, error) {
	return s3.New(context.Background(), s.Options, s.create)
}

// WithOptions function can be used if someone wants
// to set the s3 configuration directly using
// `github.com/kopia/kopia/repo/blob/s3` pkg
func (s *s3Storage) WithOptions(opts s3.Options) {
	s.options = &opts
}

// WithCreate function can be used
// if someone wants to create the underlying storage
func (s *s3Storage) WithCreate(create bool) {
	s.create = create
}

// SetOptions function is a generic way to set the configuration
// of underlying storage using a map
func (s *s3Storage) SetOptions(ctx context.Context, options map[string]string) {
	s.options = &s3.Options{
		BucketName:      options[kopialib.BucketKey],
		Endpoint:        options[kopialib.S3EndpointKey],
		Prefix:          options[kopialib.PrefixKey],
		Region:          options[kopialib.S3RegionKey],
		SessionToken:    options[kopialib.S3TokenKey],
		AccessKeyID:     options[kopialib.S3AccessKey],
		SecretAccessKey: options[kopialib.S3SecretAccessKey],
	}
	s.options.DoNotUseTLS, _ = utils.GetBoolOrDefault(options[kopialib.DoNotUseTLS], true)
	s.options.DoNotVerifyTLS, _ = utils.GetBoolOrDefault(options[kopialib.DoNotVerifyTLS], true)
}

```


#### Repository pkg

This pkg would be a wrapper over the `storage pkg` built above and the kopia repositoy pkg `github.com/kopia/kopia/repo` 
provided by kopia SDK

```go

package repository

type Repository struct {
	st          storage.Storage
	password    string
	configFile  string
	storageType storage.StorageType
}

// Create repository using kopia SDK
func (r *Repository) Create(opts *repo.NewRepositoryOptions) (err error) {
	storage, err := r.st.Connect()
	if err != nil {
		return err
	}
	return repo.Initialize(context.Background(), storage, opts, r.password)
}

// Connect to the repository using kopia SDK
func (r *Repository) Connect(opts *repo.ConnectOptions) (err error) {
	storage, err := r.st.Connect()
	if err != nil {
		return err
	}
	return repo.Connect(context.Background(), r.configFile, storage, r.password, opts)
}

// Connect to the repository by providing a config file
func (r *Repository) ConnectUsingFile() error {
	repoConfig := repositoryConfigFileName(r.configFile)
	if _, err := os.Stat(repoConfig); os.IsNotExist(err) {
		return errors.New("failed find kopia configuration file")
	}

	_, err := repo.Open(context.Background(), repoConfig, r.password, &repo.Options{})
	return err
}


func repositoryConfigFileName(configFile string) string {
	if configFile != "" {
		return configFile
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "kopia", "repository.config")
}

```