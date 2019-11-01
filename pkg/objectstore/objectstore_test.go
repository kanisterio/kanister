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

package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/graymeta/stow"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/config/aws"
)

func Test(t *testing.T) { TestingT(t) }

type ObjectStoreProviderSuite struct {
	osType         ProviderType
	provider       Provider
	rand           *rand.Rand
	root           Bucket // root of the default test bucket
	suiteDirPrefix string // directory name prefix for all tests in this suite
	testDir        string // directory name for a given test
	buckets        []string
	region         string // bucket region
}

const (
	testBucketName = "kio-store-tests"
	testRegionS3   = "us-west-2"
)

var _ = Suite(&ObjectStoreProviderSuite{osType: ProviderTypeS3, region: testRegionS3})
var _ = Suite(&ObjectStoreProviderSuite{osType: ProviderTypeGCS, region: ""})
var _ = Suite(&ObjectStoreProviderSuite{osType: ProviderTypeAzure, region: ""})

func (s *ObjectStoreProviderSuite) SetUpSuite(c *C) {
	switch s.osType {
	case ProviderTypeS3:
		getEnvOrSkip(c, "AWS_ACCESS_KEY_ID")
		getEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")
	case ProviderTypeGCS:
		// Google performs other checks as well..
		getEnvOrSkip(c, "GOOGLE_APPLICATION_CREDENTIALS")
	case ProviderTypeAzure:
		getEnvOrSkip(c, "AZURE_STORAGE_ACCOUNT")
		getEnvOrSkip(c, "AZURE_STORAGE_KEY")
	default:
		c.Fatalf("Unrecognized objectstore '%s'", s.osType)
	}
	var err error
	ctx := context.Background()

	s.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	pc := ProviderConfig{Type: s.osType}
	secret := getSecret(c, ctx, s.osType)
	s.provider, err = NewProvider(ctx, pc, secret)
	c.Check(err, IsNil)
	c.Assert(s.provider, NotNil)

	s.root, err = GetOrCreateBucket(ctx, s.provider, testBucketName, s.region)
	c.Check(err, IsNil)
	c.Assert(s.root, NotNil)
	// While two concurrent instances could potentially collide, the probability
	// is extremely low. This approach makes the directory prefix informative.
	s.suiteDirPrefix = time.Now().UTC().Format(time.RFC3339Nano)
}

func (s *ObjectStoreProviderSuite) SetUpTest(c *C) {
	s.testDir = s.suiteDirPrefix + "-" + c.TestName()
}

func (s *ObjectStoreProviderSuite) TearDownTest(c *C) {
	if s.testDir != "" {
		cleanupBucketDirectory(c, s.root, s.testDir)
	}
}

// Verifies bucket operations, create/delete/list
func (s *ObjectStoreProviderSuite) TestBuckets(c *C) {
	ctx := context.Background()
	c.Skip("intermittently fails due to rate limits on bucket creation")
	bucketName := s.createBucketName(c)

	origBuckets, err := s.provider.ListBuckets(ctx)

	_, err = s.provider.CreateBucket(ctx, bucketName, s.region)
	c.Assert(err, IsNil)

	// Duplicate bucket
	_, err = s.provider.CreateBucket(ctx, bucketName, s.region)
	c.Assert(err, Not(IsNil))

	// Should be one more than buckets. Can be racy with other activity
	// and so checking for inequality
	buckets, err := s.provider.ListBuckets(ctx)
	c.Check(len(buckets), Not(Equals), len(origBuckets))

	bucket, err := s.provider.GetBucket(ctx, bucketName)
	c.Assert(err, IsNil)
	c.Logf("Created bucket: %s", bucket)
	c.Check(len(buckets), Not(Equals), len(origBuckets))

	// Check if deletion succeeds
	err = s.provider.DeleteBucket(ctx, bucketName)
	c.Check(err, IsNil)
}

func (s *ObjectStoreProviderSuite) TestCreateExistingBucket(c *C) {
	ctx := context.Background()
	// The bucket should already exist, the suite setup creates it
	d, err := s.provider.GetBucket(ctx, testBucketName)
	c.Check(err, IsNil)
	c.Check(d, NotNil)
	d, err = s.provider.CreateBucket(ctx, testBucketName, s.region)
	c.Check(err, NotNil)
	c.Check(d, IsNil)
}

func (s *ObjectStoreProviderSuite) TestGetNonExistingBucket(c *C) {
	if s.osType != ProviderTypeS3 {
		c.Skip("Test only applicable to AWS S3")
	}
	ctx := context.Background()
	bucketName := s.createBucketName(c)
	bucket, err := s.provider.GetBucket(ctx, bucketName)
	c.Check(err, NotNil)
	c.Assert(IsBucketNotFoundError(err), Equals, true)
	c.Check(bucket, IsNil)
}

