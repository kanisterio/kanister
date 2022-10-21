package storage

import (
	"fmt"

	"gopkg.in/check.v1"
)

func (s *StorageUtilsSuite) TestGCSArgsUtil(c *check.C) {
	locSecret := map[string]string{
		prefixKey: "test-prefix",
		bucketKey: "test-bucket",
	}
	artifactPrefix := "dir/sub-dir"
	cmd := kopiaGCSArgs(locSecret, artifactPrefix)
	c.Assert(cmd.String(), check.Equals, fmt.Sprint("gcs --bucket=",
		locSecret[bucketKey],
		" --credentials-file=/tmp/creds.txt",
		" --prefix=", fmt.Sprintf("%s/%s/", locSecret[prefixKey], artifactPrefix),
	))
}
