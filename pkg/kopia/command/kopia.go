// Copyright 2022 The Kanister Authors.
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

package command

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/secrets"
)

type CommandArgs struct {
	EncryptionKey  string
	ConfigFilePath string
	LogDirectory   string
}

func bashCommand(args logsafe.Cmd) []string {
	log.Debug().Print("Kopia Command", field.M{"Command": args.String()})
	return []string{"bash", "-o", "errexit", "-c", args.PlainText()}
}

func stringSliceCommand(args logsafe.Cmd) []string {
	log.Debug().Print("Kopia Command", field.M{"Command": args.String()})
	return args.StringSliceCMD()
}

func commonArgs(password, configFilePath, logDirectory string, requireInfoLevel bool) logsafe.Cmd {
	c := logsafe.NewLoggable(kopiaCommand)
	if requireInfoLevel {
		c = c.AppendLoggable(logLevelInfoFlag)
	} else {
		c = c.AppendLoggable(logLevelErrorFlag)
	}
	if configFilePath != "" {
		c = c.AppendLoggableKV(configFileFlag, configFilePath)
	}
	if logDirectory != "" {
		c = c.AppendLoggableKV(logDirectoryFlag, logDirectory)
	}
	if password != "" {
		c = c.AppendRedactedKV(passwordFlag, password)
	}
	return c
}

// ExecKopiaArgs returns the basic Argv for executing kopia with the given config file path.
func ExecKopiaArgs(configFilePath string) []string {
	return commonArgs("", configFilePath, "", false).StringSliceCMD()
}

func ResolveS3Endpoint(endpoint string) string {
	if strings.HasSuffix(endpoint, "/") {
		log.Debug().Print(fmt.Sprintf("Removing trailing slashes from the endpoint %s", endpoint))
		endpoint = strings.TrimRight(endpoint, "/")
	}
	sp := strings.SplitN(endpoint, "://", 2)
	if len(sp) > 1 {
		log.Debug().Print(fmt.Sprintf("Removing leading protocol from the endpoint %s", endpoint))
	}
	return sp[len(sp)-1]
}

func HttpInsecureEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http:")
}

// GenerateFullRepoPath defines the manner in which a location-specific prefix
// string is joined with a repository-specific prefix to generate the full path
// for a kopia repository.
func GenerateFullRepoPath(locPrefix string, repoPathPrefix string) string {
	if locPrefix != "" {
		return path.Join(locPrefix, repoPathPrefix) + "/"
	}

	return repoPathPrefix
}

func kopiaS3Args(prof kopia.Profile, repoPathPrefix string) (logsafe.Cmd, error) {
	args := logsafe.NewLoggable(s3SubCommand)
	args = args.AppendLoggableKV(bucketFlag, prof.BucketName())

	e := prof.Endpoint()
	if e != "" {
		s3Endpoint := ResolveS3Endpoint(e)
		args = args.AppendLoggableKV(endpointFlag, s3Endpoint)

		if HttpInsecureEndpoint(e) {
			args = args.AppendLoggable(disableTLSFlag)
		}
	}

	repoPathPrefix = GenerateFullRepoPath(prof.Prefix(), repoPathPrefix)

	credArgs, err := kopiaS3CredentialArgs(prof)
	if err != nil {
		return nil, err
	}

	args = args.Combine(credArgs)
	args = args.AppendLoggableKV(prefixFlag, repoPathPrefix)

	if prof.SkipSSLVerification() {
		args = args.AppendLoggable(disableTLSVerifyFlag)
	}

	region := prof.Region()
	if region != "" {
		args = args.AppendLoggableKV(regionFlag, region)
	}

	return args, nil
}