func (s *ObjectStoreProviderSuite) TestCreateExistingBucketS3Regions(c *C) {
	if s.osType != ProviderTypeS3 {
		c.Skip("Test only applicable to AWS S3")
	}
	ctx := context.Background()
	for _, region := range []string{"us-east-2", testRegionS3, "us-east-1", "us-west-1"} {
		d, err := s.provider.CreateBucket(ctx, testBucketName, region)
		c.Check(err, NotNil)
		c.Check(d, IsNil)
	}
}

// TestDirectories verifies directory operations: create, list, delete
func (s *ObjectStoreProviderSuite) TestDirectories(c *C) {
	ctx := context.Background()
	rootDirectory, err := s.root.CreateDirectory(ctx, s.testDir)
	c.Assert(err, IsNil)
	c.Assert(rootDirectory, NotNil)

	directories, err := rootDirectory.ListDirectories(ctx)
	c.Check(err, IsNil)
	// Expecting nothing
	c.Check(directories, HasLen, 0)

	const (
		dir1 = "directory1"
		dir2 = "directory2"
	)

	directory, err := rootDirectory.CreateDirectory(ctx, dir1)
	c.Assert(err, IsNil)

	// Expecting only /dir1
	directories, err = rootDirectory.ListDirectories(ctx)
	c.Check(err, IsNil)
	c.Check(directories, HasLen, 1)

	_, ok := directories[dir1]
	c.Check(ok, Equals, true)

	// Expecting only /dir1
	directory, err = rootDirectory.GetDirectory(ctx, dir1)
	c.Assert(err, IsNil)

	// Expecting /dir1/dir2
	directory2, err := directory.CreateDirectory(ctx, dir2)
	c.Assert(err, IsNil)

	directories, err = directory2.ListDirectories(ctx)
	c.Check(err, IsNil)
	c.Check(directories, HasLen, 0)

	directories, err = directory.ListDirectories(ctx)
	c.Check(err, IsNil)
	c.Check(directories, HasLen, 1)

	directories, err = rootDirectory.ListDirectories(ctx)
	c.Check(err, IsNil)
	c.Check(directories, HasLen, 1)

	// Get dir1/dir2 from root
	directory2, err = rootDirectory.GetDirectory(ctx, path.Join(dir1, dir2))
	c.Assert(err, IsNil)

	// Get dir1/dir2 from any directory
	d2Name := path.Join(s.testDir, dir1, dir2)
	directory2, err = directory.GetDirectory(ctx, path.Join("/", d2Name))
	c.Assert(err, IsNil)

	// Test delete directory
	// Create objects and directories under dir1/dir2 and under dir1
	_, err = directory2.CreateDirectory(ctx, "d1d2d0")
	c.Assert(err, IsNil)
	_, err = directory2.CreateDirectory(ctx, "d1d2d1")
	c.Assert(err, IsNil)
	err = directory2.PutBytes(ctx, "d1d2o0", nil, nil)
	c.Assert(err, IsNil)

	_, err = directory.CreateDirectory(ctx, "d1d0")
	c.Assert(err, IsNil)
	_, err = directory.CreateDirectory(ctx, "d1d1")
	c.Assert(err, IsNil)
	err = directory.PutBytes(ctx, "d1o0", nil, nil)
	c.Assert(err, IsNil)

	// objects and directories in directory1 should be there
	ds, err := directory.ListDirectories(ctx)
	c.Assert(err, IsNil)
	c.Assert(ds, HasLen, 3)

	err = directory2.DeleteDirectory(ctx)
	c.Assert(err, IsNil)
	cont := getStowContainer(c, directory2)
	checkNoItemsWithPrefix(c, cont, d2Name)
	directory2, err = directory.GetDirectory(ctx, dir2)
	// directory2 should no longer exist
	c.Assert(err, NotNil)
	c.Assert(directory2, IsNil)

	// other objects in directory1 should be there
	ds, err = directory.ListDirectories(ctx)
	c.Assert(err, IsNil)
	c.Assert(ds, HasLen, 2)

	obs, err := directory.ListObjects(ctx)
	c.Assert(err, IsNil)
	c.Assert(obs, HasLen, 1)
	c.Assert(obs[0], Equals, "d1o0")

	directory, err = rootDirectory.GetDirectory(ctx, dir1)
	c.Check(err, IsNil)
	// Delete everything by deleting the parent directory
	err = directory.DeleteDirectory(ctx)
	c.Check(err, IsNil)
	checkNoItemsWithPrefix(c, cont, dir1)
}

