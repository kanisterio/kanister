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

package repository

import (
	"fmt"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/safecli/command"
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
)

func TestRepositoryOptions(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name: "optHostname",
		Argument: command.NewArguments(
			optHostname("host"),
			optHostname(""), // no output
		),
		ExpectedCLI: []string{"cmd", "--override-hostname=host"},
	},
	{
		Name: "optUsername",
		Argument: command.NewArguments(
			optUsername("user"),
			optUsername(""), // no output
		),
		ExpectedCLI: []string{"cmd", "--override-username=user"},
	},
	{
		Name: "optBlobRetention",
		Argument: command.NewArguments(
			optBlobRetention(retentionMode, retentionPeriod),
			optBlobRetention("", 0), // no output
		),
		ExpectedCLI: []string{"cmd", "--retention-mode=Locked", "--retention-period=15m0s"},
	},
	{
		Name:        "optStorage FS",
		Argument:    optStorage(locFS, "repoPathPrefix", nil),
		ExpectedCLI: []string{"cmd", "filesystem", "--path=/mnt/data/test-prefix/repoPathPrefix/"},
	},
	{
		Name:        "optStorage Azure",
		Argument:    optStorage(locAzure, "repoPathPrefix", nil),
		ExpectedCLI: []string{"cmd", "azure", "--container=test-bucket", "--prefix=test-prefix/repoPathPrefix/"},
	},
	{
		Name:        "optStorage S3",
		Argument:    optStorage(locS3, "repoPathPrefix", nil),
		ExpectedCLI: []string{"cmd", "s3", "--region=test-region", "--bucket=test-bucket", "--endpoint=test-endpoint", "--prefix=test-prefix/repoPathPrefix/"},
	},
	{
		Name:        "optStorage S3Compliant",
		Argument:    optStorage(locS3Compliant, "repoPathPrefix", nil),
		ExpectedCLI: []string{"cmd", "s3", "--region=test-region", "--bucket=test-bucket", "--endpoint=test-endpoint", "--prefix=test-prefix/repoPathPrefix/"},
	},
	{
		Name:        "optStorage FTP Unsupported",
		Argument:    optStorage(locFTP, "repoPathPrefix", nil),
		ExpectedErr: cli.ErrUnsupportedStorage,
	},
	{
		Name: "optReadOnly",
		Argument: command.NewArguments(
			optReadOnly(true),
			optReadOnly(false), // no output
		),
		ExpectedCLI: []string{"cmd", "--readonly"},
	},
	{
		Name: "optPointInTime supported only for azure and s3",
		Argument: func() command.Arguments {
			locations := []internal.Location{
				locFS,          // no output
				locAzure,       // idx: 1
				locGCS,         // no output
				locS3,          // idx: 3
				locS3Compliant, // idx: 4
			}
			var args command.Arguments
			for idx, l := range locations {
				t, _ := strfmt.ParseDateTime(
					fmt.Sprintf("2021-02-%02dT01:02:03.000Z", idx),
				)
				args = append(args, optPointInTime(l, t))
			}
			return args
		}(),
		ExpectedCLI: []string{"cmd",
			"--point-in-time=2021-02-01T01:02:03.000Z",
			"--point-in-time=2021-02-03T01:02:03.000Z",
			"--point-in-time=2021-02-04T01:02:03.000Z",
		},
	},
	{
		Name:        "optServerURL with ServerURL should return option",
		Argument:    optServerURL("http://test-server"),
		ExpectedCLI: []string{"cmd", "--url=http://test-server"},
	},
	{
		Name:        "optServerURL with empty ServerURL should return error",
		Argument:    optServerURL(""),
		ExpectedErr: cli.ErrInvalidServerURL,
	},
	{
		Name: "optServerCertFingerprint",
		Argument: command.NewArguments(
			optServerCertFingerprint("fingerprint"),
			optServerCertFingerprint(""), // no output
		),
		ExpectedCLI: []string{"cmd", "--server-cert-fingerprint=fingerprint"},
	},
}})
