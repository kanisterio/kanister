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

package command

// KopiaBinaryName is the name of the Kopia binary.
var (
	KopiaBinaryName = Command{"kopia"}
)

// Repository commands.
var (
	Repository = Command{"repository"}
	Create     = Command{"create"}
)

// Repository storage sub commands.
var (
	FileSystem = Command{"filesystem"}
	GCS        = Command{"gcs"}
	Azure      = Command{"azure"}
	S3         = Command{"s3"}
)
