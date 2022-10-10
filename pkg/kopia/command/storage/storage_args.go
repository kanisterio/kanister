package storage

import (
	"fmt"
	"time"

	"github.com/kanisterio/kanister/pkg/logsafe"
	v1 "k8s.io/api/core/v1"
)

type StorageCommandParams struct {
	// S3 specific param
	AssumeRoleDuration time.Duration
	// Common params
	LocationSecret     *v1.Secret
	LocationCredSecret *v1.Secret
	RepoPathPrefix     string
}

func KopiaBlobStoreArgs(params *StorageCommandParams) (logsafe.Cmd, error) {
	locType := locationType(params.LocationSecret)
	switch locationType(params.LocationSecret) {
	case locTypeFilestore:
		return filesystemArgs(params.LocationSecret, params.RepoPathPrefix), nil
	case locTypeS3:
		return kopiaS3Args(params.LocationSecret, params.LocationCredSecret, params.AssumeRoleDuration, params.RepoPathPrefix)
	case locTypeGCS:
		return kopiaGCSArgs(params.LocationSecret, params.RepoPathPrefix), nil
	case locTypeAzure:
		return kopiaAzureArgs(params.LocationSecret, params.LocationCredSecret, params.RepoPathPrefix)
	default:
		return nil, fmt.Errorf("unsupported type for the location: %s", locType)
	}
}
