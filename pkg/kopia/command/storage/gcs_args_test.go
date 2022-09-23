package storage

import (
	"fmt"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
)

func (s *RepositoryUtilsSuite) TestGCSArgsUtil(c *check.C) {
	locSecret := &v1.Secret{
		StringData: map[string]string{
			prefixKey: "test-prefix",
			bucketKey: "test-bucket",
		},
	}
	artifactPrefix := "dir/sub-dir"
	cmd := kopiaGCSArgs(locSecret, artifactPrefix)
	c.Assert(cmd.String(), check.Equals, fmt.Sprint("gcs --bucket=",
		locSecret.StringData[bucketKey],
		" --credentials-file=/tmp/creds.txt",
		" --prefix=", fmt.Sprintf("%s/%s/", locSecret.StringData[prefixKey], artifactPrefix),
	))
}
