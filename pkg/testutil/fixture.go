package testutil

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
)

const TestS3BucketName = "S3_TEST_BUCKET"

func ObjectStoreProfileOrSkip(c *check.C, osType objectstore.ProviderType, location crv1alpha1.Location) *param.Profile {
	var key, val string

	switch osType {
	case objectstore.ProviderTypeS3:
		key = GetEnvOrSkip(c, awsebs.AccessKeyID)
		val = GetEnvOrSkip(c, awsebs.SecretAccessKey)
	case objectstore.ProviderTypeGCS:
		GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)
		creds, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope)
		c.Check(err, check.IsNil)
		key = creds.ProjectID
		val = string(creds.JSON)
	}
	return &param.Profile{
		Location: location,
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     key,
				Secret: val,
			},
		},
	}
}

func GetEnvOrSkip(c *check.C, varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		reason := fmt.Sprintf("Test %s requires the environemnt variable '%s'", c.TestName(), varName)
		c.Skip(reason)
	}
	return v
}
