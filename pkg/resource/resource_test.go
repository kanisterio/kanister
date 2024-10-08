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

package resource

import (
	"context"
	"testing"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type ResourceSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = check.Suite(&ResourceSuite{})

func (s *ResourceSuite) SetUpSuite(c *check.C) {
	cli, err := kube.NewClient()
	c.Assert(err, check.IsNil)
	s.cli = cli
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = cns.Name
}

func (s *ResourceSuite) TearDownSuite(c *check.C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

var emptyGetOptions metav1.GetOptions

func (s *ResourceSuite) TestActionSetClient(c *check.C) {
	ctx := context.Background()
	config, err := kube.LoadConfig()
	c.Assert(err, check.IsNil)

	err = CreateCustomResources(ctx, config)
	c.Assert(err, check.IsNil)

	name := "testactionset"
	cli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, check.IsNil)
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	as1, err := cli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(as, check.NotNil)

	as2, err := cli.ActionSets(s.namespace).Get(ctx, name, emptyGetOptions)
	c.Assert(err, check.IsNil)
	c.Assert(as1, check.DeepEquals, as2)

	as2.Spec = &crv1alpha1.ActionSetSpec{}
	as3, err := cli.ActionSets(s.namespace).Update(ctx, as2, metav1.UpdateOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(as1.Spec, check.IsNil)
	c.Assert(as3.Spec, check.NotNil)

	as4, err := cli.ActionSets(s.namespace).Get(ctx, name, emptyGetOptions)
	c.Assert(err, check.IsNil)
	c.Assert(as4, check.DeepEquals, as3)

	err = cli.ActionSets(s.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	c.Assert(err, check.IsNil)

	_, err = cli.ActionSets(s.namespace).Get(ctx, name, emptyGetOptions)
	c.Assert(err, check.NotNil)
}

func (s *ResourceSuite) TestBlueprintClient(c *check.C) {
	ctx := context.Background()
	config, err := kube.LoadConfig()
	c.Assert(err, check.IsNil)

	err = CreateCustomResources(ctx, config)
	c.Assert(err, check.IsNil)

	name := "testblueprint"
	cli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, check.IsNil)
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	bp1, err := cli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(bp, check.NotNil)

	bp2, err := cli.Blueprints(s.namespace).Get(ctx, name, emptyGetOptions)
	c.Assert(err, check.IsNil)
	c.Assert(bp1, check.DeepEquals, bp2)
	bp2.Actions = map[string]*crv1alpha1.BlueprintAction{
		"backup": {
			Name: "takebackup",
		},
	}
	bp3, err := cli.Blueprints(s.namespace).Update(ctx, bp2, metav1.UpdateOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(bp1.Actions, check.IsNil)
	c.Assert(bp3.Actions, check.NotNil)

	bp4, err := cli.Blueprints(s.namespace).Get(ctx, name, emptyGetOptions)
	c.Assert(err, check.IsNil)
	c.Assert(bp4, check.DeepEquals, bp3)

	err = cli.Blueprints(s.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	c.Assert(err, check.IsNil)

	_, err = cli.Blueprints(s.namespace).Get(ctx, name, emptyGetOptions)
	c.Assert(err, check.NotNil)
}
