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

// Manage bucket operations using Stow

import (
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/graymeta/stow"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

var _ Provider = (*provider)(nil)

// provider implements the Provider functionality
type provider struct {
	// Object store information
	config ProviderConfig
	// Secret
	secret *Secret
}

var _ Bucket = (*bucket)(nil)

// bucket implements the Bucket functionality
type bucket struct {
	*directory                  // bucket is the root directory
	container    stow.Container // stow bucket
	location     stow.Location  // Authenticated stow handle
	hostEndPoint string         // E.g., https://s3-us-west-2.amazonaws.com/bucket1
	region       string         // E.g., us-west-2
}

func newBucket(cfg ProviderConfig, c stow.Container, l stow.Location) *bucket {
	dir := &directory{
		path: "/",
	}
	bucket := &bucket{
		directory:    dir,
		container:    c,
		location:     l,
		hostEndPoint: bucketEndpoint(cfg, c.ID()),
		region:       cfg.Region,
	}
	dir.bucket = bucket
	return bucket
}

// CreateBucket creates the bucket. Bucket naming rules are provider dependent.
func (p *provider) CreateBucket(ctx context.Context, bucketName string) (Bucket, error) {
	cfg, err := p.bucketConfig(ctx, bucketName)
	if err != nil {
		return nil, err
	}
	l, err := getStowLocation(ctx, cfg, p.secret)
	if err != nil {
		return nil, err
	}
	c, err := l.CreateContainer(bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create bucket %s", bucketName)
	}
	return newBucket(cfg, c, l), nil
}

// GetBucket gets the handle for the specified bucket. Buckets are searched using prefix search;
// if multiple buckets matched the name, then returns an error
func (p *provider) GetBucket(ctx context.Context, bucketName string) (Bucket, error) {
	cfg, err := p.bucketConfig(ctx, bucketName)
	if err != nil {
		return nil, err
	}
	l, err := getStowLocation(ctx, cfg, p.secret)
	if err != nil {
		return nil, err
	}
	c, err := l.Container(bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get bucket %s", bucketName)
	}
	return newBucket(cfg, c, l), nil
}

// ListBuckets gets the handles of all the buckets.
func (p *provider) ListBuckets(ctx context.Context) (map[string]Bucket, error) {
	// Walk all the buckets
	buckets := make(map[string]Bucket)
	l, err := getStowLocation(ctx, p.config, p.secret)
	if err != nil {
		return nil, err
	}
	err = stow.WalkContainers(l, stow.NoPrefix, 10000,
		func(c stow.Container, err error) error {
			if err != nil {
				return err
			}
			buckets[c.ID()] = newBucket(p.config, c, l)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return buckets, err
}

// DeleteBucket removes the cloud provider bucket. Does not sanity check.
// For safety, does not delete buckets with contents. Caller should ensure
// that bucket is empty.
func (p *provider) DeleteBucket(ctx context.Context, bucketName string) error {
	location, err := getStowLocation(ctx, p.config, p.secret)
	if err != nil {
		return err
	}
	return location.RemoveContainer(bucketName)
}

func (p *provider) getOrCreateBucket(ctx context.Context, bucketName string) (Bucket, error) {
	d, err := p.GetBucket(ctx, bucketName)
	if err == nil {
		return d, nil
	}
	// Attempt to create it.
	return p.CreateBucket(ctx, bucketName)
}

func (p *provider) bucketConfig(ctx context.Context, bucketName string) (ProviderConfig, error) {
	if p.config.Type == ProviderTypeS3 {
		return s3BucketConfig(ctx, p.config, p.secret, bucketName)
	}
	return p.config, nil
}

func s3BucketConfig(ctx context.Context, c ProviderConfig, s *Secret, bucketName string) (ProviderConfig, error) {
	if s == nil || s.Aws == nil {
		return c, errors.New("AWS Secret required to get region")
	}
	r, err := s3BucketRegion(ctx, c, *s, bucketName)
	if err != nil {
		log.Debug().
			WithContext(ctx).
			WithError(err).
			Print("Couldn't get config for bucket", field.M{"config": c})
		return c, nil
	}
	c.Region = r
	return c, nil
}

type s3Provider struct {
	*provider
}

// GetRegionForBucket returns the region for a particular bucket. It does not
// use the region set in the provider config is used as a hint, but may differ
// from the actual region of the bucket. If the bucket does not have a region,
// then the return value will be "".
func (p *s3Provider) GetRegionForBucket(ctx context.Context, bucketName string) (string, error) {
	if p.secret == nil || p.secret.Aws == nil {
		return "", errors.New("AWS Secret required to get region")
	}
	return s3BucketRegion(ctx, p.config, *p.secret, bucketName)
}

func s3BucketRegion(ctx context.Context, cfg ProviderConfig, sec Secret, bucketName string) (string, error) {
	c, r, err := awsConfig(ctx, cfg, *sec.Aws)
	if err != nil {
		return "", err
	}
	s, err := session.NewSession(c)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create session, region = %s", r)
	}
	svc := s3.New(s)

	// s3-compatible stores may not support s3manager.GetBucketRegion() API, so
	// prefer to use the get-bucket-location API instead
	if cfg.Endpoint != "" {
		resp, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{
			Bucket: aws.String(bucketName),
		})
		if err == nil {
			if resp.LocationConstraint == nil { // per the AWS SDK doc a nil location means us-east-1
				return "us-east-1", nil
			}
			return *resp.LocationConstraint, nil
		}
		log.Error().
			WithContext(ctx).
			WithError(err).
			Print("GetBucketLocation() failed, falling back to GetBucketRegion()", field.M{"config": c})
		// fallback to GetBucketRegion() API if we fail (could be due to
		// access-denied or incorrect policies etc)
	}

	return s3manager.GetBucketRegionWithClient(ctx, svc, bucketName, func(r *request.Request) {
		// GetBucketRegionWithClient() uses credentials.AnonymousCredentials by
		// default which fails the api request in AWS China. We override the
		// creds with the creds used by the client as a workaround.
		r.Config.Credentials = svc.Config.Credentials
	})
}

func (p *s3Provider) getOrCreateBucket(ctx context.Context, bucketName string) (Bucket, error) {
	d, err := p.GetBucket(ctx, bucketName)
	if IsBucketNotFoundError(err) {
		// Create bucket when it does not exist
		return p.CreateBucket(ctx, bucketName)
	}
	return d, err
}

func bucketEndpoint(c ProviderConfig, id string) string {
	e := c.Endpoint
	if c.Type == ProviderTypeS3 {
		e = s3Endpoint(c)
	}
	return path.Join(e, id)
}

const defaultS3region = "us-east-1"

func s3Endpoint(c ProviderConfig) string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	r := defaultS3region
	if c.Region != "" {
		r = c.Region
	}
	return awsS3Endpoint(r)
}

// Stow uses path-style requests when specifying an endpoint.
// https://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html#path-style-access
// https://github.com/graymeta/stow/blob/master/s3/config.go#L159

const awsS3EndpointFmt = "https://s3.%s.amazonaws.com"

func awsS3Endpoint(region string) string {
	return fmt.Sprintf(awsS3EndpointFmt, region)
}
