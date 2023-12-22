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
	corev1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/config"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
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
				Location: crv1alpha1.Location{
					Type:     crv1alpha1.LocationTypeS3Compliant,
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
				Location: crv1alpha1.Location{
					Type:     crv1alpha1.LocationTypeS3Compliant,
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
				Location: crv1alpha1.Location{
					Type:     crv1alpha1.LocationTypeS3Compliant,
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
				Location: crv1alpha1.Location{
					Type:     crv1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint", // Also remove all of the trailing slashes
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &corev1.Secret{
						Type: "secrets.kanister.io/aws",
						Data: map[string][]byte{
							secrets.AWSAccessKeyID:     []byte("id"),
							secrets.AWSSecretAccessKey: []byte("secret"),
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
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeAzure,
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &corev1.Secret{
						Type: corev1.SecretType(secrets.AzureSecretType),
						Data: map[string][]byte{
							secrets.AzureStorageAccountID:  []byte("id"),
							secrets.AzureStorageAccountKey: []byte("secret"),
						},
					},
				},
			},
			repo:     "repo",
			password: "my-secret",
			expected: []string{
				"export AZURE_ACCOUNT_NAME=id\n",
				"export AZURE_ACCOUNT_KEY=secret\n",
				"export RESTIC_REPOSITORY=azure:repo/\n",
				"export RESTIC_PASSWORD=my-secret\n",
				"restic",
			},
		},
		{
			profile: &param.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeAzure,
				},
				Credential: param.Credential{
					Type: param.CredentialTypeKeyPair,
					Secret: &corev1.Secret{
						Type: corev1.SecretType(secrets.AzureSecretType),
						Data: map[string][]byte{
							secrets.AzureStorageAccountID:  []byte("id"),
							secrets.AzureStorageAccountKey: []byte("secret"),
						},
					},
					KeyPair: &param.KeyPair{
						ID:     "kpID",
						Secret: "kpSecret",
					},
				},
			},
			repo:     "repo",
			password: "my-secret",
			expected: []string{
				"export AZURE_ACCOUNT_NAME=kpID\n",
				"export AZURE_ACCOUNT_KEY=kpSecret\n",
				"export RESTIC_REPOSITORY=azure:repo/\n",
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

func (s *ResticDataSuite) TestResticArgsWithAWSRole(c *C) {
	for _, tc := range []struct {
		profile *param.Profile
		output  Checker
	}{
		{
			profile: &param.Profile{
				Location: crv1alpha1.Location{
					Type:     crv1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint", // Also remove all of the trailing slashes
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &corev1.Secret{
						Type: "secrets.kanister.io/aws",
						Data: map[string][]byte{
							secrets.AWSAccessKeyID:     []byte(config.GetEnvOrSkip(c, "AWS_ACCESS_KEY_ID")),
							secrets.AWSSecretAccessKey: []byte(config.GetEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")),
							secrets.ConfigRole:         []byte(config.GetEnvOrSkip(c, "role")),
						},
					},
				},
			},
			output: IsNil,
		},
		{
			profile: &param.Profile{
				Location: crv1alpha1.Location{
					Type:     crv1alpha1.LocationTypeS3Compliant,
					Endpoint: "endpoint", // Also remove all of the trailing slashes
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &corev1.Secret{
						Type: "secrets.kanister.io/aws",
						Data: map[string][]byte{
							secrets.AWSAccessKeyID:     []byte(config.GetEnvOrSkip(c, "AWS_ACCESS_KEY_ID")),
							secrets.AWSSecretAccessKey: []byte(config.GetEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")),
							secrets.ConfigRole:         []byte("arn:aws:iam::000000000000:role/test-fake-role"),
						},
					},
				},
			},
			output: NotNil,
		},
	} {
		_, err := resticArgs(tc.profile, "repo", "my-secret")
		c.Assert(err, tc.output)
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
		// after updating restic to 0.11.0 this format is changed
		{log: "Stats in restore-size mode:", expected: "restore-size"},
		{log: "Stats in restore-size mode:", expected: "restore-size"},
		{log: "Stats in raw-data mode:", expected: "raw-data"},
		{log: "Stats in blobs-per-file mode:", expected: "blobs-per-file"},
		{log: "sudhufehfuijbfjbruifhoiwhf", expected: ""},
	} {
		mode := SnapshotStatsModeFromStatsLog(tc.log)
		c.Assert(mode, Equals, tc.expected)
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
s3:s3.amazonaws.com/abhdbhf/foodbar`, expected: false},
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
s3:s3.amazonaws.com/abhdbhf/foodbar`, expected: true},
	} {
		output := DoesRepoExist(tc.log)
		c.Assert(output, Equals, tc.expected)
	}
}

func (s *ResticDataSuite) TestGetSnapshotStatsFromBackupLog(c *C) {
	for _, tc := range []struct {
		log          string
		expectedfc   string
		expectedsize string
		expectedphy  string
	}{
		{log: "processed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: ""},
		{log: "processed 9 files, 11 KiB in 0:00", expectedfc: "9", expectedsize: "11 KiB", expectedphy: ""},
		{log: "processed 9 files, 11. KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, . KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, .111 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 0.111 KiB in 0:00", expectedfc: "9", expectedsize: "0.111 KiB", expectedphy: ""},
		{log: "processed   9 files, 11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed asdf files, 11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files,  11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "asdf 9 files, 11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9,999,999 files, 11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9.999 files, 11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed  9  files,  11.235 KiB  in  0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed  9  files,  11.235 KiB", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 KiB in", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 KiB in ", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: ""},
		{log: "processed 9 files, 11.235  KiB in ", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 in ", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 , 11.235 KiB in ", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files 11.235 KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 B in 0:00", expectedfc: "9", expectedsize: "11.235 B", expectedphy: ""},
		{log: "processed 9 files, 11.235 MiB in 0:00", expectedfc: "9", expectedsize: "11.235 MiB", expectedphy: ""},
		{log: "processed 9 files, 11.235 GiB in 0:00", expectedfc: "9", expectedsize: "11.235 GiB", expectedphy: ""},
		{log: "processed 9 files, 11.235 TiB in 0:00", expectedfc: "9", expectedsize: "11.235 TiB", expectedphy: ""},
		{log: "processed 9 files, 11.235 PiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 asdf in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 iB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 KB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 MB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 GB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 TB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 PB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 asdfB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed  files, 11.235 asdfB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed files, 11.235 asdfB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files,  KiB in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, in 0:00", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, in 0", expectedfc: "", expectedsize: "", expectedphy: ""},
		{log: "processed 9 files, 11.235 KiB in 0:00\nA", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: ""},
		{log: "processed 9 files, 11.235 KiB in 0:00\nAdded to the repo: 53.771 KiB", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "processed 9 files, 11.235 KiB in 0:00\n\nAdded to the repo: 53.771 KiB", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "processed 9 files, 11.235 KiB in 0:00\nasdfasdf\nAdded to the repo: 53.771 KiB", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "processed 9 files, 11.235 KiB in 0:00\nasdfasdf\nAdded to the repo: 53.771 KiB\nasdf", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "Added to the repo: 53.771 KiB\nprocessed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "Added to the repo: 53.771 KiB\n\nprocessed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "Added to the repo: 53.771 KiB\nasdfasdf\nprocessed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "53.771 KiB"},
		{log: "Added to the repo: 0 B\nprocessed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: "0 B"},
		{log: "Added to the repo: 0 Bogus\nprocessed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: ""},
		{log: "AAdded to the repo: 0 B\nprocessed 9 files, 11.235 KiB in 0:00", expectedfc: "9", expectedsize: "11.235 KiB", expectedphy: ""},
	} {
		c.Log(tc.log)
		fc, s, phy := SnapshotStatsFromBackupLog(tc.log)
		c.Check(fc, Equals, tc.expectedfc)
		c.Check(s, Equals, tc.expectedsize)
		c.Check(phy, Equals, tc.expectedphy)
	}
}

func (s *ResticDataSuite) TestGetSpaceFreedFromPruneLog(c *C) {
	for _, tc := range []struct {
		log                string
		expectedSpaceFreed string
	}{
		{log: "will delete 1 packs and rewrite 1 packs, this frees 11.235 KiB", expectedSpaceFreed: "11.235 KiB"},
		{log: "will delete 1 packs and rewrite 1 packs, this frees 0 KiB", expectedSpaceFreed: "0 KiB"},
		{log: "will delete 1 packs and rewrite 1 packs, this frees 0.0 KiB", expectedSpaceFreed: "0.0 KiB"},
		{log: "will delete 0 packs and rewrite 0 packs, this frees 11.235 KiB", expectedSpaceFreed: "11.235 KiB"},
		{log: "will delete 1 pack and rewrite 1 packs, this frees 11.235 KiB", expectedSpaceFreed: ""},
		{log: "will delete 1 packs and rewrite 1 pack, this frees 11.235 KiB", expectedSpaceFreed: ""},
		{log: "will delete 1 packs and rewrite 1 packs, this frees  KiB", expectedSpaceFreed: ""},
		{log: "will delete 1 packs and rewrite 1 packs, this frees KiB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 KiB", expectedSpaceFreed: "11.235 KiB"},
		{log: "will delete 100 packs and rewrite 100 packs, this frees  11.235 KiB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 KiB ", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235  KiB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 KB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 MB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 GB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 TB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 PiB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 TiB", expectedSpaceFreed: "11.235 TiB"},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 GiB", expectedSpaceFreed: "11.235 GiB"},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 MiB", expectedSpaceFreed: "11.235 MiB"},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 iB", expectedSpaceFreed: ""},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 B", expectedSpaceFreed: "11.235 B"},
		{log: "this frees 11.235 B", expectedSpaceFreed: ""},
		{log: "Some unrelated log\nwill delete 100 packs and rewrite 100 packs, this frees 11.235 B\nSome more unrelated logs\n\n", expectedSpaceFreed: "11.235 B"},
		{log: "Some unrelated log\nwill delete 100 packs and rewrite 100 packs, this frees 11.235 B\n", expectedSpaceFreed: "11.235 B"},
		{log: "Some unrelated log\nwill delete 100 packs and rewrite 100 packs, this frees 11.235 B", expectedSpaceFreed: "11.235 B"},
		{log: "\nwill delete 100 packs and rewrite 100 packs, this frees 11.235 B\nSome more unrelated logs\n\n", expectedSpaceFreed: "11.235 B"},
		{log: "will delete 100 packs and rewrite 100 packs, this frees 11.235 B\nSome more unrelated logs\n\n", expectedSpaceFreed: "11.235 B"},
		{log: "Some unrelated log in the same line, will delete 100 packs and rewrite 100 packs, this frees 11.235 B", expectedSpaceFreed: ""},
	} {
		spaceFreed := SpaceFreedFromPruneLog(tc.log)
		c.Check(spaceFreed, Equals, tc.expectedSpaceFreed)
	}
}

func (s *ResticDataSuite) TestResticSizeStringParser(c *C) {
	for _, tc := range []struct {
		input         string
		expectedSizeB int64
	}{
		{input: "11235 B", expectedSizeB: 11235},
		{input: "11235 KB", expectedSizeB: 0},
		{input: "11235 MB", expectedSizeB: 0},
		{input: "11235 GB", expectedSizeB: 0},
		{input: "11235 TB", expectedSizeB: 0},
		{input: "11235 TiB", expectedSizeB: 11235 * (1 << 40)},
		{input: "11235 GiB", expectedSizeB: 11235 * (1 << 30)},
		{input: "11235 MiB", expectedSizeB: 11235 * (1 << 20)},
		{input: "11235 KiB", expectedSizeB: 11235 * (1 << 10)},
		{input: "", expectedSizeB: 0},
		{input: "asdf", expectedSizeB: 0},
		{input: "123 asdf", expectedSizeB: 0},
		{input: "asdf GiB", expectedSizeB: 0},
		{input: "11235", expectedSizeB: 0},
		{input: "1.1 GiB", expectedSizeB: 1181116006},
		{input: "1.1235 GiB", expectedSizeB: 1206348939},
		{input: " 1.1235 GiB", expectedSizeB: 0},
		{input: "1.1235  GiB", expectedSizeB: 0},
		{input: "1.1235  GiB ", expectedSizeB: 0},
		{input: "1.1235 GiB ", expectedSizeB: 0},
		{input: "1.1235 GiB GiB", expectedSizeB: 0},
		{input: "1.1235 1 GiB", expectedSizeB: 0},
		{input: "-1.1235 GiB", expectedSizeB: 0},
		{input: "GiB 1.1235", expectedSizeB: 0},
	} {
		parsedSize := ParseResticSizeStringBytes(tc.input)
		c.Check(parsedSize, Equals, tc.expectedSizeB)
	}
}
