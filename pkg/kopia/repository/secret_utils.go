package repository

import (
	corev1 "k8s.io/api/core/v1"
)

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

func bucketName(s *corev1.Secret) string {
	return s.StringData["bucket"]
}

func endpoint(s *corev1.Secret) string {
	return s.StringData["endpoint"]
}

func prefix(s *corev1.Secret) string {
	return s.StringData["prefix"]
}

func region(s *corev1.Secret) string {
	return s.StringData["region"]
}

func skipSSLVerify(s *corev1.Secret) bool {
	v := s.StringData["skipSSLVerify"]
	return v == "true"
}

func locationType(s *corev1.Secret) locType {
	return locType(s.StringData["type"])
}
