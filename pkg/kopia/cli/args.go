// Copyright 2024 The Kanister Authors.
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

package cli

// The global arguments for Kopia CLI.

// CommonArgs defines the common arguments for Kopia CLI.
type CommonArgs struct {
	ConfigFilePath string // ConfigFilePath is the path to the config file.
	LogDirectory   string // LogDirectory is the directory where logs are stored.
	LogLevel       string // LogLevel is the level of logging.
	RepoPassword   string // RepoPassword is the password for the repository.
}

// CacheArgs defines the cache arguments for Kopia CLI.
type CacheArgs struct {
	CacheDirectory           string // CacheDirectory is the directory where cache is stored.
	ContentCacheSizeMB       int    // ContentCacheSizeMB is the size of the content cache in MB.
	ContentCacheSizeLimitMB  int    // ContentCacheSizeLimitMB is the maximum size of the content cache in MB.
	MetadataCacheSizeMB      int    // MetadataCacheSizeMB is the size of the metadata cache in MB.
	MetadataCacheSizeLimitMB int    // MetadataCacheSizeLimitMB is the maximum size of the metadata cache in MB.
}
