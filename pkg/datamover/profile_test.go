// Copyright 2023 The Kanister Authors.
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

package datamover

import (
	"bytes"
	"context"
	"path/filepath"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type ProfileSuite struct {
	ctx      context.Context
	location *crv1alpha1.Location
}

var _ = Suite(&ProfileSuite{})

const testContent = "test-content"

func (ps *ProfileSuite) SetUpSuite(c *C) {
	// Set Context as Background
	ps.ctx = context.Background()

	// Set Location
	ps.location = &crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Bucket: testutil.TestS3BucketName,
	}
}

func (ps *ProfileSuite) TestLocationOperationsForProfileDataMover(c *C) {
	p := testutil.ObjectStoreProfileOrSkip(c, objectstore.ProviderTypeS3, *ps.location)
	dir := c.MkDir()
	path := filepath.Join(dir, "test-object1.txt")

	source := bytes.NewBufferString(testContent)
	err := locationPush(ps.ctx, p, path, source)
	c.Assert(err, IsNil)

	target := bytes.NewBuffer(nil)
	err = locationPull(ps.ctx, p, path, target)
	c.Assert(err, IsNil)
	c.Assert(target.String(), Equals, testContent)

	// test deleting single artifact
	err = locationDelete(ps.ctx, p, path)
	c.Assert(err, IsNil)

	// test deleting dir with multiple artifacts
	source = bytes.NewBufferString(testContent)
	err = locationPush(ps.ctx, p, path, source)
	c.Assert(err, IsNil)

	path = filepath.Join(dir, "test-object2.txt")

	source = bytes.NewBufferString(testContent)
	err = locationPush(ps.ctx, p, path, source)
	c.Assert(err, IsNil)

	err = locationDelete(ps.ctx, p, dir)
	c.Assert(err, IsNil)
}
