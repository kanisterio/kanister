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

func (s *BucketSuite) TestRegionEndpointMismatch(c *C) {
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
		},
		secret,
	)
	c.Assert(err, IsNil)

	const ahmRe = `[\w\W]*AuthorizationHeaderMalformed[\w\W]*`

	_, err = p.GetBucket(ctx, bn, r)
	c.Assert(err, ErrorMatches, ahmRe)
	c.Assert(err, NotNil)

	_, err = p.CreateBucket(ctx, bn, r)
	c.Assert(err, ErrorMatches, ahmRe)
	c.Assert(err, NotNil)

	err = p.DeleteBucket(ctx, bn, r)
	c.Assert(err, ErrorMatches, ahmRe)
	c.Assert(err, NotNil)

	err = p.DeleteBucket(ctx, bn, r)
	c.Assert(err, ErrorMatches, ahmRe)
	c.Assert(err, NotNil)
}
