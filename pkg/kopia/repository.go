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

package kopia

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	apiconfig "github.com/kanisterio/kanister/pkg/apis/config/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	kerrors "github.com/kanisterio/kanister/pkg/errors"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	errInvalidPassword        = "invalid repository password"
	errAccessDenied           = "Access Denied"
	errRepoNotInitialized     = "repository not initialized in the provided storage"
	errFilesystemRepoNotFound = "no such file or directory"
	errRepoNotFoundStr        = "repository not found"
	MaintenanceOwnerFormat    = "%s@%s-maintenance"
	ErrCodeOutOfMemory        = "command terminated with exit code 137"
	OutOfMemoryStr            = "kanister-tools container ran out of memory"

	// DefaultAWSAssumeRoleDuration is the default for Assume Role Duration (in minutes)
	DefaultAWSAssumeRoleDuration = "60m"
	// AWSAssumeRoleDurationVarName is the environment variable that controls AWS Assume Role Duration.
	AWSAssumeRoleDurationVarName = "AWS_ASSUME_ROLE_DURATION"
)

func ResolveS3Endpoint(endpoint string) string {
	if strings.HasSuffix(endpoint, "/") {
		log.Debug().Print("Removing trailing slashes from the endpoint")
		endpoint = strings.TrimRight(endpoint, "/")
	}
	sp := strings.SplitN(endpoint, "://", 2)
	if len(sp) > 1 {
		log.Debug().Print("Removing leading protocol from the endpoint")
	}
	return sp[len(sp)-1]
}

func HttpInsecureEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http:")
}

// GenerateFullRepoPath defines the manner in which a location-specific prefix
// string is joined with a repository-specific prefix to generate the full path
// for a kopia repository.
func GenerateFullRepoPath(locPrefix string, artifactPrefix string) string {
	if locPrefix != "" {
		return path.Join(locPrefix, artifactPrefix) + "/"
	}

	return artifactPrefix
}

func kopiaS3Args(prof profile, artifactPrefix string) (logsafe.Cmd, error) {
	args := logsafe.NewLoggable(s3SubCommand)
	args = args.AppendLoggableKV(bucketFlag, prof.bucketName())

	e := prof.endpoint()
	if e != "" {
		s3Endpoint := ResolveS3Endpoint(e)
		args = args.AppendLoggableKV(endpointFlag, s3Endpoint)

		if HttpInsecureEndpoint(e) {
			args = args.AppendLoggable(disableTLSFlag)
		}
	}

	artifactPrefix = GenerateFullRepoPath(prof.prefix(), artifactPrefix)

	credArgs, err := kopiaS3CredentialArgs(prof)
	if err != nil {
		return nil, err
	}

	args = args.Combine(credArgs)
	args = args.AppendLoggableKV(prefixFlag, artifactPrefix)

	if prof.skipSSLVerify() {
		args = args.AppendLoggable(disableTLSVerifyFlag)
	}

	region := prof.region()
	if region != "" {
		args = args.AppendLoggableKV(regionFlag, region)
	}

	return args, nil
}

func kopiaS3CredentialArgs(prof profile) (logsafe.Cmd, error) {
	credsType, err := prof.credType()
	if err != nil {
		return nil, err
	}
	switch credsType {
	case secretTypeK8sSecret:
		d, err := time.ParseDuration(utils.GetEnvAsStringOrDefault(AWSAssumeRoleDurationVarName, DefaultAWSAssumeRoleDuration))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse AWS Assume Role duration")
		}
		s3Creds, err := secrets.ExtractAWSCredentials(context.TODO(), prof.secret(), d)
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
	case secretTypeKeyPair, apiconfig.SecretTypeAwsAccessKey:
		args := logsafe.Cmd{}
		args = args.AppendRedactedKV(accessKeyFlag, prof.accessKeyID())
		args = args.AppendRedactedKV(secretAccessKeyFlag, prof.secretAccessKey())
		return args, nil
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type for credentials %s", credsType))
	}
}