func kopiaS3CredentialArgs(prof kopia.Profile) (logsafe.Cmd, error) {
	credsType, err := prof.CredType()
	if err != nil {
		return nil, err
	}
	switch credsType {
	case kopia.SecretTypeK8sSecret:
		d, err := time.ParseDuration(AWSAssumeRoleDuration())
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse AWS Assume Role duration")
		}
		s3Creds, err := secrets.ExtractAWSCredentials(context.TODO(), prof.Secret(), d)
		if err != nil {
			return nil, err
		}

		args := logsafe.Cmd{}
		args = args.AppendRedactedKV(accessKeyFlag, s3Creds.AccessKeyID)
		args = args.AppendRedactedKV(secretAccessKeyFlag, s3Creds.SecretAccessKey)

		if s3Creds.SessionToken != "" {
			args = args.AppendRedactedKV(sessionTokenFlag, s3Creds.SessionToken)
		}

		return args, nil
	case kopia.SecretTypeKeyPair:
		args := logsafe.Cmd{}
		args = args.AppendRedactedKV(accessKeyFlag, prof.AccessKeyID())
		args = args.AppendRedactedKV(secretAccessKeyFlag, prof.SecretAccessKey())
		return args, nil
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type for credentials %s", credsType))
	}
}

func kopiaAzureArgs(prof kopia.Profile, repoPathPrefix string) (logsafe.Cmd, error) {
	repoPathPrefix = GenerateFullRepoPath(prof.Prefix(), repoPathPrefix)

	args := logsafe.NewLoggable(azureSubCommand)
	args = args.AppendLoggableKV(containerFlag, prof.BucketName())
	args = args.AppendLoggableKV(prefixFlag, repoPathPrefix)

	credArgs, err := kopiaAzureCredentialArgs(prof)
	if err != nil {
		return nil, err
	}

	return args.Combine(credArgs), nil
}

func kopiaAzureCredentialArgs(prof kopia.Profile) (logsafe.Cmd, error) {
	credsType, err := prof.CredType()
	if err != nil {
		return nil, err
	}
	var storageAccount, storageKey, storageEnv string
	switch credsType {
	case kopia.SecretTypeK8sSecret:
		azureSecret, err := secrets.ExtractAzureCredentials(prof.Secret())
		if err != nil {
			return nil, err
		}
		storageAccount = azureSecret.StorageAccount
		storageKey = azureSecret.StorageKey
		storageEnv = azureSecret.EnvironmentName
	case kopia.SecretTypeKeyPair:
		storageAccount = prof.StorageAccount()
		storageKey = prof.StorageKey()
		storageEnv = prof.StorageDomain()
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type for credentials %s", credsType))
	}

	args := logsafe.Cmd{}
	args = args.AppendRedactedKV(storageAccountFlag, storageAccount)
	args = args.AppendRedactedKV(storageKeyFlag, storageKey)
	if storageEnv != "" {
		env, err := azure.EnvironmentFromName(storageEnv)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to get azure environment from name %s", storageEnv))
		}
		blobDomain := "blob." + env.StorageEndpointSuffix
		args = args.AppendLoggableKV(storageDomainFlag, blobDomain)
	}
	return args, nil
}

func kopiaGCSArgs(prof kopia.Profile, repoPathPrefix string) logsafe.Cmd {
	repoPathPrefix = GenerateFullRepoPath(prof.Prefix(), repoPathPrefix)

	args := logsafe.NewLoggable(googleSubCommand)
	args = args.AppendLoggableKV(bucketFlag, prof.BucketName())
	args = args.AppendLoggableKV(credentialsFileFlag, consts.GoogleCloudCredsFilePath)
	return args.AppendLoggableKV(prefixFlag, repoPathPrefix)
}

func filesystemArgs(prof kopia.Profile, repoPathPrefix string) logsafe.Cmd {
	repoPathPrefix = GenerateFullRepoPath(prof.Prefix(), repoPathPrefix)

	args := logsafe.NewLoggable(filesystemSubCommand)
	return args.AppendLoggableKV(pathFlag, kopia.DefaultFSMountPath+"/"+repoPathPrefix)
}

func kopiaBlobStoreArgs(prof kopia.Profile, repoPathPrefix string) (logsafe.Cmd, error) {
	locType, err := prof.LocationType()
	if err != nil {
		return nil, err
	}
	switch locType {
	//case LocationTypeFileStore:
	//	return filesystemArgs(prof, repoPathPrefix), nil
	case kopia.LocationTypeS3:
		return kopiaS3Args(prof, repoPathPrefix)
	case kopia.LocationTypeGCS:
		return kopiaGCSArgs(prof, repoPathPrefix), nil
	case kopia.LocationTypeAzure:
		return kopiaAzureArgs(prof, repoPathPrefix)
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type for the location %s", locType))
	}
}
