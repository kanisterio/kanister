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

package opts_test

import (
	"testing"

	"github.com/kanisterio/safecli/command"
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
)

func TestCacheOptions(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "CacheDirectory",
		Argument:    command.NewArguments(opts.CacheDirectory(""), opts.CacheDirectory("/path/to/cache")),
		ExpectedCLI: []string{"cmd", "--cache-directory=/tmp/kopia-cache", "--cache-directory=/path/to/cache"},
	},
	{
		Name:        "ContentCacheSizeLimitMB",
		Argument:    command.NewArguments(opts.ContentCacheSizeLimitMB(0), opts.ContentCacheSizeLimitMB(1024)),
		ExpectedCLI: []string{"cmd", "--content-cache-size-limit-mb=0", "--content-cache-size-limit-mb=1024"},
	},
	{
		Name:        "MetadataCacheSizeLimitMB",
		Argument:    command.NewArguments(opts.MetadataCacheSizeLimitMB(0), opts.MetadataCacheSizeLimitMB(1024)),
		ExpectedCLI: []string{"cmd", "--metadata-cache-size-limit-mb=0", "--metadata-cache-size-limit-mb=1024"},
	},
	{
		Name: "Cache",
		Argument: opts.Cache(args.Cache{
			CacheDirectory:           "/path/to/cache",
			ContentCacheSizeLimitMB:  1024,
			MetadataCacheSizeLimitMB: 2048,
		}),
		ExpectedCLI: []string{"cmd", "--cache-directory=/path/to/cache", "--content-cache-size-limit-mb=1024", "--metadata-cache-size-limit-mb=2048"},
	},
}})
