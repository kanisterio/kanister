package storage

type LocType string

const (
	LocTypeS3        LocType = "s3"
	LocTypeGCS       LocType = "gcs"
	LocTypeAzure     LocType = "azure"
	LocTypeFilestore LocType = "filestore"
)

const (
	bucketKey        = "bucket"
	endpointKey      = "endpoint"
	prefixKey        = "prefix"
	regionKey        = "region"
	skipSSLVerifyKey = "skipSSLVerify"
	typeKey          = "type"
)

func bucketName(m map[string]string) string {
	return m[bucketKey]
}

func endpoint(m map[string]string) string {
	return m[endpointKey]
}

func prefix(m map[string]string) string {
	return m[prefixKey]
}

func region(m map[string]string) string {
	return m[regionKey]
}

func skipSSLVerify(m map[string]string) bool {
	v := m[skipSSLVerifyKey]
	return v == "true"
}

func locationType(m map[string]string) LocType {
	return LocType(m[typeKey])
}