func (s *ObjectStoreProviderSuite) TestDeleteAllWithPrefix(c *C) {
	ctx := context.Background()
	rootDirectory, err := s.root.CreateDirectory(ctx, s.testDir)
	c.Assert(err, IsNil)
	c.Assert(rootDirectory, NotNil)
	const (
		dir1 = "directory1"
		dir2 = "directory2"
		dir3 = "directory3"
	)

	directory, err := rootDirectory.CreateDirectory(ctx, dir1)
	c.Assert(err, IsNil)

	// Expecting /dir1/dir2
	_, err = directory.CreateDirectory(ctx, dir2)
	c.Assert(err, IsNil)

	// Expecting root dir to have /dir1/dir2 and /dir3
	_, err = rootDirectory.CreateDirectory(ctx, dir3)
	c.Assert(err, IsNil)

	// Delete everything with prefix "dir1"
	err = rootDirectory.DeleteAllWithPrefix(ctx, dir1)
	c.Assert(err, IsNil)

	// Expecting root dir to have /dir3
	directories, err := rootDirectory.ListDirectories(ctx)
	c.Check(err, IsNil)
	c.Check(directories, HasLen, 1)
	_, ok := directories[dir3]
	c.Check(ok, Equals, true)
}

// TestObjects verifies object operations: GetBytes and PutBytes
func (s *ObjectStoreProviderSuite) TestObjects(c *C) {
	ctx := context.Background()
	rootDirectory, err := s.root.CreateDirectory(ctx, s.testDir)
	c.Assert(err, IsNil)

	const (
		obj1  = "object1"
		data1 = "Some text"

		obj2  = "/some/deep/directory/structure/object2"
		data2 = "Some other text"
	)

	tags := map[string]string{
		"key":  "value",
		"key2": "value2",
	}

	err = rootDirectory.PutBytes(ctx, obj1, []byte(data1), nil)
	c.Check(err, IsNil)

	objs, err := rootDirectory.ListObjects(ctx)
	c.Check(err, IsNil)
	c.Assert(objs, HasLen, 1)
	c.Check(objs[0], Equals, obj1)

	data, _, err := rootDirectory.GetBytes(ctx, obj1)
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte(data1))

	err = rootDirectory.PutBytes(ctx, obj2, []byte(data2), tags)
	data, ntags, err := rootDirectory.GetBytes(ctx, obj2)
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte(data2))
	c.Check(ntags, DeepEquals, tags)

	err = rootDirectory.Delete(ctx, obj1)
	c.Check(err, IsNil)

	err = rootDirectory.Delete(ctx, obj2)
	c.Check(err, IsNil)
}

// TestObjectsStreaming verifies object operations: Get and Put
func (s *ObjectStoreProviderSuite) TestObjectsStreaming(c *C) {
	ctx := context.Background()
	rootDirectory, err := s.root.CreateDirectory(ctx, s.testDir)
	c.Assert(err, IsNil)

	const (
		obj1  = "object1"
		data1 = "Some text"

		obj2  = "/Some/deep/directory/structure/object2"
		data2 = "Some other text"
	)

	tags := map[string]string{
		"key":  "value",
		"key2": "value2",
	}

	data1B := []byte(data1)
	data2B := []byte(data2)

	err = rootDirectory.Put(ctx, obj1, bytes.NewReader(data1B), int64(len(data1B)), nil)
	c.Check(err, IsNil)

	objs, err := rootDirectory.ListObjects(ctx)
	c.Check(err, IsNil)
	c.Assert(objs, HasLen, 1)
	c.Check(objs[0], Equals, obj1)

	r, _, err := rootDirectory.Get(ctx, obj1)
	c.Check(err, IsNil)
	data, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	r.Close()
	c.Check(data, DeepEquals, data1B)

	err = rootDirectory.Put(ctx, obj2, bytes.NewReader(data2B), int64(len(data2B)), tags)
	c.Check(err, IsNil)
	r, ntags, err := rootDirectory.Get(ctx, obj2)
	c.Check(err, IsNil)
	data, err = ioutil.ReadAll(r)
	c.Check(err, IsNil)
	r.Close()
	c.Check(data, DeepEquals, data2B)
	c.Check(ntags, DeepEquals, tags)

	err = rootDirectory.Delete(ctx, obj1)
	c.Check(err, IsNil)

	err = rootDirectory.Delete(ctx, obj2)
	c.Check(err, IsNil)
}

