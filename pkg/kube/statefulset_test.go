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

//go:build !unit
// +build !unit

package kube

import (
	"context"
	"fmt"
	"strings"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
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
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "statefulsettest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *StatefulSetSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}

const ssSpec = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
spec:
  replicas: 1
  serviceName: fake-svc
  selector:
    matchLabels:
      app: fake-app
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
		err = s.cli.AppsV1().StatefulSets(s.namespace).Delete(ctx, name, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}()
	_, err = s.cli.CoreV1().Pods(s.namespace).Get(ctx, name+"-0", metav1.GetOptions{})
	c.Assert(err, IsNil)
}
