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

	"github.com/graymeta/stow"
	"github.com/pkg/errors"
)

var _ Provider = (*provider)(nil)

// provider implements the Provider functionality
type provider struct {
	// e.g., s3-us-west-2.amazonaws.com
	hostEndPoint string
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

// CreateBucket creates the bucket. Bucket naming rules are provider dependent.
func (p *provider) CreateBucket(ctx context.Context, bucketName string) (Bucket, error) {
	location, err := getStowLocation(ctx, p.config, p.secret)
	if err != nil {
		return nil, err
	}
	c, err := location.CreateContainer(bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create bucket %s", bucketName)
	}
	dir := &directory{
		path: "/",
	}
	bucket := &bucket{
		directory:    dir,
		container:    c,
		location:     location,
		hostEndPoint: path.Join(p.hostEndPoint, c.ID()),
		region:       p.config.Region,
	}
	dir.bucket = bucket
	return bucket, nil
}

// GetBucket gets the handle for the specified bucket. Buckets are searched using prefix search;
// if multiple buckets matched the name, then returns an error
func (p *provider) GetBucket(ctx context.Context, bucketName string) (Bucket, error) {
	location, err := getStowLocation(ctx, p.config, p.secret)
	if err != nil {
		return nil, err
	}
	c, err := location.Container(bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get bucket %s", bucketName)
	}
	dir := &directory{
		path: "/",
	}
	bucket := &bucket{
		directory:    dir,
		container:    c,
		location:     location,
		hostEndPoint: path.Join(p.hostEndPoint, c.ID()),
	}
	dir.bucket = bucket
	return bucket, nil
}

// ListBuckets gets the handles of all the buckets.
func (p *provider) ListBuckets(ctx context.Context) (map[string]Bucket, error) {
	// Walk all the buckets
	buckets := make(map[string]Bucket)
	location, err := getStowLocation(ctx, p.config, p.secret)
	if err != nil {
		return nil, err
	}
	err = stow.WalkContainers(location, stow.NoPrefix, 10000,
		func(c stow.Container, err error) error {
			if err != nil {
				return err
			}

			dir := &directory{
				path: "/",
			}
			bucket := &bucket{
				directory:    dir,
				container:    c,
				location:     location,
				hostEndPoint: path.Join(p.hostEndPoint, c.ID()),
			}
			dir.bucket = bucket
			buckets[c.ID()] = bucket
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
	// Attempt creating it
	return p.CreateBucket(ctx, bucketName)
}

type s3Provider struct {
	*provider
}

// Stow uses path-style requests when specifying an endpoint.
// https://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html#path-style-access
// https://github.com/graymeta/stow/blob/master/s3/config.go#L159

const awsS3HostFmt = "https://s3.%s.amazonaws.com"

func awsS3Endpoint(region string) string {
	return fmt.Sprintf(awsS3HostFmt, region)
}

func (p *s3Provider) GetBucket(ctx context.Context, bucketName string) (Bucket, error) {
	cfg := p.config
	var err error
	cfg.Region, err = p.getRegionForBucket(ctx, bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get region for bucket %s", bucketName)
	}
	location, err := getStowLocation(ctx, cfg, p.secret)
	if err != nil {
		return nil, err
	}
	c, err := location.Container(bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get bucket %s", bucketName)
	}
	dir := &directory{
		path: "/",
	}
	hostEndPoint := p.hostEndPoint
	if hostEndPoint == "" {
		hostEndPoint = awsS3Endpoint(cfg.Region)
	}
	bucket := &bucket{
		directory:    dir,
		container:    c,
		location:     location,
		hostEndPoint: path.Join(hostEndPoint, c.ID()),
		region:       cfg.Region,
	}
	dir.bucket = bucket
	return bucket, nil
}

func (p *s3Provider) DeleteBucket(ctx context.Context, bucketName string) error {
	cfg := p.config
	if cfg.Region == "" {
		// We swalllow this error because region may not be required. If it is,
		// we'll fail in the next few lines.
		cfg.Region, _ = p.getRegionForBucket(ctx, bucketName)
	}
	location, err := getStowLocation(ctx, p.config, p.secret)
	if err != nil {
		return errors.Wrapf(err, "Failed to get location for bucket deletion. bucket: %s", bucketName)
	}
	return location.RemoveContainer(bucketName)
}

// returns the region for a particular bucket
func (p *s3Provider) getRegionForBucket(ctx context.Context, bucketName string) (string, error) {
	return GetS3BucketRegion(ctx, bucketName, p.config.Region)
}

func (p *s3Provider) getOrCreateBucket(ctx context.Context, bucketName string) (Bucket, error) {
	d, err := p.GetBucket(ctx, bucketName)
	if IsBucketNotFoundError(err) {
		// Create bucket when it does not exist
		return p.CreateBucket(ctx, bucketName)
	}
	return d, err
}
