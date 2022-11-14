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
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/util/rand"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ChronicleSuite struct {
	profile param.Profile
}

var _ = Suite(&ChronicleSuite{})

func (s *ChronicleSuite) SetUpSuite(c *C) {
	osType := objectstore.ProviderTypeS3
	loc := crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Region: testutil.TestS3Region,
		Bucket: testutil.TestS3BucketName,
	}
	s.profile = *testutil.ObjectStoreProfileOrSkip(c, osType, loc)
}

func (s *ChronicleSuite) TestPushPull(c *C) {
	pp := filepath.Join(c.MkDir(), "profile.json")
	err := writeProfile(pp, s.profile)
	c.Assert(err, IsNil)

	a := filepath.Join(c.MkDir(), "artifact")
	ap := rand.String(10)
	err = os.WriteFile(a, []byte(ap), os.ModePerm)
	c.Assert(err, IsNil)
	p := PushParams{
		ProfilePath:  pp,
		ArtifactFile: a,
	}
	ctx := context.Background()

	for i := range make([]struct{}, 5) {
		// Write i to bucket
		p.Command = []string{"echo", strconv.Itoa(i)}
		err = push(ctx, p, i)
		c.Assert(err, IsNil)

		// Pull and check that we still get i
		buf := bytes.NewBuffer(nil)
		c.Log("File: ", p.ArtifactFile)
		err = Pull(ctx, buf, s.profile, ap)
		c.Assert(err, IsNil)
		str, err := io.ReadAll(buf)
		c.Assert(err, IsNil)
		// Remove additional '\n'
		t := strings.TrimSuffix(string(str), "\n")
		c.Assert(t, Equals, strconv.Itoa(i))
	}
}

func (s *ChronicleSuite) TestEnv(c *C) {
	ctx := context.Background()
	cmd := []string{"echo", "X:", "$X"}
	suffix := c.TestName() + rand.String(5)
	env := []string{"X=foo"}

	err := pushWithEnv(ctx, cmd, suffix, 0, s.profile, env)
	c.Assert(err, IsNil)
	buf := bytes.NewBuffer(nil)
	err = Pull(ctx, buf, s.profile, suffix)
	c.Assert(err, IsNil)
	str, err := io.ReadAll(buf)
	c.Assert(err, IsNil)
	t := strings.TrimSuffix(string(str), "\n")
	c.Assert(t, Equals, "X: foo")
}