func kopiaAzureArgs(prof profile, artifactPrefix string) (logsafe.Cmd, error) {
	artifactPrefix = GenerateFullRepoPath(prof.prefix(), artifactPrefix)

	args := logsafe.NewLoggable(azureSubCommand)
	args = args.AppendLoggableKV(containerFlag, prof.bucketName())
	args = args.AppendLoggableKV(prefixFlag, artifactPrefix)

	credArgs, err := kopiaAzureCredentialArgs(prof)
	if err != nil {
		return nil, err
	}

	return args.Combine(credArgs), nil
}

func kopiaAzureCredentialArgs(prof profile) (logsafe.Cmd, error) {
	credsType, err := prof.credType()
	if err != nil {
		return nil, err
	}
	var storageAccount, storageKey, storageEnv string
	switch credsType {
	case secretTypeK8sSecret:
		azureSecret, err := secrets.ExtractAzureCredentials(prof.secret())
		if err != nil {
			return nil, err
		}
		storageAccount = azureSecret.StorageAccount
		storageKey = azureSecret.StorageKey
		storageEnv = azureSecret.EnvironmentName
	case secretTypeKeyPair, apiconfig.SecretTypeAzStorageAccount:
		storageAccount = prof.storageAccount()
		storageKey = prof.storageKey()
		storageEnv = prof.storageDomain()
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

func kopiaGCSArgs(prof profile, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(prof.prefix(), artifactPrefix)

	args := logsafe.NewLoggable(googleSubCommand)
	args = args.AppendLoggableKV(bucketFlag, prof.bucketName())
	args = args.AppendLoggableKV(credentialsFileFlag, consts.GoogleCloudCredsFilePath)
	return args.AppendLoggableKV(prefixFlag, artifactPrefix)
}

func filesystemArgs(prof profile, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(prof.prefix(), artifactPrefix)

	args := logsafe.NewLoggable(filesystemSubCommand)
	return args.AppendLoggableKV(pathFlag, DefaultFSMountPath+"/"+artifactPrefix)
}

func kopiaBlobStoreArgs(prof profile, artifactPrefix string) (logsafe.Cmd, error) {
	locType, err := prof.locationType()
	if err != nil {
		return nil, err
	}
	switch locType {
	case apiconfig.LocationTypeFileStore:
		return filesystemArgs(prof, artifactPrefix), nil
	case locationTypeS3:
		return kopiaS3Args(prof, artifactPrefix)
	case locationTypeGCS:
		return kopiaGCSArgs(prof, artifactPrefix), nil
	case locationTypeAzure:
		return kopiaAzureArgs(prof, artifactPrefix)
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type for the location %s", locType))
	}
}

// RepositoryCreateCommand returns the kopia command for creation of a blob-store repo
// TODO: Consolidate all the repository options into a struct and pass
func RepositoryCreateCommand(
	prof profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
) ([]string, error) {
	cmd, err := repositoryCreateCommand(
		prof,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
	)
	if err != nil {
		return nil, err
	}

	return stringSliceCommand(cmd), nil
}

func repositoryCreateCommand(
	prof profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
) (logsafe.Cmd, error) {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, createSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}

	bsArgs, err := kopiaBlobStoreArgs(prof, artifactPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	return args.Combine(bsArgs), nil
}

// RepositoryConnectCommand returns the kopia command for connecting to an existing blob-store repo
func RepositoryConnectCommand(
	prof profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	pointInTimeConnection strfmt.DateTime,
) ([]string, error) {
	cmd, err := repositoryConnectCommand(
		prof,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		pointInTimeConnection,
	)
	if err != nil {
		return nil, err
	}

	return stringSliceCommand(cmd), nil
}

func repositoryConnectCommand(
	prof profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	pointInTimeConnection strfmt.DateTime,
) (logsafe.Cmd, error) {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}

	bsArgs, err := kopiaBlobStoreArgs(prof, artifactPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	if !time.Time(pointInTimeConnection).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, pointInTimeConnection.String())
	}

	return args.Combine(bsArgs), nil
}

