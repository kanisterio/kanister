package command

import (
	"strconv"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

type CacheArgs struct {
	ContentCacheMB       int
	MetadataCacheMB      int
	ContentCacheLimitMB  int
	MetadataCacheLimitMB int
}

func (c CacheArgs) kopiaCacheArgs(args logsafe.Cmd, cacheDirectory string) logsafe.Cmd {
	args = args.AppendLoggableKV(cacheDirectoryFlag, cacheDirectory)
	args = args.AppendLoggableKV(contentCacheSizeMBFlag, strconv.Itoa(c.ContentCacheMB))
	args = args.AppendLoggableKV(metadataCacheSizeMBFlag, strconv.Itoa(c.MetadataCacheMB))
	args = args.AppendLoggableKV(contentCacheSizeLimitMBFlag, strconv.Itoa(c.ContentCacheLimitMB))
	args = args.AppendLoggableKV(metadataCacheSizeLimitMBFlag, strconv.Itoa(c.MetadataCacheLimitMB))
	return args
}
