package objectstore

import (
	"context"

	. "gopkg.in/check.v1"
)

type BucketSuite struct{}

var _ = Suite(&BucketSuite{})

func (s *BucketSuite) SetUpSuite(c *C) {
	getEnvOrSkip(c, "AWS_ACCESS_KEY_ID")
	getEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")
}

const ahmRe = `[\w\W]*AuthorizationHeaderMalformed[\w\W]*`

func (s *BucketSuite) TestInvalidS3RegionEndpointMismatch(c *C) {
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
	c.Assert(err, IsNil)

	// Get Bucket will use the regions correct endpoint.
	_, err = p.GetBucket(ctx, bn)
	c.Assert(IsBucketNotFoundError(err), Equals, true)

	_, err = p.CreateBucket(ctx, bn)
	c.Assert(err, ErrorMatches, ahmRe)
	c.Assert(err, NotNil)

	err = p.DeleteBucket(ctx, bn)
	c.Assert(err, ErrorMatches, ahmRe)
	c.Assert(err, NotNil)
}

func (s *BucketSuite) TestValidS3ClientBucketRegionMismatch(c *C) {
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
	c.Assert(err, IsNil)

	// p2's region does not match the bucket's region, but does not specify an
	// endpoint.
	p2, err := NewProvider(ctx, pc2, secret)
	c.Assert(err, IsNil)

	// p3's region does not match the bucket's region and specifies an endpoint.
	p3, err := NewProvider(ctx, pc3, secret)
	c.Assert(err, IsNil)

	// Delete and recreate the bucket to ensure it's region is r1.
	_ = p1.DeleteBucket(ctx, bn)
	_, err = p1.CreateBucket(ctx, bn)
	c.Assert(err, IsNil)
	defer func() {
		err = p1.DeleteBucket(ctx, bn)
		c.Assert(err, IsNil)
	}()

	// Check the bucket's region is r1
	checkProviderWithBucket(c, ctx, p1, bn, r1)

	// We can read a bucket even though it our provider's does not match, as
	// long as we don't specify an endpoint.
	checkProviderWithBucket(c, ctx, p2, bn, r1)

	// Specifying an endpoint causes this to fail.
	_, err = p3.GetBucket(ctx, bn)
	c.Assert(err, ErrorMatches, ahmRe)
}

func checkProviderWithBucket(c *C, ctx context.Context, p Provider, bucketName, region string) {
	bs, err := p.ListBuckets(ctx)
	c.Assert(err, IsNil)
	_, ok := bs[bucketName]
	c.Assert(ok, Equals, true)
	b, err := p.GetBucket(ctx, bucketName)
	c.Assert(err, IsNil)
	c.Assert(b, NotNil)
	bu, ok := b.(*bucket)
	c.Assert(ok, Equals, true)
	c.Assert(bu, NotNil)
	c.Assert(bu.region, Equals, region)
	_, err = b.ListObjects(ctx)
	c.Assert(err, IsNil)
}
