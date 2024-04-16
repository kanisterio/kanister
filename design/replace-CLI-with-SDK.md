
<!-- toc -->
- [Motivation](#motivation)
- [Scope](#scope)
- [High Level Design](#high-level-design)
  - [Kopia SDK wrappers](#kopia-sdk-wrappers)
    - [Storage pkg](#storage-pkg)
    - [Repository pkg](#repository-pkg)
  - [Repository server controller changes](#repository-server-controller-changes)
	- [Kopia CLI Approach](#kopia-cli-approach)
	- [Kopia SDK Approach](#kopia-sdk-approach)
<!-- /toc -->


# Motivation

Kanister uses a Kubernetes custom controller that makes use of Kopia as a primary backup and restore tool. 
The detailed design of the customer controller can be found [here](https://github.com/kanisterio/kanister/blob/master/design/kanister-kopia-integration.md)

The custom controller called as repository server controller currently uses Kopia CLI to perform the Kopia operations. All the
operations are executed inside a pod using the `kubectl exec` function. 

We can get following benefits if we start using Kopia SDK instead of CLI:
- Better error handling
- Dependency management

The goal over here is to start the Kopia server by executing a pre-built binary. The binary would execute most of the 
operations to start the server using kopia SDK and reduce the dependency on Kopia CLI and gain more flexibility over the operations

## Scope
1. Implement a library that wraps the SDK functions provided by Kopia to connect to underlying storage providers - S3, Azure, GCP, Filestore
2. Build another library on top of the library implemented in #1 that can be used to perform repository operations
3. Modify the repository server controller to run a pod that executes a custom image. The binary would take all the
necessary steps to make the Kopia server ready using Kopia SDK and Kopia CLI. 

## High Level Design

### Kopia SDK Wrappers

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

#### Repository pkg

This pkg would be a wrapper over the `storage pkg` built above and the Kopia repositoy pkg `github.com/kopia/kopia/repo` 
provided by Kopia SDK

```go

package repository

type Repository struct {
	st          storage.Storage
	password    string
	configFile  string
	storageType storage.StorageType
}

// Create repository using Kopia SDK
func (r *Repository) Create(opts *repo.NewRepositoryOptions) (err error) {
	storage, err := r.st.Connect()
	if err != nil {
		return err
	}
	return repo.Initialize(context.Background(), storage, opts, r.password)
}

// Connect to the repository using Kopia SDK
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

### Repository server controller changes

#### Kopia CLI Approach

![Alt text](images/kopia-CLI-workflow.png?raw=true "Kanister Kopia Integration using Kopia CLI")

Above diagram explains the current workflow repository server controller
uses to start the Kopia repository server. All the commands are executed from the controller pod inside 
the respoitory server pod using `kube.exec`. The repository server pod that is created by controller
uses `kanister-tools` image


#### Kopia SDK Approach

![Alt text](images/kopia-SDK-workflow.png?raw=true "Kanister Kopia Integration using Kopia SDK")


As shown in the figure we will be building a custom image which is going have this workflow:
1. Start the Kopia repository server in `--async-repo-connect` mode that means the server would be started without
connecting to the repository in an async mode. Existing approach starts the server only after the connection to Kopia repository
is successful. Kopia SDK currently does not have an exported function to start the Kopia server. So we would still be using Kopia
CLI to start the server
2. Check Kopia server status using Kopia SDK wrappers explained in section [Kopia SDK wrappers](#kopia-sdk-wrappers)
2. Connect to Kopia repository using Kopia SDK wrappers
3. Add or Update server users using Kopia SDK wrappers
4. Refresh server using Kopia SDK wrappers