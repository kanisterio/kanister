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

package discovery

import (
	"context"

	. "gopkg.in/check.v1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/kanisterio/kanister/pkg/filter"
	"github.com/kanisterio/kanister/pkg/kube"
)

type CRDSuite struct{}

var _ = Suite(&CRDSuite{})

func (s *CRDSuite) TestCRDMatcher(c *C) {
	ctx := context.Background()
	cfg, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := crdclient.NewForConfig(cfg)
	c.Assert(err, IsNil)

	g, err := CRDMatcher(ctx, cli)
	c.Assert(err, IsNil)

	gvrs, err := NamespacedGVRs(ctx, cli.Discovery())
	c.Assert(err, IsNil)
	c.Assert(gvrs, Not(HasLen), 0)

	// We assume there's at least one CRD in the cluster.
	igvrs := filter.GroupVersionResourceList(gvrs).Include(g)
	egvrs := filter.GroupVersionResourceList(gvrs).Exclude(g)
	c.Assert(igvrs, Not(HasLen), 0)
	c.Assert(egvrs, Not(HasLen), 0)
	c.Assert(len(igvrs)+len(egvrs), Equals, len(gvrs))
}
