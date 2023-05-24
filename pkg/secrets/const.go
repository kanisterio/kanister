package secrets

type LocType string

const (
	// Location types
	LocTypeS3        LocType = "s3"
	LocTypeGCS       LocType = "gcs"
	LocTypeAzure     LocType = "azure"
	LocTypeFilestore LocType = "filestore"
)

const (
	// Location secret keys
	BucketKey        = "bucket"
	EndpointKey      = "endpoint"
	PrefixKey        = "prefix"
	RegionKey        = "region"
	SkipSSLVerifyKey = "skipSSLVerify"
	TypeKey          = "type"
)
