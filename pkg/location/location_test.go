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

package location

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type LocationSuite struct {
	osType            objectstore.ProviderType
	provider          objectstore.Provider
	rand              *rand.Rand
	root              objectstore.Bucket // root of the default test bucket
	suiteDirPrefix    string             // directory name prefix for all tests in this suite
	testpath          string
	testMultipartPath string
	region            string // bucket region
	profile           param.Profile
}

const (
	testBucketName = "kio-store-tests"
	testRegionS3   = "us-west-2"
)

var _ = Suite(&LocationSuite{osType: objectstore.ProviderTypeS3, region: testRegionS3})
var _ = Suite(&LocationSuite{osType: objectstore.ProviderTypeGCS, region: ""})
var _ = Suite(&LocationSuite{osType: objectstore.ProviderTypeAzure, region: ""})

func (s *LocationSuite) SetUpSuite(c *C) {
	var location crv1alpha1.Location
	switch s.osType {
	case objectstore.ProviderTypeS3:
		testutil.GetEnvOrSkip(c, AWSAccessKeyID)
		testutil.GetEnvOrSkip(c, AWSSecretAccessKey)
		location = crv1alpha1.Location{
			Type:   crv1alpha1.LocationTypeS3Compliant,
			Region: s.region,
		}
	case objectstore.ProviderTypeGCS:
		testutil.GetEnvOrSkip(c, GoogleCloudCreds)
		location = crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeGCS,
		}
	case objectstore.ProviderTypeAzure:
		testutil.GetEnvOrSkip(c, blockstorage.AzureStorageAccount)
		testutil.GetEnvOrSkip(c, blockstorage.AzureStorageKey)
		location = crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeAzure,
		}
	default:
		c.Fatalf("Unrecognized objectstore '%s'", s.osType)
	}
	location.Bucket = testBucketName
	s.profile = *testutil.ObjectStoreProfileOrSkip(c, s.osType, location)
	var err error
	ctx := context.Background()

	s.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	pc := objectstore.ProviderConfig{
		Type:   s.osType,
		Region: s.region,
	}
	secret, err := getOSSecret(ctx, s.osType, s.profile.Credential)
	c.Check(err, IsNil)
	s.provider, err = objectstore.NewProvider(ctx, pc, secret)
	c.Check(err, IsNil)
	c.Assert(s.provider, NotNil)

	s.root, err = objectstore.GetOrCreateBucket(ctx, s.provider, testBucketName)
	c.Check(err, IsNil)
	c.Assert(s.root, NotNil)
	s.suiteDirPrefix = time.Now().UTC().Format(time.RFC3339Nano)
	s.testpath = s.suiteDirPrefix + "/testlocation.txt"
	s.testMultipartPath = s.suiteDirPrefix + "/testchunk.txt"
}

func (s *LocationSuite) TearDownTest(c *C) {
	if s.testpath != "" {
		c.Assert(s.root, NotNil)
		ctx := context.Background()
		err := s.root.Delete(ctx, s.testpath)
		if err != nil {
			c.Log("Cannot cleanup test directory: ", s.testpath)
		}
	}
	if s.testMultipartPath != "" {
		c.Assert(s.root, NotNil)
		ctx := context.Background()
		err := s.root.Delete(ctx, s.testMultipartPath)
		if err != nil {
			c.Log("Cannot cleanup test directory: ", s.testMultipartPath)
		}
	}
}

func (s *LocationSuite) TestWriteAndReadData(c *C) {
	ctx := context.Background()
	teststring := "test-content-check"
	err := writeData(ctx, s.osType, s.profile, bytes.NewBufferString(teststring), s.testpath)
	c.Check(err, IsNil)
	buf := bytes.NewBuffer(nil)
	err = readData(ctx, s.osType, s.profile, buf, s.testpath)
	c.Check(err, IsNil)
	c.Check(buf.String(), Equals, teststring)
}

func (s *LocationSuite) TestAzMultipartUpload(c *C) {
	if s.osType != objectstore.ProviderTypeAzure {
		c.Skip(fmt.Sprintf("Not applicable for location type %s", s.osType))
	}

	// Create dir if not exists
	_, err := os.Stat(s.suiteDirPrefix)
	if os.IsNotExist(err) {
		err := os.MkdirAll(s.suiteDirPrefix, 0755)
		c.Check(err, IsNil)
	}
	// Create test file
	f, err := os.Create(s.testMultipartPath)
	c.Check(err, IsNil)
	defer func() {
		err = f.Close()
		c.Assert(err, IsNil)
	}()
	ctx := context.Background()
	for _, fileSize := range []int64{
		0,                 // empty file
		100 * 1024 * 1024, // 100M ie < buffSize
		buffSize - 1,
		buffSize,
		buffSize + 1,
		300 * 1024 * 1024, // 300M ie > buffSize
	} {
		_, err := f.Seek(0, io.SeekStart)
		c.Assert(err, IsNil)

		// Create dump file
		err = os.Truncate(s.testMultipartPath, fileSize)
		c.Assert(err, IsNil)
		err = writeData(ctx, s.osType, s.profile, f, s.testMultipartPath)
		c.Check(err, IsNil)
		buf := bytes.NewBuffer(nil)
		err = readData(ctx, s.osType, s.profile, buf, s.testMultipartPath)
		c.Check(err, IsNil)
		c.Check(int64(buf.Len()), Equals, fileSize)
	}
}

