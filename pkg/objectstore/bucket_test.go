package objectstore

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/check.v1"
)

type BucketSuite struct{}

var _ = check.Suite(&BucketSuite{})

func (s *BucketSuite) SetUpSuite(c *check.C) {
	getEnvOrSkip(c, "AWS_ACCESS_KEY_ID")
	getEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")
}

const ahmRe = `[\w\W]*AuthorizationHeaderMalformed[\w\W]*`

func (s *BucketSuite) TestInvalidS3RegionEndpointMismatch(c *check.C) {
	ctx := context.Background()
	const pt = ProviderTypeS3
	const bn = `kanister-fake-bucket`
	const r = `us-east-1`
	const endpoint = `https://s3.us-gov-west-1.amazonaws.com`

	secret := getSecret(ctx, c, pt)
	p, err := NewProvider(
		ctx,
		ProviderConfig{
			Type:     pt,
			Endpoint: endpoint,
			Region:   r,
		},
		secret,
	)
	c.Assert(err, check.IsNil)

	// Get Bucket will use the region's correct endpoint.
	_, err = p.GetBucket(ctx, bn)
	c.Assert(err, check.ErrorMatches, ahmRe)
	c.Assert(err, check.NotNil)

	_, err = p.CreateBucket(ctx, bn)
	c.Assert(err, check.ErrorMatches, ahmRe)
	c.Assert(err, check.NotNil)

	err = p.DeleteBucket(ctx, bn)
	c.Assert(err, check.ErrorMatches, ahmRe)
	c.Assert(err, check.NotNil)
}

func (s *BucketSuite) TestValidS3ClientBucketRegionMismatch(c *check.C) {
	if useMinio, ok := os.LookupEnv("USE_MINIO"); ok && useMinio == "true" {
		c.Skip("Skipping region mismatch test because MinIO always returns the same region")
	}

	ctx := context.Background()
	const pt = ProviderTypeS3
	const bn = `kanister-test-bucket-us-west-1`
	const r1 = `us-west-1`
	const r2 = `us-west-2`

	pc1 := ProviderConfig{
		Type:     pt,
		Endpoint: awsS3Endpoint(r1),
		Region:   r1,
	}

	pc2 := ProviderConfig{
		Type:   pt,
		Region: r2,
	}

	pc3 := ProviderConfig{
		Type:     pt,
		Endpoint: awsS3Endpoint(r2),
		Region:   r2,
	}

	secret := getSecret(ctx, c, pt)

	// p1's region matches the bucket's region.
	p1, err := NewProvider(ctx, pc1, secret)
	c.Assert(err, check.IsNil)

	// p2's region does not match the bucket's region, but does not specify an
	// endpoint.
	p2, err := NewProvider(ctx, pc2, secret)
	c.Assert(err, check.IsNil)

	// p3's region does not match the bucket's region and specifies an endpoint.
	p3, err := NewProvider(ctx, pc3, secret)
	c.Assert(err, check.IsNil)

	// Delete and recreate the bucket to ensure it's region is r1.
	_ = p1.DeleteBucket(ctx, bn)
	_, err = p1.CreateBucket(ctx, bn)
	c.Assert(err, check.IsNil)
	defer func() {
		err = p1.DeleteBucket(ctx, bn)
		c.Assert(err, check.IsNil)
	}()

	// Check the bucket's region is r1
	err = checkProviderWithBucket(ctx, c, p1, bn, r1)
	c.Assert(err, check.IsNil)

	// We can read a bucket even though it our provider's does not match, as
	// long as we don't specify an endpoint.
	err = checkProviderWithBucket(ctx, c, p2, bn, r1)
	c.Assert(err, check.IsNil)

	// Specifying an the wrong endpoint causes bucket ops to fail.
	err = checkProviderWithBucket(ctx, c, p3, bn, r1)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ahmRe)
}

func checkProviderWithBucket(ctx context.Context, c *check.C, p Provider, bucketName, region string) error {
	bs, err := p.ListBuckets(ctx)
	c.Assert(err, check.IsNil)
	_, ok := bs[bucketName]
	c.Assert(ok, check.Equals, true)
	// We should fail here if the endpoint is set and does not match bucket region.
	b, err := p.GetBucket(ctx, bucketName)
	if err != nil {
		return err
	}
	c.Assert(err, check.IsNil)
	c.Assert(b, check.NotNil)

	s3p, ok := p.(*s3Provider)
	c.Assert(ok, check.Equals, true)
	c.Assert(s3p, check.NotNil)
	r, err := s3p.GetRegionForBucket(ctx, bucketName)
	c.Assert(err, check.IsNil)
	c.Assert(r, check.Equals, region)

	_, err = b.ListObjects(ctx)
	c.Assert(err, check.IsNil)
	return nil
}

