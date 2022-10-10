package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

type StorageCommandParams struct {
	// S3 specific param
	AssumeRoleDuration time.Duration
	// Common params
	Location       map[string]string
	Credentials    map[string]string
	RepoPathPrefix string
}

func KopiaBlobStoreArgs(params *StorageCommandParams) (logsafe.Cmd, error) {
	paramsJson, _ := json.Marshal(params)
	fmt.Println("PARAMS: ", string(paramsJson))
	locType := locationType(params.Location)
	switch locationType(params.Location) {
	case locTypeFilestore:
		return filesystemArgs(params.Location, params.RepoPathPrefix), nil
	case locTypeS3:
		return kopiaS3Args(params.Location, params.Credentials, params.AssumeRoleDuration, params.RepoPathPrefix)
	case locTypeGCS:
		return kopiaGCSArgs(params.Location, params.RepoPathPrefix), nil
	case locTypeAzure:
		return kopiaAzureArgs(params.Location, params.Credentials, params.RepoPathPrefix)
	default:
		return nil, fmt.Errorf("unsupported type for the location: %s", locType)
	}
}
