package repository

import (
	"errors"
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
	ArtifactPrefix     string
}

func kopiaBlobStoreArgs(params *StorageCommandParams) (logsafe.Cmd, error) {
	locType := locationType(params.LocationSecret)
	switch locationType(params.LocationSecret) {
	case locTypeFilestore:
		return filesystemArgs(params.LocationSecret, params.ArtifactPrefix), nil
	case locTypeS3:
		return kopiaS3Args(params.LocationSecret, params.LocationCredSecret, params.AssumeRoleDuration, params.ArtifactPrefix)
	case locTypeGCS:
		return kopiaGCSArgs(params.LocationSecret, params.ArtifactPrefix), nil
	case locTypeAzure:
		return kopiaAzureArgs(params.LocationSecret, params.LocationCredSecret, params.ArtifactPrefix)
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported type for the location: %s", locType))
	}
}