// RepositoryConnectServerCommand returns the kopia command for connecting to a remote repository on Kopia API server
func RepositoryConnectServerCommand(
	cacheDirectory,
	configFilePath,
	hostname,
	logDirectory,
	serverURL,
	fingerprint,
	username,
	userPassword string,
	contentCacheMB,
	metadataCacheMB int,
) []string {
	return stringSliceCommand(repositoryConnectServerCommand(
		cacheDirectory,
		configFilePath,
		hostname,
		logDirectory,
		serverURL,
		fingerprint,
		username,
		userPassword,
		contentCacheMB,
		metadataCacheMB,
	))
}

func repositoryConnectServerCommand(
	cacheDirectory,
	configFilePath,
	hostname,
	logDirectory,
	serverURL,
	fingerprint,
	username,
	userPassword string,
	contentCacheMB,
	metadataCacheMB int,
) logsafe.Cmd {
	args := kopiaArgs(userPassword, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}
	args = args.AppendLoggableKV(urlFlag, serverURL)

	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)

	return args
}

// CreateKopiaRepository creates a kopia repository if not already present
// Returns true if successful or false with an error
// If the error is an already exists error, returns false with no error
func CreateKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	prof profile,
) error {
	cmd, err := RepositoryCreateCommand(
		prof,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository create command")
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)

	message := "Failed to create the backup repository"
	switch {
	case err != nil && strings.Contains(err.Error(), ErrCodeOutOfMemory):
		message = message + ": " + OutOfMemoryStr
	case strings.Contains(stderr, errAccessDenied):
		message = message + ": " + errAccessDenied
	}
	if err != nil {
		return errors.Wrap(err, message)
	}

	if err := setGlobalPolicy(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		configFilePath,
		logDirectory,
	); err != nil {
		return errors.Wrap(err, "Failed to set global policy")
	}

	// Set custom maintenance owner in case of successful repository creation
	if err := setCustomMaintenanceOwner(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		configFilePath,
		logDirectory,
	); err != nil {
		log.WithError(err).Print("Failed to set custom kopia maintenance owner, proceeding with default owner")
	}
	return nil
}

// ConnectToKopiaRepository connects to an already existing kopia repository
func ConnectToKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	prof profile,
	pointInTimeConnection strfmt.DateTime,
) error {
	cmd, err := RepositoryConnectCommand(
		prof,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		pointInTimeConnection,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository connect command")
	}

	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(stderr, errInvalidPassword):
		err = errors.WithMessage(err, ErrInvalidPassword.Error())
	case err != nil && strings.Contains(err.Error(), ErrCodeOutOfMemory):
		err = errors.WithMessage(err, ErrOutOfMemory.Error())
	case strings.Contains(stderr, errAccessDenied):
		err = errors.WithMessage(err, ErrAccessDenied.Error())
	case repoNotInitialized(stderr):
		err = errors.WithMessage(err, ErrRepoNotFound.Error())
	}
	return errors.Wrap(err, "Failed to connect to the backup repository")
}

// ConnectToOrCreateKopiaRepository connects to a kopia repository if present or creates if not already present
func ConnectToOrCreateKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	prof profile,
) error {
	err := ConnectToKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		prof,
		strfmt.DateTime{},
	)
	switch {
	case err == nil:
		// If repository connect was successful, we're done!
		return nil
	case IsInvalidPasswordError(err):
		// If connect failed due to invalid password, no need to attempt creation
		return err
	}

	// Create a new repository
	err = CreateKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		prof,
	)

	if err == nil {
		// Successfully created repository, we're done!
		return nil
	}

	// Creation failed. Repository may already exist.
	// Attempt connecting to it.
	// Multiple workers attempting to back up volumes from the
	// same app may race when trying to create the repository if
	// it doesn't yet exist. If this thread initially fails to connect
	// to the repo, then also fails to create a repo, it can try to
	// connect again under the assumption that the repo may have been
	// created by a parallel worker. No harm done if the connect still fails.
	connectErr := ConnectToKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		prof,
		strfmt.DateTime{},
	)

	// Connected successfully after all
	if connectErr == nil {
		return nil
	}

	err = kerrors.Append(err, connectErr)
	return err
}

