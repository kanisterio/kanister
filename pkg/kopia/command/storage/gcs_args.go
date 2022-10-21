package storage

import (
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	gcsSubCommand       = "gcs"
	credentialsFileFlag = "--credentials-file"
	gcsBucketFlag       = "--bucket"
	gcsPrefixFlag       = "--prefix"
)

func kopiaGCSArgs(location map[string]string, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(prefix(location), artifactPrefix)

	args := logsafe.NewLoggable(gcsSubCommand)
	args = args.AppendLoggableKV(gcsBucketFlag, bucketName(location))
	args = args.AppendLoggableKV(credentialsFileFlag, consts.GoogleCloudCredsFilePath)
	return args.AppendLoggableKV(gcsPrefixFlag, artifactPrefix)
}
