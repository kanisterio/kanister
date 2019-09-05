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

package restic

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

type ResticDataSuite struct{}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ResticDataSuite{})

func (s *ResticDataSuite) TestGetSnapshotIDFromTag(c *C) {
	for _, tc := range []struct {
		log      string
		expected string
		checker  Checker
	}{
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: "7c0bfeb9", checker: IsNil},
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"},{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: "", checker: NotNil},
		{log: `null`, expected: "", checker: NotNil},
	} {
		id, err := SnapshotIDFromSnapshotLog(tc.log)
		c.Assert(err, tc.checker)
		c.Assert(id, Equals, tc.expected)

	}
}

func (s *ResticDataSuite) TestGetSnapshotID(c *C) {
	for _, tc := range []struct {
		log      string
		expected string
	}{
		{"snapshot 1a2b3c4d saved", "1a2b3c4d"},
		{"snapshot 123abcd", ""},
		{"Invalid message", ""},
		{"snapshot abc123\n saved", ""},
	} {
		id := SnapshotIDFromBackupLog(tc.log)
		c.Check(id, Equals, tc.expected, Commentf("Failed for log: %s", tc.log))
	}
}

func (s *ResticDataSuite) TestResticArgs(c *C) {
	for _, tc := range []struct {
		profile  *param.Profile
		repo     string
		password string
		expected []string
	}{
		{
			profile: &param.Profile{
				Location: v1alpha1.Location{
					Type:     v1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint",
				},
				Credential: param.Credential{
					Type: param.CredentialTypeKeyPair,
					KeyPair: &param.KeyPair{
						ID:     "id",
						Secret: "secret",
					},
				},
			},
			repo:     "repo",
			password: "my-secret",
			expected: []string{
				"export AWS_ACCESS_KEY_ID=id\n",
				"export AWS_SECRET_ACCESS_KEY=secret\n",
				"export RESTIC_REPOSITORY=s3:endpoint/repo\n",
				"export RESTIC_PASSWORD=my-secret\n",
				"restic",
			},
		},
		{
			profile: &param.Profile{
				Location: v1alpha1.Location{
					Type:     v1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint/", // Remove trailing slash
				},
				Credential: param.Credential{
					Type: param.CredentialTypeKeyPair,
					KeyPair: &param.KeyPair{
						ID:     "id",
						Secret: "secret",
					},
				},
			},
			repo:     "repo",
			password: "my-secret",
			expected: []string{
				"export AWS_ACCESS_KEY_ID=id\n",
				"export AWS_SECRET_ACCESS_KEY=secret\n",
				"export RESTIC_REPOSITORY=s3:endpoint/repo\n",
				"export RESTIC_PASSWORD=my-secret\n",
				"restic",
			},
		},
		{
			profile: &param.Profile{
				Location: v1alpha1.Location{
					Type:     v1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint/////////", // Also remove all of the trailing slashes
				},
				Credential: param.Credential{
					Type: param.CredentialTypeKeyPair,
					KeyPair: &param.KeyPair{
						ID:     "id",
						Secret: "secret",
					},
				},
			},
			repo:     "repo",
			password: "my-secret",
			expected: []string{
				"export AWS_ACCESS_KEY_ID=id\n",
				"export AWS_SECRET_ACCESS_KEY=secret\n",
				"export RESTIC_REPOSITORY=s3:endpoint/repo\n",
				"export RESTIC_PASSWORD=my-secret\n",
				"restic",
			},
		},
	} {
		c.Assert(resticArgs(tc.profile, tc.repo, tc.password), DeepEquals, tc.expected)
	}
}

func (s *ResticDataSuite) TestGetSnapshotStatsFromStatsLog(c *C) {
	for _, tc := range []struct {
		log          string
		expectedfc   string
		expectedsize string
	}{
		{log: "Total File Count:   9", expectedfc: "9", expectedsize: ""},
		{log: "Total Size:   10.322 KiB", expectedfc: "", expectedsize: "10.322 KiB"},
		{log: "sudhufehfuijbfjbruifhoiwhf", expectedfc: "", expectedsize: ""},
		{log: "      Total File Count:   9", expectedfc: "9", expectedsize: ""},
		{log: "    Total Size:   10.322 KiB", expectedfc: "", expectedsize: "10.322 KiB"},
	} {
		fc, s := SnapshotStatsFromStatsLog(tc.log)
		c.Assert(fc, Equals, tc.expectedfc)
		c.Assert(s, Equals, tc.expectedsize)
	}
}