func (s *LocationSuite) TestReaderSize(c *C) {
	for _, tc := range []struct {
		input        string
		buffSize     int64
		expectedSize int64
	}{
		{
			input:        "dummy-string-1",
			buffSize:     4,
			expectedSize: 1073741824, // buffSizeLimit       = 1 * 1024 * 1024 * 1024
		},
		{
			input:        "dummy-string-1",
			buffSize:     14,
			expectedSize: 1073741824,
		},
		{
			input:        "dummy-string-1",
			buffSize:     44,
			expectedSize: 0,
		},
		{
			input:        "",
			buffSize:     4,
			expectedSize: 0,
		},
	} {
		_, size, err := readerSize(bytes.NewBufferString(tc.input), tc.buffSize)
		c.Assert(err, IsNil)
		c.Assert(size, Equals, tc.expectedSize)
	}
}

func (s *LocationSuite) TestGetAzureSecret(c *C) {
	for _, tc := range []struct {
		cred        param.Credential
		retAzSecret *objectstore.SecretAzure
		errChecker  Checker
	}{
		{
			cred: param.Credential{
				Type: param.CredentialTypeKeyPair,
				KeyPair: &param.KeyPair{
					ID:     "id",
					Secret: "secret",
				},
				Secret: &corev1.Secret{
					Type: corev1.SecretType(secrets.AzureSecretType),
					Data: map[string][]byte{
						secrets.AzureStorageAccountID:   []byte("said"),
						secrets.AzureStorageAccountKey:  []byte("sakey"),
						secrets.AzureStorageEnvironment: []byte("env"),
					},
				},
			},
			retAzSecret: &objectstore.SecretAzure{
				StorageAccount: "id",
				StorageKey:     "secret",
			},
			errChecker: IsNil,
		},
		{
			cred: param.Credential{
				Type: param.CredentialTypeSecret,
				KeyPair: &param.KeyPair{
					ID:     "id",
					Secret: "secret",
				},
				Secret: &corev1.Secret{
					Type: corev1.SecretType(secrets.AzureSecretType),
					Data: map[string][]byte{
						secrets.AzureStorageAccountID:   []byte("said"),
						secrets.AzureStorageAccountKey:  []byte("sakey"),
						secrets.AzureStorageEnvironment: []byte("env"),
					},
				},
			},
			retAzSecret: &objectstore.SecretAzure{
				StorageAccount:  "said",
				StorageKey:      "sakey",
				EnvironmentName: "env",
			},
			errChecker: IsNil,
		},
		{ // missing required field
			cred: param.Credential{
				Type: param.CredentialTypeSecret,
				KeyPair: &param.KeyPair{
					ID:     "id",
					Secret: "secret",
				},
				Secret: &corev1.Secret{
					Type: corev1.SecretType(secrets.AzureSecretType),
					Data: map[string][]byte{
						secrets.AzureStorageAccountID:   []byte("said"),
						secrets.AzureStorageEnvironment: []byte("env"),
					},
				},
			},
			retAzSecret: &objectstore.SecretAzure{
				StorageAccount:  "said",
				StorageKey:      "sakey",
				EnvironmentName: "env",
			},
			errChecker: NotNil,
		},
		{ // additional incorrect field
			cred: param.Credential{
				Type: param.CredentialTypeSecret,
				KeyPair: &param.KeyPair{
					ID:     "id",
					Secret: "secret",
				},
				Secret: &corev1.Secret{
					Type: corev1.SecretType(secrets.AzureSecretType),
					Data: map[string][]byte{
						secrets.AzureStorageAccountID:   []byte("said"),
						secrets.AzureStorageAccountKey:  []byte("sakey"),
						"extrafield":                    []byte("extra"),
						secrets.AzureStorageEnvironment: []byte("env"),
					},
				},
			},
			retAzSecret: &objectstore.SecretAzure{
				StorageAccount:  "said",
				StorageKey:      "sakey",
				EnvironmentName: "env",
			},
			errChecker: NotNil,
		},
	} {
		secret, err := getAzureSecret(tc.cred)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(secret.Azure.StorageKey, Equals, tc.retAzSecret.StorageKey)
			c.Assert(secret.Azure.StorageAccount, Equals, tc.retAzSecret.StorageAccount)
			c.Assert(secret.Azure.EnvironmentName, Equals, tc.retAzSecret.EnvironmentName)
		}
	}
}
