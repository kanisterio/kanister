package param

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParamsSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&ParamsSuite{})

func (s *ParamsSuite) SetUpSuite(c *C) {
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterparamstest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *ParamsSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
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

func (s *ParamsSuite) TestFetchStatefulSetParams(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name)
	ss, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	ok := kube.WaitOnStatefulSetReady(ctx, s.cli, ss)
	c.Assert(ok, Equals, true)

	ssp, err := fetchStatefulSetParams(ctx, s.cli, s.namespace, name)
	c.Assert(err, IsNil)
	c.Assert(ssp, DeepEquals, &StatefulSetParams{
		Name:                   name,
		Namespace:              s.namespace,
		Pods:                   []string{name + "-0"},
		Containers:             [][]string{[]string{"test-container"}},
		PersistentVolumeClaims: [][]string{[]string{}},
	})

}

const deploySpec = `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: %s
spec:
  replicas: 1
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

func (s *ParamsSuite) TestFetchDeploymentParams(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(deploySpec, name)
	d, err := kube.CreateDeployment(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	ok := kube.WaitOnDeploymentReady(ctx, s.cli, d)
	c.Assert(ok, Equals, true)

	dp, err := fetchDeploymentParams(ctx, s.cli, s.namespace, name)
	c.Assert(err, IsNil)
	c.Assert(dp.Namespace, Equals, s.namespace)
	c.Assert(dp.Pods, HasLen, 1)
	c.Assert(dp.Containers, DeepEquals, [][]string{{"test-container"}})
	c.Assert(dp.PersistentVolumeClaims, DeepEquals, [][]string{{}})
}

const cmSpec = `
apiVersion: v1
kind: ConfigMap
metadata:
  Name: %s-cm
data:
  someKey: some-value
`

func (s *ParamsSuite) TestNewTemplateParamsDeployment(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(deploySpec, name)
	d, err := kube.CreateDeployment(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	ok := kube.WaitOnDeploymentReady(ctx, s.cli, d)
	c.Assert(ok, Equals, true)

	s.testNewTemplateParams(ctx, c, name, "Deployment")
}

func (s *ParamsSuite) TestNewTemplateParamsStatefulSet(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name)
	ss, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	ok := kube.WaitOnStatefulSetReady(ctx, s.cli, ss)
	c.Assert(ok, Equals, true)

	s.testNewTemplateParams(ctx, c, name, "Statefulset")
}

func (s *ParamsSuite) testNewTemplateParams(ctx context.Context, c *C, name string, kind string) {
	spec := fmt.Sprintf(cmSpec, name)
	cm, err := kube.CreateConfigMap(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	c.Assert(cm, NotNil)

	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Name:      name,
			Namespace: s.namespace,
		},
		ConfigMaps: map[string]crv1alpha1.ObjectReference{
			"myCM": crv1alpha1.ObjectReference{
				Name:      name + "-cm",
				Namespace: s.namespace,
			},
		},
	}
	artsTpl := map[string]crv1alpha1.Artifact{
		"my-art":  crv1alpha1.Artifact{"my-key": "{{ .ConfigMaps.myCM.Data.someKey }}"},
		"my-time": crv1alpha1.Artifact{"my-time": "{{ .Time }}"},
	}
	tp, err := New(ctx, s.cli, as)
	c.Assert(err, IsNil)
	c.Assert(tp.ConfigMaps["myCM"].Data, DeepEquals, map[string]string{"someKey": "some-value"})

	arts, err := RenderArtifacts(artsTpl, *tp)
	c.Assert(err, IsNil)
	c.Assert(arts["my-art"], DeepEquals, crv1alpha1.Artifact{"my-key": "some-value"})
	_, err = time.Parse(timeFormat, arts["my-time"]["my-time"])
	c.Assert(err, IsNil)
}
