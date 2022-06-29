package kopia

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"

	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/snapshot"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

type KopiaTestSuite struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&KopiaTestSuite{})

func (s *KopiaTestSuite) TestKopiaCommandLogging(c *check.C) {
	testProfile := &KanisterProfile{
		Profile: &param.Profile{
			Location: v1alpha1.Location{
				Type:     v1alpha1.LocationTypeS3Compliant,
				Endpoint: "endpoint/", // Remove trailing slash
				Bucket:   "my-bucket",
			},
			Credential: param.Credential{
				Type: param.CredentialTypeSecret,
				Secret: &v1.Secret{
					Type: v1.SecretType(secrets.AWSSecretType),
					Data: map[string][]byte{
						secrets.AWSAccessKeyID:     []byte("id"),
						secrets.AWSSecretAccessKey: []byte("secret"),
					},
				},
			},
			SkipSSLVerify: false,
		},
	}

	for _, tc := range []struct {
		f           func() logsafe.Cmd
		expectedLog string
	}{
		{
			f: func() logsafe.Cmd {
				cmd, err := repositoryCreateCommand(testProfile, "artifact-prefix", "encr-key", "a-hostname", "a-username", "cache/path", "path/kopia.config", "cache/log", 11235, 813)
				c.Assert(err, check.IsNil)
				return cmd
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> repository create --no-check-for-updates --cache-directory=cache/path --content-cache-size-mb=11235 --metadata-cache-size-mb=813 --override-hostname=a-hostname --override-username=a-username s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=artifact-prefix",
		},
		{
			f: func() logsafe.Cmd {
				cmd, err := repositoryCreateCommand(testProfile, "artifact-prefix", "encr-key", "a-hostname", "a-username", "cache/path", "", "", 11235, 813)
				c.Assert(err, check.IsNil)
				return cmd
			},
			expectedLog: "kopia --log-level=error --password=<****> repository create --no-check-for-updates --cache-directory=cache/path --content-cache-size-mb=11235 --metadata-cache-size-mb=813 --override-hostname=a-hostname --override-username=a-username s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=artifact-prefix",
		},
		{
			f: func() logsafe.Cmd {
				cmd, err := repositoryConnectCommand(testProfile, "artifact-prefix", "encr-key", "a-hostname", "a-username", "cache/path", "path/kopia.config", "cache/log", 11235, 813, strfmt.DateTime{})
				c.Assert(err, check.IsNil)
				return cmd
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> repository connect --no-check-for-updates --cache-directory=cache/path --content-cache-size-mb=11235 --metadata-cache-size-mb=813 --override-hostname=a-hostname --override-username=a-username s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=artifact-prefix",
		},
		{
			f: func() logsafe.Cmd {
				cmd, err := repositoryConnectCommand(testProfile, "artifact-prefix", "encr-key", "a-hostname", "a-username", "cache/path", "path/kopia.config", "cache/log", 11235, 813, strfmt.DateTime(time.Time{}.Add(1)))
				c.Assert(err, check.IsNil)
				return cmd
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> repository connect --no-check-for-updates --cache-directory=cache/path --content-cache-size-mb=11235 --metadata-cache-size-mb=813 --override-hostname=a-hostname --override-username=a-username s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=artifact-prefix --point-in-time=0001-01-01T00:00:00.000Z",
		},
		{
			f: func() logsafe.Cmd {
				return snapshotCreateCommand("encr-key", "path/to/backup", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=info --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot create path/to/backup --json --parallel=8 --progress-update-interval=1h",
		},
		{
			f: func() logsafe.Cmd {
				return snapshotExpireCommand("encr-key", "root-id", "path/kopia.config", "cache/log", true)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot expire root-id --delete",
		},
		{
			f: func() logsafe.Cmd {
				return snapshotRestoreCommand("encr-key", "snapshot-id", "target/path", "path/kopia.config", "cache/log", false /* sparse */)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot restore snapshot-id target/path",
		},
		{
			f: func() logsafe.Cmd {
				return snapshotRestoreCommand("encr-key", "snapshot-id", "target/path", "path/kopia.config", "cache/log", true /* sparse */)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot restore snapshot-id target/path --sparse",
		},
		{
			f: func() logsafe.Cmd {
				return restoreCommand("encr-key", "snapshot-id", "target/path", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> restore snapshot-id target/path",
		},
		{
			f: func() logsafe.Cmd {
				return deleteCommand("encr-key", "snapshot-id", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot delete snapshot-id --unsafe-ignore-source",
		},
		{
			f: func() logsafe.Cmd {
				return snapshotGCCommand("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot gc --delete",
		},
		{
			f: func() logsafe.Cmd {
				return maintenanceSetOwner("encr-key", "path/kopia.config", "cache/log", "username@hostname")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance set --owner=username@hostname",
		},
		{
			f: func() logsafe.Cmd {
				return maintenanceRunCommand("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance run",
		},
		{
			f: func() logsafe.Cmd {
				return maintenanceInfoCommand("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance info",
		},
		{
			f: func() logsafe.Cmd {
				return blobList("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> blob list",
		},
		{
			f: func() logsafe.Cmd {
				return blobStats("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> blob stats --raw",
		},
		{
			f: func() logsafe.Cmd {
				return snapListAllWithSnapIDs("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> manifest list --json --filter=type:snapshot",
		},
		{
			f: func() logsafe.Cmd {
				return policySetGlobalCommandSetup("encr-key", "path/kopia.config", "cache/log", policyChanges{"asdf": "bsdf"})
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> policy set --global asdf=bsdf",
		},
		{
			f: func() logsafe.Cmd {
				return repositoryConnectServerCommand("cache/path", "path/kopia.config", "a-hostname", "cache/log", "a-url", "a-fingerprint", "a-username", "encr-key", 11235, 813)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> repository connect server --no-check-for-updates --no-grpc --cache-directory=cache/path --content-cache-size-mb=11235 --metadata-cache-size-mb=813 --override-hostname=a-hostname --override-username=a-username --url=a-url --server-cert-fingerprint=<****>",
		},
		{
			f: func() logsafe.Cmd {
				return serverStartCommand("path/kopia.config", "cache/log", "a-server-address", "/path/to/cert/tls.crt", "/path/to/key/tls.key", "a-username@a-hostname", "a-user-password", true, true)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=<****> --server-control-username=a-username@a-hostname --server-control-password=<****> --no-grpc > /dev/null 2>&1 &",
		},
		{
			f: func() logsafe.Cmd {
				return serverStartCommand("path/kopia.config", "cache/log", "a-server-address", "/path/to/cert/tls.crt", "/path/to/key/tls.key", "a-username@a-hostname", "a-user-password", true, false)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --tls-generate-cert --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=<****> --server-control-username=a-username@a-hostname --server-control-password=<****> --no-grpc",
		},
		{
			f: func() logsafe.Cmd {
				return serverStartCommand("path/kopia.config", "cache/log", "a-server-address", "/path/to/cert/tls.crt", "/path/to/key/tls.key", "a-username@a-hostname", "a-user-password", false, true)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server start --address=a-server-address --tls-cert-file=/path/to/cert/tls.crt --tls-key-file=/path/to/key/tls.key --server-username=a-username@a-hostname --server-password=<****> --server-control-username=a-username@a-hostname --server-control-password=<****> --no-grpc > /dev/null 2>&1 &",
		},
		{
			f: func() logsafe.Cmd {
				return serverStatusCommand("path/kopia.config", "cache/log", "a-server-address", "a-username@a-hostname", "a-user-password", "a-fingerprint")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log server status --address=a-server-address --server-cert-fingerprint=<****> --server-username=a-username@a-hostname --server-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				return serverAddUserCommand("encr-key", "path/kopia.config", "cache/log", "a-username@a-hostname", "a-user-password")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server user add a-username@a-hostname --user-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				return serverSetUserCommand("encr-key", "path/kopia.config", "cache/log", "a-username@a-hostname", "a-user-password")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server user set a-username@a-hostname --user-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				return serverListUserCommand("encr-key", "path/kopia.config", "cache/log")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server user list --json",
		},
		{
			f: func() logsafe.Cmd {
				return serverRefreshCommand("encr-key", "path/kopia.config", "cache/log", "a-server-address", "a-username@a-hostname", "a-user-password", "a-fingerprint")
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> server refresh --server-cert-fingerprint=<****> --address=a-server-address --server-username=a-username@a-hostname --server-password=<****>",
		},
		{
			f: func() logsafe.Cmd {
				tmp := *testProfile
				{
					tmpProfile := *testProfile.Profile
					tmpProfile.Location.Region = "my-region"
					tmp.Profile = &tmpProfile
				}
				cmd, err := repositoryCreateCommand(&tmp, "artifact-prefix", "encr-key", "a-hostname", "a-username", "cache/path", "path/kopia.config", "cache/log", 11235, 813)
				c.Assert(err, check.IsNil)
				return cmd
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> repository create --no-check-for-updates --cache-directory=cache/path --content-cache-size-mb=11235 --metadata-cache-size-mb=813 --override-hostname=a-hostname --override-username=a-username s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=artifact-prefix --region=my-region",
		},
	} {
		cmd := tc.f()
		c.Check(cmd.String(), check.Equals, tc.expectedLog)
	}
}

func (s *KopiaTestSuite) TestPhysicalSizeFromBlobStatsRaw(c *check.C) {
	for _, tc := range []struct {
		blobStatsOutput string
		expSizeVal      int64
		expCount        int
		errChecker      check.Checker
	}{
		{
			"Count: 813\nTotal: 11235\n",
			11235,
			813,
			check.IsNil,
		},
		{
			"Total: 11235\nCount: 813\n",
			11235,
			813,
			check.IsNil,
		},
		{
			"Count: 0\nTotal: 0\n",
			0,
			0,
			check.IsNil,
		},
		{
			"Count: 5\nTotal: 0.0\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Count: 5\nTotal: asdf\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Count: 5\nTotal: 11235,\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Total: -11235\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Total: 11235",
			0,
			0,
			check.NotNil,
		},
		{
			"Count: 11235",
			0,
			0,
			check.NotNil,
		},
		{
			"Other-field: 11235",
			0,
			0,
			check.NotNil,
		},
		{
			"random input that doesn't comply with expected format",
			0,
			0,
			check.NotNil,
		},
		{
			`
Count: 26
Total: 65628
Average: 2524
Histogram:

		0 between 0 and 10 (total 0)
		0 between 10 and 100 (total 0)
		11 between 100 and 1000 (total 2132)
		15 between 1000 and 10000 (total 63496)
		0 between 10000 and 100000 (total 0)
		0 between 100000 and 1000000 (total 0)
		0 between 1000000 and 10000000 (total 0)
		0 between 10000000 and 100000000 (total 0)`,
			65628,
			26,
			check.IsNil,
		},
	} {
		gotSize, gotCount, err := RepoSizeStatsFromBlobStatsRaw(tc.blobStatsOutput)
		c.Check(err, tc.errChecker, check.Commentf("Failed for log: %s", tc.blobStatsOutput))
		c.Check(gotSize, check.Equals, tc.expSizeVal)
		c.Check(gotCount, check.Equals, tc.expCount)
	}
}

func (s *KopiaTestSuite) TestSnapSizeStatsFromSnapListAll(c *check.C) {
	for ti, tc := range []struct {
		description     string
		outputGenFunc   func(*check.C, []*snapshot.Manifest) string
		expManifestList []*snapshot.Manifest
		expCount        int
		expSize         int64
		errChecker      check.Checker
	}{
		{
			description:     "empty manifest list",
			outputGenFunc:   marshalManifestList,
			expManifestList: []*snapshot.Manifest{},
			expCount:        0,
			expSize:         0,
			errChecker:      check.IsNil,
		},
		{
			description:   "basic manifest list",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 1,
						},
					},
				},
			},
			expCount:   1,
			expSize:    1,
			errChecker: check.IsNil,
		},
		{
			description:   "manifest list with multiple snapshots",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 1,
						},
					},
				},
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 10,
						},
					},
				},
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 100,
						},
					},
				},
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 1000,
						},
					},
				},
			},
			expCount:   4,
			expSize:    1111,
			errChecker: check.IsNil,
		},
		{
			description:   "error: snapshot with no directory summary, size is treated as zero",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{
					RootEntry: &snapshot.DirEntry{},
				},
			},
			expCount:   1,
			expSize:    0,
			errChecker: check.IsNil,
		},
		{
			description:   "error: snapshot with no root entry, size is treated as zero",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{},
			},
			expCount:   1,
			expSize:    0,
			errChecker: check.IsNil,
		},
		{
			description: "error: parse empty output",
			outputGenFunc: func(c *check.C, manifestList []*snapshot.Manifest) string {
				return ""
			},
			expCount:   0,
			expSize:    0,
			errChecker: check.NotNil,
		},
		{
			description: "error: unmarshal fails",
			outputGenFunc: func(c *check.C, manifestList []*snapshot.Manifest) string {
				return "asdf"
			},
			expCount:   0,
			expSize:    0,
			errChecker: check.NotNil,
		},
	} {
		c.Logf("%d: %s", ti, tc.description)

		outputToParse := tc.outputGenFunc(c, tc.expManifestList)

		gotTotSizeB, gotNumSnapshots, err := SnapSizeStatsFromSnapListAll(outputToParse)
		c.Check(err, tc.errChecker, check.Commentf("Failed for output: %q", outputToParse))
		c.Check(gotTotSizeB, check.Equals, tc.expSize)
		c.Check(gotNumSnapshots, check.Equals, tc.expCount)
		c.Log(err)
	}
}

func marshalManifestList(c *check.C, manifestList []*snapshot.Manifest) string {
	c.Assert(manifestList, check.NotNil)

	b, err := json.Marshal(manifestList)
	c.Assert(err, check.IsNil)

	return string(b)
}

// TestKopiaPolicySetGlobalCommand
// Motivation: very basic test for populating the kopia policy set command fields
// Description:
//   - Feed different combinations of RetentionChanges and compression algorithm changes
//      into PolicySetGlobalCommand, the function that constructs the policy set command.
//   - Check that the command has the requested fields, does not have any unrequested
//     fields, and the field values that appear match as requested.
func (s *KopiaTestSuite) TestKopiaPolicySetGlobalCommand(c *check.C) {
	const maxInt32 = 1<<31 - 1
	for _, tc := range []struct {
		rc policyChanges
	}{
		{rc: policyChanges{
			keepLatest: strconv.Itoa(maxInt32),
		}},
		{rc: policyChanges{
			keepLatest:  strconv.Itoa(rand.Intn(maxInt32)),
			keepHourly:  strconv.Itoa(rand.Intn(maxInt32)),
			keepDaily:   strconv.Itoa(rand.Intn(maxInt32)),
			keepWeekly:  strconv.Itoa(rand.Intn(maxInt32)),
			keepMonthly: strconv.Itoa(rand.Intn(maxInt32)),
			keepAnnual:  strconv.Itoa(rand.Intn(maxInt32)),
		}},
		{rc: policyChanges{}},
		{rc: policyChanges{
			keepLatest:  strconv.Itoa(0),
			keepHourly:  strconv.Itoa(0),
			keepDaily:   strconv.Itoa(0),
			keepWeekly:  strconv.Itoa(0),
			keepMonthly: strconv.Itoa(0),
			keepAnnual:  strconv.Itoa(0),
		}},
		{rc: policyChanges{
			compressionAlgorithm: "compr-algo",
		}},
		{rc: policyChanges{
			compressionAlgorithm: s2DefaultComprAlgo,
		}},
		{rc: policyChanges{
			compressionAlgorithm: s2DefaultComprAlgo,
			keepLatest:           strconv.Itoa(0),
			keepHourly:           strconv.Itoa(0),
			keepDaily:            strconv.Itoa(0),
			keepWeekly:           strconv.Itoa(0),
			keepMonthly:          strconv.Itoa(0),
			keepAnnual:           strconv.Itoa(0),
		}},
	} {
		encryptionKey := "asdf"
		kopiaCmd := policySetGlobalCommand(encryptionKey, "path/kopia.config", "cache/log", tc.rc)

		fieldsFound := make(map[string]bool)
		for i, field := range kopiaCmd {
			switch {
			case hasKnownFlag(field):
				// Executed only for policy set command flags
				// Finds the flag with its value of form `flag=value`
				// Extracts and checks if the correct value is set
				c.Assert(i < len(kopiaCmd), check.Equals, true)
				flagEqVal := kopiaCmd[i]
				args := strings.Split(flagEqVal, "=")
				c.Check(len(args) > 0, check.Equals, true)
				key := args[0]
				val := args[1]
				c.Check(val, check.Equals, tc.rc[key])
				_, ok := fieldsFound[field]
				// Expect no duplicate fields
				c.Check(ok, check.Equals, false)
				fieldsFound[field] = true
			}
		}

		// Check all changing fields were found in the command
		c.Check(len(fieldsFound), check.Equals, len(tc.rc))
	}
}

func hasKnownFlag(field string) bool {
	return strings.Contains(field, keepLatest) ||
		strings.Contains(field, keepHourly) ||
		strings.Contains(field, keepDaily) ||
		strings.Contains(field, keepWeekly) ||
		strings.Contains(field, keepMonthly) ||
		strings.Contains(field, keepAnnual) ||
		strings.Contains(field, compressionAlgorithm)
}

func (s *KopiaTestSuite) TestSnapshotStatsFromSnapshotCreate(c *check.C) {
	type args struct {
		snapCreateOutput string
	}
	tests := []struct {
		name      string
		args      args
		wantStats *SnapshotCreateStats
	}{
		{
			name: "Basic test case",
			args: args{
				snapCreateOutput: " * 0 hashing, 1 hashed (2 B), 3 cached (40 KB), uploaded 6.7 GB, estimated 2044.2 MB (100.0%) 0s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:   1,
				SizeHashedB:   2,
				FilesCached:   3,
				SizeCachedB:   40000,
				SizeUploadedB: 6700000000,
			},
		},
		{
			name: "Real test case",
			args: args{
				snapCreateOutput: " * 0 hashing, 283 hashed (219.5 MB), 0 cached (0 B), uploaded 10.5 MB, estimated 6.01 MB (100.0%) 0s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:   283,
				SizeHashedB:   219500000,
				FilesCached:   0,
				SizeCachedB:   0,
				SizeUploadedB: 10500000,
			},
		},
		{
			name: "Check multiple digits each field",
			args: args{
				snapCreateOutput: " * 0 hashing, 123 hashed (1234.5 MB), 123 cached (1234 B), uploaded 1234.5 KB, estimated 941.2 KB (100.0%) 0s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:   123,
				SizeHashedB:   1234500000,
				FilesCached:   123,
				SizeCachedB:   1234,
				SizeUploadedB: 1234500,
			},
		},
	}
	for _, tt := range tests {
		c.Log(tt.name)
		if gotStats := SnapshotStatsFromSnapshotCreate(tt.args.snapCreateOutput); !reflect.DeepEqual(gotStats, tt.wantStats) {
			c.Errorf("SnapshotStatsFromSnapshotCreate() = %v, want %v", gotStats, tt.wantStats)
		}
	}
}
