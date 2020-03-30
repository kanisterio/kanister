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

package zone

import (
	"context"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kube"
)

type KubeTestZoneSuite struct{}

var _ = Suite(&KubeTestZoneSuite{})

func (s KubeTestZoneSuite) TestNodeZones(c *C) {
	c.Skip("Fails in Minikube")
	ctx := context.Background()
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	zones, _, err := NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(zones, Not(HasLen), 0)
}
