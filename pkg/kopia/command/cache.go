package command

import (
	"strconv"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

// CacheArgs has fields that can be used to set
// cache settings for different kopia repository operations
type CacheArgs struct {
	ContentCacheLimitMB  int
	MetadataCacheLimitMB int
}

func (c CacheArgs) kopiaCacheArgs(args logsafe.Cmd, cacheDirectory string) logsafe.Cmd {
	args = args.AppendLoggableKV(cacheDirectoryFlag, cacheDirectory)
	// The hard limit flags for cache would be set using the env variables that are passed through
	// helm settings
	// Soft limit flags would be set to the default values by kopia automatically
	// in connectOptions.setup() function as shown here - https://github.com/kopia/kopia/blob/master/cli/command_repository_connect.go#L71
	args = args.AppendLoggableKV(contentCacheSizeLimitMBFlag, strconv.Itoa(c.ContentCacheLimitMB))
	args = args.AppendLoggableKV(metadataCacheSizeLimitMBFlag, strconv.Itoa(c.MetadataCacheLimitMB))
	return args
}