func (s *BucketSuite) TestGetRegionForBucket(c *check.C) {
	if useMinio, ok := os.LookupEnv("USE_MINIO"); ok && useMinio == "true" {
		c.Skip("Skipping region mismatch test because MinIO always returns the same region")
	}

	ctx := context.Background()
	const pt = ProviderTypeS3
	secret := getSecret(ctx, c, pt)

	// Ensure existingBucket exists and non-existing bucket does not
	const existingBucket = testBucketName
	const nonExistentBucket = "kanister-test-should-not-exist"
	minioEnpoint := os.Getenv("LOCATION_ENDPOINT")
	pc := ProviderConfig{
		Type:   pt,
		Region: testRegionS3,
		//Region:   "tom-minio-region",
		Endpoint: minioEnpoint,
	}
	p, err := NewProvider(ctx, pc, secret)
	c.Assert(err, check.IsNil)
	_, err = p.getOrCreateBucket(ctx, existingBucket)
	c.Log(fmt.Sprintf("%+v", err))
	c.Assert(err, check.IsNil)
	bucket, err := p.GetBucket(ctx, nonExistentBucket)
	c.Log(bucket, err)
	c.Assert(err, check.NotNil)
	c.Assert(IsBucketNotFoundError(err), check.Equals, true)

	for _, tc := range []struct {
		bucketName   string
		endpoint     string
		clientRegion string
		bucketRegion string
		valid        bool
	}{
		{
			bucketName:   existingBucket,
			endpoint:     "",
			clientRegion: "",
			bucketRegion: testRegionS3,
			valid:        true,
		},
		{
			bucketName:   existingBucket,
			endpoint:     "",
			clientRegion: "us-west-1",
			bucketRegion: testRegionS3,
			valid:        true,
		},
		{
			bucketName:   existingBucket,
			endpoint:     "",
			clientRegion: testRegionS3,
			bucketRegion: testRegionS3,
			valid:        true,
		},
		{
			bucketName:   existingBucket,
			endpoint:     "",
			clientRegion: "asdf",
			bucketRegion: testRegionS3,
			valid:        false,
		},
		{
			bucketName:   nonExistentBucket,
			endpoint:     "",
			clientRegion: testRegionS3,
			bucketRegion: "",
			valid:        false,
		},
		{
			bucketName:   nonExistentBucket,
			endpoint:     "",
			clientRegion: "",
			bucketRegion: "",
			valid:        false,
		},
		// We don't yet have credentials for the following in CI, but can be
		// used for manual tests
		{
			bucketName:   existingBucket,
			endpoint:     minioEnpoint,
			clientRegion: "tom-minio-region",
			bucketRegion: "tom-minio-region",
			valid:        false,
		},
		{
			bucketName:   existingBucket,
			endpoint:     minioEnpoint,
			clientRegion: "asdf",
			bucketRegion: "tom-minio-region",
			valid:        false,
		},
		// {
		// 	bucketName:   existingBucket,
		// 	endpoint:     "https://play.min.io:9000",
		// 	clientRegion: "",
		// 	bucketRegion: "minio-region",
		// 	valid:        false,
		// },
		{
			bucketName:   "kanister-test-govcloud",
			endpoint:     "",
			clientRegion: "us-gov-east-1",
			bucketRegion: "us-gov-west-1",
			valid:        false,
		},
	} {
		p, err := NewProvider(
			ctx,
			ProviderConfig{
				Type:     pt,
				Endpoint: tc.endpoint,
				Region:   tc.clientRegion,
			},
			secret,
		)
		c.Assert(err, check.IsNil)
		cmt := check.Commentf("Case: %#v", tc)

		sp, ok := p.(*s3Provider)
		c.Assert(ok, check.Equals, true)
		rfb, err := sp.GetRegionForBucket(ctx, tc.bucketName)
		if tc.valid {
			c.Assert(err, check.IsNil, cmt)
			c.Assert(rfb, check.Equals, tc.bucketRegion, cmt)
		} else {
			c.Assert(err, check.NotNil, cmt)
		}
	}
}
