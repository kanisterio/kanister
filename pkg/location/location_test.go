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
	"math/rand"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type LocationSuite struct {
	osType         objectstore.ProviderType
	provider       objectstore.Provider
	rand           *rand.Rand
	root           objectstore.Bucket // root of the default test bucket
	suiteDirPrefix string             // directory name prefix for all tests in this suite
	testpath       string
	region         string // bucket region
	profile        param.Profile
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
}

func (s *LocationSuite) TearDownTest(c *C) {
	if s.testpath != "" {
		c.Assert(s.root, NotNil)
		ctx := context.Background()
		err := s.root.Delete(ctx, s.testpath)
		if err != nil {
			c.Log("Cannot cleanup test directory: ", s.testpath)
			return
		}
	}
}

func (s *LocationSuite) TestWriteAndReadData(c *C) {
	ctx := context.Background()
	teststring := "test-content"
	err := writeData(ctx, s.osType, s.profile, bytes.NewBufferString(teststring), s.testpath)
	c.Check(err, IsNil)
	buf := bytes.NewBuffer(nil)
	err = readData(ctx, s.osType, s.profile, buf, s.testpath)
	c.Check(err, IsNil)
	c.Check(buf.String(), Equals, teststring)
}
