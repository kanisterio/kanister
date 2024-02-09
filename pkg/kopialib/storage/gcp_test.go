package storage

import (
	"context"
	"fmt"

	"github.com/kanisterio/kanister/pkg/kopialib"
	"github.com/kopia/kopia/repo/blob/gcs"
	. "gopkg.in/check.v1"
)

type gcpStorageTestSuite struct{}

var _ = Suite(&gcpStorageTestSuite{})

func (s *gcpStorageTestSuite) TestSetOptions(c *C) {
	for i, tc := range []struct {
		name            string
		options         map[string]string
		expectedOptions gcs.Options
		expectedErr     string
		errChecker      Checker
	}{
		{
			name:        "options not set",
			options:     map[string]string{},
			expectedErr: fmt.Sprintf(ErrStorageOptionsCannotBeNilMsg, TypeGCP),
			errChecker:  NotNil,
		},
		{
			name: "bucket name is required",
			options: map[string]string{
				kopialib.PrefixKey: "test-prefix",
			},
			expectedErr: fmt.Sprintf(ErrMissingRequiredFieldMsg, kopialib.BucketKey, TypeGCP),
			errChecker:  NotNil,
		},
		{
			name: "set correct options",
			options: map[string]string{
				kopialib.BucketKey:                        "test-bucket",
				kopialib.PrefixKey:                        "test-prefix",
				kopialib.GCPServiceAccountCredentialsFile: "credentials-file",
			},
			expectedOptions: gcs.Options{
				BucketName:                    "test-bucket",
				Prefix:                        "test-prefix",
				ServiceAccountCredentialsFile: "credentials-file",
			},
			errChecker: IsNil,
		},
	} {
		gcpStorage := gcpStorage{}
		err := gcpStorage.SetOptions(context.Background(), tc.options)
		c.Check(err, tc.errChecker)
		if err != nil {
			c.Check(err.Error(), Equals, tc.expectedErr, Commentf("test number: %d", i))
		}

		c.Assert(gcpStorage.Options, DeepEquals, tc.expectedOptions)
	}
}
