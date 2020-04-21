// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package objectstore

import "context"

// GetOrCreateBucket is a helper function to access the package level getOrCreateBucket
func GetOrCreateBucket(ctx context.Context, p Provider, bucketName string) (Directory, error) {
	return p.getOrCreateBucket(ctx, bucketName)
}

// IsS3Provider is a helper function to find out if a provider is an s3Provider
func IsS3Provider(p Provider) bool {
	if _, ok := p.(*s3Provider); ok {
		return true
	}
	return false
}
