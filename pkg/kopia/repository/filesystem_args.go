package repository

import (
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	filesystemSubCommand = "filesystem"
	pathFlag             = "--path"
	// DefaultFSMountPath is the mount path for the file store PVC on Kopia API server
	DefaultFSMountPath = "/mnt/data"
)

func filesystemArgs(locationSecret *v1.Secret, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(prefix(locationSecret), artifactPrefix)

	args := logsafe.NewLoggable(filesystemSubCommand)
	return args.AppendLoggableKV(pathFlag, DefaultFSMountPath+"/"+artifactPrefix)
}
