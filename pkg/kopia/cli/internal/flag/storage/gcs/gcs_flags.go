package gcs

import "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"

//
// GCS flags.
//

// Bucket creates a new GCS bucket flag with a given bucket name.
func Bucket(bucket string) flag.Applier {
	return flag.NewStringFlag("--bucket", bucket)
}

// Prefix creates a new GCS prefix flag with a given prefix.
func Prefix(prefix string) flag.Applier {
	return flag.NewStringFlag("--prefix", prefix)
}

// CredentialsFile creates a new GCS credentials file flag with a given file path.
func CredentialsFile(filePath string) flag.Applier {
	return flag.NewStringFlag("--credentials-file", filePath)
}
