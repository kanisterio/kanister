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
	v1 "k8s.io/api/core/v1"

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
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"},{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: "7c0bfeb9", checker: IsNil},
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
		{
			profile: &param.Profile{
				Location: v1alpha1.Location{
					Type:     v1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint", // Also remove all of the trailing slashes
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &v1.Secret{
						Type: "secrets.kanister.io/aws",
						Data: map[string][]byte{
							"access_key_id":     []byte("id"),
							"secret_access_key": []byte("secret"),
						},
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
					Endpoint: "endpoint", // Also remove all of the trailing slashes
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &v1.Secret{
						Type: "secrets.kanister.io/aws",
						Data: map[string][]byte{
							"access_key_id":     []byte("id"),
							"secret_access_key": []byte("secret"),
							"session_token":     []byte("token"),
						},
					},
				},
			},
			repo:     "repo",
			password: "my-secret",
			expected: []string{
				"export AWS_ACCESS_KEY_ID=id\n",
				"export AWS_SECRET_ACCESS_KEY=secret\n",
				"export AWS_SESSION_TOKEN=token\n",
				"export RESTIC_REPOSITORY=s3:endpoint/repo\n",
				"export RESTIC_PASSWORD=my-secret\n",
				"restic",
			},
		},
	} {
		args, err := resticArgs(tc.profile, tc.repo, tc.password)
		c.Assert(err, IsNil)
		c.Assert(args, DeepEquals, tc.expected)
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
		_, fc, s := SnapshotStatsFromStatsLog(tc.log)
		c.Assert(fc, Equals, tc.expectedfc)
		c.Assert(s, Equals, tc.expectedsize)
	}
}

func (s *ResticDataSuite) TestGetSnapshotStatsModeFromStatsLog(c *C) {
	for _, tc := range []struct {
		log      string
		expected string
	}{
		{log: "Stats for all snapshots in restore-size mode:", expected: "restore-size"},
		{log: "Stats for 7e17e764 in restore-size mode:", expected: "restore-size"},
		{log: "Stats for all snapshots in raw-data mode:", expected: "raw-data"},
		{log: "Stats for all snapshots in blobs-per-file mode:", expected: "blobs-per-file"},
		{log: "sudhufehfuijbfjbruifhoiwhf", expected: ""},
	} {
		mode := SnapshotStatsModeFromStatsLog(tc.log)
		c.Assert(mode, Equals, tc.expected)
	}
}

func (s *ResticDataSuite) TestGetSnapshotIDsFromSnapshotCommand(c *C) {
	for _, tc := range []struct {
		log      string
		expected []string
		checker  Checker
	}{
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: []string{"7c0bfeb9"}, checker: IsNil},
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb67"},{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: []string{"7c0bfeb67", "7c0bfeb9"}, checker: IsNil},
		{log: `null`, expected: []string(nil), checker: NotNil},
	} {
		ids, err := SnapshotIDsFromSnapshotCommand(tc.log)
		c.Assert(err, tc.checker)
		c.Assert(ids, DeepEquals, tc.expected)

	}
}

func (s *ResticDataSuite) TestIsPasswordIncorrect(c *C) {
	for _, tc := range []struct {
		log      string
		expected bool
	}{
		{log: `Fatal: create key in repository at s3:s3.amazonaws.com/ddixit-test/testDir-dz4dv failed: repository master key and config already initialized`, expected: false},
		{log: `Fatal: wrong password or no key found`, expected: true},
		{log: `Fatal: unable to open config file: Stat: The specified key does not exist.
Is there a repository at the following location?
s3:s3.amazonaws.com/ddixit-test/testDir-dz`, expected: false},
	} {
		output := IsPasswordIncorrect(tc.log)
		c.Assert(output, Equals, tc.expected)
	}
}

func (s *ResticDataSuite) TestDoesRepoExist(c *C) {
	for _, tc := range []struct {
		log      string
		expected bool
	}{
		{log: `Fatal: create key in repository at s3:s3.amazonaws.com/ddixit-test/testDir-dz4dv failed: repository master key and config already initialized`, expected: false},
		{log: `Fatal: wrong password or no key found`, expected: false},
		{log: `Fatal: unable to open config file: Stat: The specified key does not exist.
Is there a repository at the following location?
s3:s3.amazonaws.com/ddixit-test/testDir-dz`, expected: true},
	} {
		output := DoesRepoExist(tc.log)
		c.Assert(output, Equals, tc.expected)
	}
}
