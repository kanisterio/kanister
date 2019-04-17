package filter

import (
	"context"

	. "gopkg.in/check.v1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/kanisterio/kanister/pkg/discovery"
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

	gvrs, err := discovery.NamespacedGVRs(ctx, cli.Discovery())
	c.Assert(err, IsNil)
	c.Assert(gvrs, Not(HasLen), 0)

	// We assume there's at least one CRD in the cluster.
	c.Assert(g.Include(gvrs), Not(HasLen), 0)
	c.Assert(g.Exclude(gvrs), Not(HasLen), 0)
	c.Assert(len(g.Exclude(gvrs))+len(g.Include(gvrs)), Equals, len(gvrs))
}
