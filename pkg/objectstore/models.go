package objectstore

// ProviderConfig describes the config for the object store (which provider to use)
type ProviderConfig struct {
	// object store type
	Type ProviderType
	// Endpoint used to access the object store. It can be implicit for
	// stores from certain cloud providers such as AWS. In that case it can
	// be empty
	Endpoint string
	// If true, disable SSL verification. If false (the default), SSL
	// verification is enabled.
	SkipSSLVerify bool
}

// SecretAws AWS keys
type SecretAws struct {
	// access key Id
	AccessKeyID string
	// secret access key
	SecretAccessKey string
}

// SecretAzure Azure credentials
type SecretAzure struct {
	// storage account
	StorageAccount string
	// storage key
	StorageKey string
}

// SecretGcp GCP credentials
type SecretGcp struct {
	// project Id
	ProjectID string
	// base64 encoded service account key
	ServiceKey string
}

// Secret contains the credentials for different providers
type Secret struct {
	// aws
	Aws *SecretAws
	// azure
	Azure *SecretAzure
	// gcp
	Gcp *SecretGcp
	// type
	Type SecretType
}
