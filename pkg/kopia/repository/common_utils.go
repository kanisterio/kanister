package repository

import (
	"path"
)

// GenerateFullRepoPath defines the manner in which a location-specific prefix
// string is joined with a repository-specific prefix to generate the full path
// for a kopia repository.
func GenerateFullRepoPath(locPrefix, artifactPrefix string) string {
	if locPrefix != "" {
		return path.Join(locPrefix, artifactPrefix) + "/"
	}

	return artifactPrefix
}
