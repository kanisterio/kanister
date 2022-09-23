package storage

import (
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	gcsSubCommand       = "gcs"
	credentialsFileFlag = "--credentials-file"
	gcsBucketFlag       = "--bucket"
	gcsPrefixFlag       = "--prefix"
)

func kopiaGCSArgs(locationSecret *v1.Secret, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(prefix(locationSecret), artifactPrefix)

	args := logsafe.NewLoggable(gcsSubCommand)
	args = args.AppendLoggableKV(gcsBucketFlag, bucketName(locationSecret))
	args = args.AppendLoggableKV(credentialsFileFlag, consts.GoogleCloudCredsFilePath)
	return args.AppendLoggableKV(gcsPrefixFlag, artifactPrefix)
}
