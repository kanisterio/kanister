package secrets

import v1 "k8s.io/api/core/v1"

type LocType string

const (
	// Location types
	LocTypeS3        LocType = "s3"
	LocTypeGCS       LocType = "gcs"
	LocTypeAzure     LocType = "azure"
	LocTypeFilestore LocType = "filestore"

	// LocationSecretType represents the storage location secret type for kopia repository server
	Location v1.SecretType = "secrets.kanister.io/storage-location"
	// RepositoryPasswordSecretType represents the kopia repository passowrd secret type
	RepositoryPassword v1.SecretType = "secrets.kanister.io/kopia-repository/password"
	// RepositoryServerAdminCredentialsSecretType represents the kopia server admin credentials secret type
	RepositoryServerAdminCredentials v1.SecretType = "secrets.kanister.io/kopia-repository/serveradmin"
	// LocationTypeKey represents the key used to define the location type in
	// the kopia repository server location secret
	LocationTypeKey string = "type"
)
