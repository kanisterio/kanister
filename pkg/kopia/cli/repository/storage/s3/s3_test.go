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

package s3

import (
	"strconv"
	"testing"

	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	intlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
)

func TestNewS3(t *testing.T) { check.TestingT(t) }

func newLocation(prefix, endpoint, region, bucket string, skipSSLVerify bool) internal.Location {
	return internal.Location{
		"prefix":        []byte(prefix),
		"endpoint":      []byte(endpoint),
		"region":        []byte(region),
		"bucket":        []byte(bucket),
		"skipSSLVerify": []byte(strconv.FormatBool(skipSSLVerify)),
	}
}

var _ = check.Suite(&ArgSuite{Cmd: "cmd", Arguments: []ArgTest{
	{
		test: test.ArgumentTest{
			Name: "NewS3",
			ExpectedCLI: []string{"cmd", "s3",
				"--region=region",
				"--bucket=bucket",
				"--endpoint=endpoint/path",
				"--prefix=prefix/repoPath/",
				"--disable-tls",
				"--disable-tls-verification",
			},
		},
		location: newLocation("prefix", "http://endpoint/path/", "region", "bucket", true),
		repoPath: "repoPath",
		Logger:   &intlog.StringLogger{},
		LoggerRegex: []string{
			"Removing leading",
			"Removing trailing",
		},
	},
	{
		test: test.ArgumentTest{
			Name: "NewS3 w/o logger should not panic",
			ExpectedCLI: []string{"cmd", "s3",
				"--region=region",
				"--bucket=bucket",
				"--endpoint=endpoint/path",
				"--prefix=prefix/repoPath/",
				"--disable-tls",
				"--disable-tls-verification",
			},
		},
		location: newLocation("prefix", "http://endpoint/path/", "region", "bucket", true),
		repoPath: "repoPath",
	},
	{
		test: test.ArgumentTest{
			Name: "NewS3 with empty repoPath and https endpoint",
			ExpectedCLI: []string{"cmd", "s3",
				"--region=region",
				"--bucket=bucket",
				"--endpoint=endpoint/path",
				"--prefix=prefix/",
			},
		},
		location: newLocation("prefix", "https://endpoint/path/", "region", "bucket", false),
		repoPath: "",
		Logger:   &intlog.StringLogger{},
		LoggerRegex: []string{
			"Removing leading",
			"Removing trailing",
		},
	},
	{
		test: test.ArgumentTest{
			Name: "NewS3 with empty repoPath and endpoint",
			ExpectedCLI: []string{"cmd", "s3",
				"--region=region",
				"--bucket=bucket",
				"--prefix=prefix/",
				"--disable-tls-verification",
			},
		},
		location:    newLocation("prefix", "", "region", "bucket", true),
		repoPath:    "",
		Logger:      &intlog.StringLogger{},
		LoggerRegex: []string{""}, // no output expected
	},
	{
		test: test.ArgumentTest{
			Name: "NewS3 with empty repoPath, prefix and endpoint",
			ExpectedCLI: []string{"cmd", "s3",
				"--region=region",
				"--bucket=bucket",
				"--prefix=",
				"--disable-tls-verification",
			},
		},
		location:    newLocation("", "", "region", "bucket", true),
		repoPath:    "",
		Logger:      &intlog.StringLogger{},
		LoggerRegex: []string{""}, // no output expected
	},
	{
		test: test.ArgumentTest{
			Name:        "NewS3 with empty repoPath, prefix, endpoint and bucket",
			ExpectedErr: cli.ErrInvalidBucketName,
		},
		location:    internal.Location{},
		repoPath:    "",
		Logger:      &intlog.StringLogger{},
		LoggerRegex: []string{""}, // no output expected
	},
}})
