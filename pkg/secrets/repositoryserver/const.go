// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repositoryserver

import (
	corev1 "k8s.io/api/core/v1"
)

type LocType string

const (
	LocTypeS3          LocType = "s3"
	LocTypes3Compliant LocType = "s3Compliant"
	LocTypeGCS         LocType = "gcs"
	LocTypeAzure       LocType = "azure"
	LocTypeFilestore   LocType = "filestore"

	// Location represents the storage location secret type for kopia repository server
	Location corev1.SecretType = "secrets.kanister.io/storage-location"
	// RepositoryPasswordSecret represents the kopia repository passowrd secret type
	RepositoryPasswordSecret corev1.SecretType = "secrets.kanister.io/kopia-repository/password"
	// AdminCredentialsSecret represents the kopia server admin credentials secret type
	AdminCredentialsSecret corev1.SecretType = "secrets.kanister.io/kopia-repository/serveradmin"
)

const (
	// Location secret keys
	BucketKey        = "bucket"
	EndpointKey      = "endpoint"
	PrefixKey        = "prefix"
	RegionKey        = "region"
	SkipSSLVerifyKey = "skipSSLVerify"
	TypeKey          = "type"
	// Location secret key to be used only for filestore location type
	ClaimNameKey = "claimName"

	// Kopia Repository Server secret keys
	RepoPasswordKey  = "repo-password"
	AdminUsernameKey = "username"
	AdminPasswordKey = "password"
)
