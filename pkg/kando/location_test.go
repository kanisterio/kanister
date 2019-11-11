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

package kando

import (
	"bytes"
	"context"
	"path/filepath"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type LocationSuite struct{}

var _ = Suite(&LocationSuite{})

const testContent = "test-content"

func (s *LocationSuite) TestLocationObjectStore(c *C) {
	location := crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Bucket: testutil.TestS3BucketName,
	}
	p := testutil.ObjectStoreProfileOrSkip(c, objectstore.ProviderTypeS3, location)
	ctx := context.Background()
	dir := c.MkDir()
	path := filepath.Join(dir, "test-object1.txt")

	source := bytes.NewBufferString(testContent)
	err := locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	target := bytes.NewBuffer(nil)
	err = locationPull(ctx, p, path, target)
	c.Assert(err, IsNil)
	c.Assert(target.String(), Equals, testContent)

	// test deleting single artifact
	err = locationDelete(ctx, p, path)
	c.Assert(err, IsNil)

	//test deleting dir with multiple artifacts
	source = bytes.NewBufferString(testContent)
	err = locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	path = filepath.Join(dir, "test-object2.txt")

	source = bytes.NewBufferString(testContent)
	err = locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	err = locationDelete(ctx, p, dir)
	c.Assert(err, IsNil)

}
