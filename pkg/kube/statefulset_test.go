// +build !unit

package kube

import (
	"context"
	"fmt"
	"strings"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type StatefulSetSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&StatefulSetSuite{})

func (s *StatefulSetSuite) SetUpSuite(c *C) {
	c.Skip("Too slow")
	cli, err := NewClient()
	c.Assert(err, IsNil)
	s.cli = cli
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "statefulsettest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *StatefulSetSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.Core().Namespaces().Delete(s.namespace, nil)
		c.Assert(err, IsNil)
	}
}

const ssSpec = `
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: %s
spec:
  replicas: 1
  serviceName: fake-svc
  template:
    metadata:
      labels:
        app: fake-app
    spec:
      containers:
        - name: test-container
          image: alpine:3.6
          command: ["tail"]
          args: ["-f", "/dev/null"]
`

func (s *StatefulSetSuite) TestCreateStatefulSet(c *C) {
	ctx := context.Background()
	// Stateful set names have strict requirements.
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name)
	_, err := CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	defer func() {
		err = s.cli.AppsV1beta1().StatefulSets(s.namespace).Delete(name, nil)
		c.Assert(err, IsNil)
	}()
	_, err = s.cli.Core().Pods(s.namespace).Get(name+"-0", metav1.GetOptions{})
	c.Assert(err, IsNil)
}