// repoNotInitialized returns true if the stderr logs contains `repository not initialized` for object stores
// or `no such file or directory` for filestore backend
func repoNotInitialized(stderr string) bool {
	return strings.Contains(stderr, errRepoNotInitialized) || strings.Contains(stderr, errFilesystemRepoNotFound)
}

// IsRepoNotFoundError returns true if the error contains `repository not found` message
func IsRepoNotFoundError(err error) bool {
	return kerrors.FirstMatching(err, func(err error) bool {
		return strings.Contains(err.Error(), errRepoNotFoundStr)
	}) != nil
}

// IsInvalidPasswordError returns true if the error chain has `invalid repository password` error
func IsInvalidPasswordError(err error) bool {
	return kerrors.FirstMatching(err, func(err error) bool {
		return strings.Contains(err.Error(), errInvalidPassword)
	}) != nil
}

// setGlobalPolicy sets the global policy of the kopia repo to keep max-int32 latest
// snapshots and zeros all other time-based retention fields
func setGlobalPolicy(cli kubernetes.Interface, namespace, pod, container, artifactPrefix, encryptionKey, configFilePath, logDirectory string) error {
	cmd := PolicySetGlobalCommand(encryptionKey, configFilePath, logDirectory)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

// PolicySetGlobalCommand creates the command for setting the global policy to the desired settings.
func PolicySetGlobalCommand(encryptionKey, configFilePath, logDirectory string) []string {
	const maxInt32 = 1<<31 - 1

	pc := policyChanges{
		// Retention changes
		keepLatest:  strconv.Itoa(maxInt32),
		keepHourly:  strconv.Itoa(0),
		keepDaily:   strconv.Itoa(0),
		keepWeekly:  strconv.Itoa(0),
		keepMonthly: strconv.Itoa(0),
		keepAnnual:  strconv.Itoa(0),

		// Compression changes
		compressionAlgorithm: s2DefaultComprAlgo,
	}

	return policySetGlobalCommand(encryptionKey, configFilePath, logDirectory, pc)
}

// GetCustomConfigFileAndLogDirectory returns a config file path and log directory based on the hostname
func GetCustomConfigFileAndLogDirectory(hostname string) (string, string) {
	hostname = strings.Replace(hostname, ".", "-", -1)
	configFile := filepath.Join(DefaultConfigDirectory, hostname+".config")
	logDir := filepath.Join(DefaultLogDirectory, hostname)
	return configFile, logDir
}

// setCustomMaintenanceOwner sets custom maintenance owner as hostname@NSUID-maintenance
func setCustomMaintenanceOwner(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	configFilePath,
	logDirectory string,
) error {
	nsUID, err := utils.GetNamespaceUID(context.Background(), cli, namespace)
	if err != nil {
		return errors.Wrap(err, "Failed to get namespace UID")
	}
	newOwner := fmt.Sprintf(MaintenanceOwnerFormat, username, nsUID)
	cmd := MaintenanceSetCommandWithOwner(encryptionKey, configFilePath, logDirectory, newOwner)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

type ErrorType string

const (
	ErrorInvalidPassword ErrorType = errInvalidPassword
	ErrorRepoNotFound    ErrorType = errRepoNotFoundStr
)

// CheckKopiaErrors loops through all the permitted
// error types and returns true on finding a match
func CheckKopiaErrors(err error, errorTypes []ErrorType) bool {
	for _, errorType := range errorTypes {
		if checkKopiaError(err, errorType) {
			return true
		}
	}
	return false
}

func checkKopiaError(err error, errorType ErrorType) bool {
	switch errorType {
	case ErrorInvalidPassword:
		return IsInvalidPasswordError(err)
	case ErrorRepoNotFound:
		return IsRepoNotFoundError(err)
	default:
		return false
	}
}
