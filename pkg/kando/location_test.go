package kando

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

type LocationSuite struct{}

var _ = Suite(&LocationSuite{})

const testBucketName = "S3_TEST_BUCKET"

var objectStoreTestEnvVars []string = []string{
	location.AWSAccessKeyID,
	location.AWSSecretAccessKey,
	testBucketName,
}

const testContent = "test-content"

func (s *LocationSuite) TestLocationObjectStore(c *C) {
	skipIfEnvNotSet(c, objectStoreTestEnvVars)
	ctx := context.Background()
	p := &param.Profile{
		Location: crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeS3Compliant,
			S3Compliant: &crv1alpha1.S3CompliantLocation{
				Bucket: os.Getenv(testBucketName),
				Prefix: c.TestName(),
			},
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     os.Getenv(location.AWSAccessKeyID),
				Secret: os.Getenv(location.AWSSecretAccessKey),
			},
		},
	}
	path := filepath.Join(c.MkDir(), "test-object.txt")

	source := bytes.NewBufferString(testContent)
	err := locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	target := bytes.NewBuffer(nil)
	err = locationPull(ctx, p, path, target)
	c.Assert(err, IsNil)
	c.Assert(target.String(), Equals, testContent)
}

func skipIfEnvNotSet(c *C, envVars []string) {
	for _, ev := range envVars {
		if os.Getenv(ev) == "" {
			reason := fmt.Sprintf("Test %s requires the environemnt variable '%s'", c.TestName(), ev)
			c.Skip(reason)
		}
	}
}
