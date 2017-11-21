package function

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

type ScaleSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&ScaleSuite{})

func (s *ScaleSuite) SetUpSuite(c *C) {
	c.Skip("Too slow")
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterscaletest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *ScaleSuite) TearDownSuite(c *C) {
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

func newScaleBlueprint(name string, scale int) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"someAction": &crv1alpha1.BlueprintAction{
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "ScaleMyApp",
						Func: "ScaleStatefulSet",
						Args: []string{
							"{{ .StatefulSet.Namespace }}",
							name,
							strconv.Itoa(scale),
						},
					},
				},
			},
		},
	}
}

func scaleStatefulSet(c *C, name string, tp kanister.TemplateParams, scale int) {
	phases, err := kanister.GetPhases(*newScaleBlueprint(name, scale), "someAction", tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err := p.Exec(context.Background())
		c.Assert(err, IsNil)
	}
}

func (s *ScaleSuite) TestScaleStatefulSet(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name)
	_, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)

	kind := "statefulset"
	// Run the delete action.
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Name:      name,
			Namespace: s.namespace,
		},
	}
	tp, err := kanister.NewTemplateParams(ctx, s.cli, as)
	c.Assert(err, IsNil)

	// Check that there's only 1 pod.
	_, err = s.cli.Core().Pods(s.namespace).Get(name+"-1", metav1.GetOptions{})
	c.Assert(err, NotNil)

	// Scale up.
	scaleStatefulSet(c, name, *tp, 2)
	p, _ := s.cli.Core().Pods(s.namespace).Get(name+"-1", metav1.GetOptions{})
	c.Assert(p.Status.Phase, Equals, v1.PodRunning)

	// Scale down.
	scaleStatefulSet(c, name, *tp, 0)
	// The pod status can take some time to update. kubectl get pods updates much quicker.
	retryWithTimeout(45*time.Second, func() bool {
		time.Sleep(20 * time.Millisecond)
		p, _ := s.cli.Core().Pods(s.namespace).Get(name+"-0", metav1.GetOptions{})
		return p.Status.Phase != v1.PodRunning
	})
}
