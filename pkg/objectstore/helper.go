package objectstore

import "context"

// GetOrCreateBucket is a helper function to access the package level getOrCreateBucket
func GetOrCreateBucket(ctx context.Context, p Provider, bucketName string, region string) (Directory, error) {
	return p.getOrCreateBucket(ctx, bucketName, region)
}

// IsS3Provider is a helper function to find out if a provider is an s3Provider
func IsS3Provider(p Provider) bool {
	if _, ok := p.(*s3Provider); ok {
		return true
	}
	return false
}
