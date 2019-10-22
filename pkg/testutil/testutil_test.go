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

package testutil

import (
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type TestUtilSuite struct {
}

var _ = Suite(&TestUtilSuite{})

func (s *TestUtilSuite) TestDeployment(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)

	ns := NewTestNamespace()
	ns, err = cli.CoreV1().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	defer func() {
		err := cli.CoreV1().Namespaces().Delete(ns.GetName(), nil)
		c.Assert(err, IsNil)
	}()

	d := NewTestDeployment(1)
	d, err = cli.AppsV1().Deployments(ns.GetName()).Create(d)
	c.Assert(err, IsNil)
	defer func() {
		err = cli.AppsV1().Deployments(ns.GetName()).Delete(d.GetName(), nil)
		c.Assert(err, IsNil)
	}()

	ss := NewTestStatefulSet(1)
	ss, err = cli.AppsV1().StatefulSets(ns.GetName()).Create(ss)
	c.Assert(err, IsNil)
	defer func() {
		err := cli.AppsV1().StatefulSets(ns.GetName()).Delete(ss.GetName(), nil)
		c.Assert(err, IsNil)
	}()

	for _, po := range []struct {
		kind string
		name string
	}{
		{
			kind: "Deployment",
			name: d.GetName(),
		},
		{
			kind: "StatefulSet",
			name: ss.GetName(),
		},
	} {
		testCRs(c, crCli, ns.GetName(), po.kind, po.name)
	}

	cm := NewTestConfigMap()
	cm, err = cli.CoreV1().ConfigMaps(ns.GetName()).Create(cm)
	c.Assert(err, IsNil)
	err = cli.CoreV1().ConfigMaps(ns.GetName()).Delete(cm.GetName(), nil)
	c.Assert(err, IsNil)
}

func testCRs(c *C, cli crclientv1alpha1.CrV1alpha1Interface, namespace, poKind, poName string) {
	var err error
	bp := NewTestBlueprint(poKind, FailFuncName)
	bp = BlueprintWithConfigMap(bp)
	bp, err = cli.Blueprints(namespace).Create(bp)
	c.Assert(err, IsNil)
	defer func() {
		err := cli.Blueprints(namespace).Delete(bp.GetName(), nil)
		c.Assert(err, IsNil)
	}()

	as := NewTestActionSet(namespace, bp.GetName(), poKind, poName, namespace, kanister.DefaultVersion)
	as = ActionSetWithConfigMap(as, "")
	as, err = cli.ActionSets(namespace).Create(as)
	c.Assert(err, IsNil)
	err = cli.ActionSets(namespace).Delete(as.GetName(), nil)
	c.Assert(err, IsNil)
}
