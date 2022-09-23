package repository

import (
	"fmt"

	"gopkg.in/check.v1"

	v1 "k8s.io/api/core/v1"
)

func (s *RepositoryUtilsSuite) TestFilesystemArgsUtil(c *check.C) {
	for _, tc := range []struct {
		prefix         string
		artifactPrefix string
		expectedPath   string
	}{
		{
			prefix:         "",
			artifactPrefix: "dir1/subdir/",
			expectedPath:   fmt.Sprintf("%s/dir1/subdir/", DefaultFSMountPath),
		},
		{
			prefix:         "test-prefix",
			artifactPrefix: "dir1/subdir/",
			expectedPath:   fmt.Sprintf("%s/test-prefix/dir1/subdir/", DefaultFSMountPath),
		},
	} {
		sec := &v1.Secret{
			StringData: map[string]string{
				prefixKey: tc.prefix,
			},
		}
		args := filesystemArgs(sec, tc.artifactPrefix)
		expectedValue := fmt.Sprintf("filesystem --path=%s", tc.expectedPath)
		c.Assert(args.String(), check.Equals, expectedValue)
	}
}
