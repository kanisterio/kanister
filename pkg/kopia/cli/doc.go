// Copyright 2024 The Kanister Authors.
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

// Package cli provides a command-line interface for Kopia tool.
// It is built on top of the safecli package to ensure robust command construction and execution.
//
// THis package supports the following Kopia commands:
// - policy, snapshot, blob, maintenance, repository, server, manifest and restore.
//
// Example:
//
// package main
//
// import (
//
//	"time"
//
//	"github.com/kanisterio/kanister/pkg/log"
//	"github.com/kanisterio/kanister/pkg/kopia/cli/repository"
//
// )
//
//	func main() {
//		// Initialize debug logger
//		logger := log.Debug()
//
//		// define repo location and other args
//		// The location map should contain the necessary data
//		location := map[string][]byte{
//			"type":   		 []byte("s3"),
//			"bucket":        []byte("bucket.example.com"),
//			"endpoint": 	 []byte("s3.amazonaws.com"),
//			"region": 		 []byte("us-west-2"),
//			"prefix": 		 []byte("projects/backup/"),
//			"skipSSLVerify": []byte("false"),
//		}
//
//		// Create repository creation arguments
//		args := repository.CreateArgs{
//			CommonArgs: cli.CommonArgs{
//				ConfigFilePath: "/etc/kopia/config",
//				LogDirectory:   "/var/log/kopia",
//				LogLevel:       "info",
//				RepoPassword:   "pass12345",
//			},
//			CacheArgs: cli.CacheArgs{
//				CacheDirectory:           "/var/cache/kopia",
//				ContentCacheSizeMB:       1024,
//				ContentCacheSizeLimitMB:  2048,
//				MetadataCacheSizeMB:      256,
//				MetadataCacheSizeLimitMB: 512,
//			},
//			Hostname:         "backup.example.com",
//			Username:         "backup_user",
//			Location:         location,
//			RepoPathPrefix:   "repo/backup",
//			RetentionMode:    "keep_latest",
//			RetentionPeriod:  30 * 24 * time.Hour, // retain for 30 days
//			Logger:           logger,
//		}
//
//		// Create repository
//		command, err := repository.Create(args)
//		if err != nil {
//			logger.Print("Failed to create repository command", log.Field{"error": err})
//			return
//		}
//
//		// use command.Build() to get []string of command and args
//		// ...
//	}
package cli
