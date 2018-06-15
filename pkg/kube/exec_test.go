// +build !unit

package kube

import (
	. "gopkg.in/check.v1"
)

type ExecSuite struct{}

var _ = Suite(&ExecSuite{})

func (s *ExecSuite) TestExecEcho(c *C) {
	cmd := []string{"sh", "-c", "echo badabing"}
	cli, err := NewClient()
	c.Assert(err, IsNil)
	pods, err := cli.Core().Pods(defaultNamespace).List(emptyListOptions)
	c.Assert(err, IsNil)
	if len(pods.Items) == 0 {
		c.Skip("Test requires a running pod")
	}
	p := pods.Items[0] // We run on all containers in a single pod.
	for _, cs := range p.Status.ContainerStatuses {
		stdout, stderr, err := Exec(cli, p.Namespace, p.Name, cs.Name, cmd)
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "badabing")
		c.Assert(stderr, Equals, "")
	}
}
