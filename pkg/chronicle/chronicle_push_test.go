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

package chronicle

import (
	"context"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/util/rand"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type ChroniclePushSuite struct{}

var _ = Suite(&ChroniclePushSuite{})

func (s *ChroniclePushSuite) TestPush(c *C) {
	osType := objectstore.ProviderTypeS3
	loc := crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Region: testutil.TestS3Region,
		Bucket: testutil.TestS3BucketName,
	}
	prof := *testutil.ObjectStoreProfileOrSkip(c, osType, loc)
	pp := filepath.Join(c.MkDir(), "profile.json")
	err := writeProfile(pp, prof)
	c.Assert(err, IsNil)

	a := filepath.Join(c.MkDir(), "artifact")
	err = os.WriteFile(a, []byte(rand.String(10)), os.ModePerm)
	c.Assert(err, IsNil)
	p := PushParams{
		ProfilePath:  pp,
		ArtifactFile: a,
		Command:      []string{"echo hello"},
	}
	ctx := context.Background()
	err = push(ctx, p, 0)
	c.Assert(err, IsNil)
}
