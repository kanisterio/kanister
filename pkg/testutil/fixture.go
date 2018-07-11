package testutil

import (
	"fmt"
	"os"

	"gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

const testBucketName = "S3_TEST_BUCKET"

var objectStoreTestEnvVars []string = []string{
	location.AWSAccessKeyID,
	location.AWSSecretAccessKey,
	testBucketName,
}

func ObjectStoreProfileOrSkip(c *check.C) *param.Profile {
	skipIfEnvNotSet(c, objectStoreTestEnvVars)
	return &param.Profile{
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
}

func skipIfEnvNotSet(c *check.C, envVars []string) {
	for _, ev := range envVars {
		if os.Getenv(ev) == "" {
			reason := fmt.Sprintf("Test %s requires the environemnt variable '%s'", c.TestName(), ev)
			c.Skip(reason)
		}
	}
}
