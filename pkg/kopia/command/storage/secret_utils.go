package storage

type locType string

const (
	locTypeS3        locType = "s3"
	locTypeGCS       locType = "gcs"
	locTypeAzure     locType = "azure"
	locTypeFilestore locType = "filestore"
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

func locationType(m map[string]string) locType {
	return locType(m[typeKey])
}

func SkipCredentialSecretMount(m map[string]string) bool {
	return locType(m[typeKey]) == locTypeFilestore
}
