package storage

import (
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	filesystemSubCommand = "filesystem"
	pathFlag             = "--path"
	// DefaultFSMountPath is the mount path for the file store PVC on Kopia API server
	DefaultFSMountPath = "/mnt/data"
)

func filesystemArgs(location map[string]string, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(prefix(location), artifactPrefix)

	args := logsafe.NewLoggable(filesystemSubCommand)
	return args.AppendLoggableKV(pathFlag, DefaultFSMountPath+"/"+artifactPrefix)
}