func (s *ObjectStoreProviderSuite) createBucketName(c *C) string {
	// Generate a bucket name
	bucketName := fmt.Sprintf("kio-io-tests-%v-%d", strings.ToLower(c.TestName()), s.rand.Uint32())
	if len(bucketName) > 63 {
		bucketName = bucketName[:62]
	}

	// GCS bucket names cannot contain '.' (except for recognized top-level domains)
	bucketName = strings.Replace(bucketName, ".", "-", -1)

	return bucketName
}

func checkNoItemsWithPrefix(c *C, cont stow.Container, prefix string) {
	items, _, err := cont.Items(prefix, stow.CursorStart, 2)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 0)
}

func (s *ObjectStoreProviderSuite) TestBucketGetRegions(c *C) {
	role := os.Getenv(aws.ConfigRole)
	if s.osType != ProviderTypeS3 || role != "" {
		c.Skip("Test only applicable to S3")
	}
	ctx := context.Background()
	buckets, err := s.provider.ListBuckets(ctx)
	c.Check(err, IsNil)
	for _, region := range []string{testRegionS3, "us-east-1", "us-east-2", "us-west-1"} {
		b, err := GetOrCreateBucket(ctx, s.provider, testBucketName, region)
		c.Check(err, IsNil, Commentf("Region: %s", region))
		c.Check(b, NotNil)
		if b == nil {
			continue
		}
		// Make sure no new bucket was created
		newBuckets, err := s.provider.ListBuckets(ctx)
		c.Check(err, IsNil)
		c.Check(newBuckets, HasLen, len(buckets))

		l, err := b.ListObjects(ctx)
		c.Check(err, IsNil)
		c.Check(l, NotNil)
		objectName := s.suiteDirPrefix + region + "foo"
		err = b.PutBytes(ctx, objectName, []byte("content"), nil)
		c.Check(err, IsNil)
		err = b.Delete(ctx, objectName)
		c.Check(err, IsNil)
	}
}

func getSecret(c *C, ctx context.Context, osType ProviderType) *Secret {
	secret := &Secret{}
	switch osType {
	case ProviderTypeS3:
		secret.Type = SecretTypeAwsAccessKey
		awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
		awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		var awsSessionToken string
		if role, ok := os.LookupEnv(aws.ConfigRole); ok {
			creds, err := aws.SwitchRole(ctx, awsAccessKeyID, awsSecretAccessKey, role, assumeRoleDuration)
			c.Check(err, IsNil)
			val, err := creds.Get()
			c.Check(err, IsNil)
			awsAccessKeyID = val.AccessKeyID
			awsSecretAccessKey = val.SecretAccessKey
			awsSessionToken = val.SessionToken
		}
		secret.Aws = &SecretAws{
			AccessKeyID:     awsAccessKeyID,
			SecretAccessKey: awsSecretAccessKey,
			SessionToken:    awsSessionToken,
		}
		c.Check(secret.Aws.AccessKeyID, Not(Equals), "")
		c.Check(secret.Aws.SecretAccessKey, Not(Equals), "")
	case ProviderTypeGCS:
		creds, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope)
		c.Check(err, IsNil)

		secret.Type = SecretTypeGcpServiceAccountKey
		secret.Gcp = &SecretGcp{
			ServiceKey: string(creds.JSON),
			ProjectID:  creds.ProjectID,
		}
		c.Check(secret.Gcp.ServiceKey, Not(Equals), "")
		c.Check(secret.Gcp.ProjectID, Not(Equals), "")
	case ProviderTypeAzure:
		secret.Type = SecretTypeAzStorageAccount
		secret.Azure = &SecretAzure{
			StorageAccount: os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"),
			StorageKey:     os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
		}
		c.Check(secret.Azure.StorageAccount, Not(Equals), "")
		c.Check(secret.Azure.StorageKey, Not(Equals), "")
	default:
		c.Logf("Unsupported provider '%s'", osType)
		c.Fail()
	}
	return secret
}

// Can be added to a common place in Kanister
func getEnvOrSkip(c *C, varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		c.Skip("Required environment variable '" + varName + "' not set")
	}
	return v
}

func cleanupBucketDirectory(c *C, bucket Bucket, directory string) {
	c.Assert(bucket, NotNil)
	ctx := context.Background()
	d, err := bucket.GetDirectory(ctx, directory)
	if err != nil {
		c.Log("Cannot cleanup test directory: ", directory)
		return
	}
	c.Assert(d, NotNil)
	err = d.DeleteDirectory(ctx)
	c.Check(err, IsNil)
}

// getStowContainer checks that the given directory matches the implementation
// type
func getStowContainer(c *C, d Directory) stow.Container {
	c.Assert(d, FitsTypeOf, &directory{})
	sd, ok := d.(*directory)
	c.Assert(ok, Equals, true)
	c.Assert(sd, NotNil)
	return sd.bucket.container
}
