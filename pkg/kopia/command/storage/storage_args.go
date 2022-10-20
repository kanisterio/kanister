package storage

import (
	"fmt"
	"time"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

type StorageCommandParams struct {
	// S3 specific param
	AssumeRoleDuration time.Duration
	// Common params
	Location       map[string]string
	RepoPathPrefix string
}

func KopiaBlobStoreArgs(params *StorageCommandParams) (logsafe.Cmd, error) {
	LocType := locationType(params.Location)
	switch locationType(params.Location) {
	case LocTypeFilestore:
		return filesystemArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeS3:
		return kopiaS3Args(params.Location, params.AssumeRoleDuration, params.RepoPathPrefix)
	case LocTypeGCS:
		return kopiaGCSArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeAzure:
		return kopiaAzureArgs(params.Location, params.RepoPathPrefix)
	default:
		return nil, fmt.Errorf("unsupported type for the location: %s", LocType)
	}
}
