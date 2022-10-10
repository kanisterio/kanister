// Copyright 2022 The Kanister Authors.
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

package errors

const (
	ErrInvalidPasswordStr        = "invalid repository password"
	ErrOutOfMemoryStr            = "kanister-tools container ran out of memory"
	ErrAccessDeniedStr           = "Access Denied"
	ErrRepoNotFoundStr           = "repository not found"
	ErrRepoNotInitializedStr     = "repository not initialized in the provided storage"
	ErrFilesystemRepoNotFoundStr = "no such file or directory"
	ErrCodeOutOfMemoryStr        = "command terminated with exit code 137"
	ErrBucketDoesNotExistStr     = "bucket doesn't exist"
	ErrUnableToListFromBucketStr = "unable to list from the bucket"
)
