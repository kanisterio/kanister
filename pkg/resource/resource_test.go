package resource

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ResourceSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&ResourceSuite{})

func (s *ResourceSuite) SetUpSuite(c *C) {
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *ResourceSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

var emptyGetOptions metav1.GetOptions

func (s *ResourceSuite) TestActionSetClient(c *C) {
	ctx := context.Background()
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	err = CreateCustomResources(ctx, config)
	c.Assert(err, IsNil)

	name := "testactionset"
	cli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	as1, err := cli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)
	c.Assert(as, NotNil)

	as2, err := cli.ActionSets(s.namespace).Get(name, emptyGetOptions)
	c.Assert(err, IsNil)
	c.Assert(as1, DeepEquals, as2)

	as2.Spec = &crv1alpha1.ActionSetSpec{}
	as3, err := cli.ActionSets(s.namespace).Update(as2)
	c.Assert(err, IsNil)
	c.Assert(as1.Spec, IsNil)
	c.Assert(as3.Spec, NotNil)

	as4, err := cli.ActionSets(s.namespace).Get(name, emptyGetOptions)
	c.Assert(err, IsNil)
	c.Assert(as4, DeepEquals, as3)

	err = cli.ActionSets(s.namespace).Delete(name, nil)
	c.Assert(err, IsNil)

	_, err = cli.ActionSets(s.namespace).Get(name, emptyGetOptions)
	c.Assert(err, NotNil)
}

func (s *ResourceSuite) TestBlueprintClient(c *C) {
	ctx := context.Background()
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	err = CreateCustomResources(ctx, config)
	c.Assert(err, IsNil)

	name := "testblueprint"
	cli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	bp1, err := cli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)
	c.Assert(bp, NotNil)

	bp2, err := cli.Blueprints(s.namespace).Get(name, emptyGetOptions)
	c.Assert(err, IsNil)
	c.Assert(bp1, DeepEquals, bp2)

	bp2.Actions = map[string]*crv1alpha1.BlueprintAction{}
	bp3, err := cli.Blueprints(s.namespace).Update(bp2)
	c.Assert(err, IsNil)
	c.Assert(bp1.Actions, IsNil)
	c.Assert(bp3.Actions, NotNil)

	bp4, err := cli.Blueprints(s.namespace).Get(name, emptyGetOptions)
	c.Assert(err, IsNil)
	c.Assert(bp4, DeepEquals, bp3)

	err = cli.Blueprints(s.namespace).Delete(name, nil)
	c.Assert(err, IsNil)

	_, err = cli.Blueprints(s.namespace).Get(name, emptyGetOptions)
	c.Assert(err, NotNil)
}
