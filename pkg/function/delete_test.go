package function

import (
	"context"
	"fmt"
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

type DeleteSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&DeleteSuite{})

func (s *DeleteSuite) SetUpSuite(c *C) {
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterdeletetest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *DeleteSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

const ssWithVolume = `
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: %s
spec:
  replicas: 2
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
          volumeMounts:
            - name: empty-volume
              mountPath: /mnt/data
      volumes:
      - name: empty-volume
        persistentVolumeClaim:
          claimName: empty-volume
  volumeClaimTemplates:
  - metadata:
      name: empty-volume
      annotations:
        volume.alpha.kubernetes.io/storage-class: anything
    spec:
      accessModes:
        - "ReadWriteOnce"
      resources:
        requests:
          storage: 1Gi
`

func newDeleteBlueprint(name string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"someAction": &crv1alpha1.BlueprintAction{
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "DeletePVC",
						Func: "DeleteGeneratedResources",
						Args: []string{
							"pvc",
							"{{ .StatefulSet.Namespace }}",
							fmt.Sprintf("{{ index .StatefulSet.PersistentVolumeClaims 0 0 }}-%s-", name),
							"{{ len .StatefulSet.PersistentVolumeClaims }}",
						},
					},
				},
			},
		},
	}
}

func (s *DeleteSuite) TestDelete(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssWithVolume, name)
	_, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)

	// Check that the volumes exists.
	for i := 0; i < 2; i++ {
		pvcName := fmt.Sprintf("%s-%s-%d", "empty-volume", name, i)
		retryWithTimeout(10*time.Second, func() bool {
			_, err = s.cli.Core().PersistentVolumeClaims(s.namespace).Get(pvcName, metav1.GetOptions{})
			return err == nil
		})
	}

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

	phases, err := kanister.GetPhases(*newDeleteBlueprint(name), "someAction", *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err = p.Exec(context.Background())
		c.Assert(err, IsNil)
	}

	// Check to make sure the volume was deleted.
	for i := 0; i < 2; i++ {
		pvcName := fmt.Sprintf("%s-%s-%d", "empty-volume", name, i)
		_, err = s.cli.Core().PersistentVolumeClaims(s.namespace).Get(pvcName, metav1.GetOptions{})
		c.Assert(err, NotNil)
	}
}

func retryWithTimeout(t time.Duration, f func() bool) {
	ch := time.Tick(t)
	for {
		if f() {
			return
		}
		select {
		case <-ch:
			return
		default:
		}
	}
}
