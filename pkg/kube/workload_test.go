package kube

import (
	"context"

	osapps "github.com/openshift/api/apps/v1"
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type WorkloadSuite struct{}

var _ = Suite(&WorkloadSuite{})

func (s *WorkloadSuite) TestScaleDeploymentConfig(c *C) {
	// Get K8s client
	cfg, err := LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(cfg)
	c.Assert(err, IsNil)

	// Check if we're in OpenShift
	ctx := context.Background()
	ok, err := IsOSAppsGroupAvailable(ctx, cli.Discovery())
	c.Assert(err, IsNil)
	if !ok {
		c.Skip("Skipping test since this only runs on OpenShift")
	}

	// Create a test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dc-scale-test-",
		},
	}
	ns, err = cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		err = cli.CoreV1().Namespaces().Delete(context.TODO(), ns.GetName(), metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}()

	// Create simple DeploymentConfig
	dc := newDeploymentConfig()
	osCli, err := osversioned.NewForConfig(cfg)
	c.Assert(err, IsNil)
	dc, err = osCli.AppsV1().DeploymentConfigs(ns.GetName()).Create(ctx, dc, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = ScaleDeploymentConfig(ctx, cli, osCli, dc.GetNamespace(), dc.GetName(), 0, true)
	c.Assert(err, IsNil)
	err = ScaleDeploymentConfig(ctx, cli, osCli, dc.GetNamespace(), dc.GetName(), 1, true)
	c.Assert(err, IsNil)
}

func newDeploymentConfig() *osapps.DeploymentConfig {
	return &osapps.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "tmp",
		},
		Spec: osapps.DeploymentConfigSpec{
			Replicas: 1,
			Selector: map[string]string{
				"app": "test",
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image:   "alpine",
							Name:    "container",
							Command: []string{"tail", "-f", "/dev/null"},
						},
					},
				},
			},
		},
	}
}
