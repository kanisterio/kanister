package kube_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/jpillora/backoff"
	check "gopkg.in/check.v1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
)

var emptyListOptions = metav1.ListOptions{}

const defaultNamespace = "default"

const k8sTestPrefix = "kubetest-"

// CreateDeployment creates a new deployment for testing
func CreateDeployment(c *check.C, cli kubernetes.Interface, namespace string, labels map[string]string, annotations map[string]string, numVols int) *v1beta1.Deployment {
	var volumeMounts []v1.VolumeMount
	var podVolumes []v1.Volume
	type volRef struct {
		podVolName   string
		podMountPath string
	}
	var vols []volRef
	for i := 0; i < numVols; i++ {
		volName := fmt.Sprintf("vol%d", i)
		mount := fmt.Sprintf("/var/lib/%s", volName)
		vols = append(vols, volRef{volName, mount})
	}

	// Build and create a deployment with multiple volumes
	for _, v := range vols {
		// Create the initial PVC (use dynamic provisioning to create the PV)
		origPVC, err := kube.CreatePVC(context.TODO(), cli, namespace, kube.NoPVCNameSpecified, 1, "", annotations)
		c.Assert(err, check.IsNil)

		volumeMounts = append(volumeMounts, v1.VolumeMount{Name: v.podVolName, MountPath: v.podMountPath})
		podVolumes = append(podVolumes,
			v1.Volume{
				Name: v.podVolName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: origPVC,
					},
				},
			},
		)
	}
	// Create deployment with initial PVCs
	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: k8sTestPrefix,
			Labels:       labels,
			Annotations:  annotations,
		},
		Spec: v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "nginx"}},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:         "nginx",
							Image:        "nginx:1.7.9",
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: podVolumes,
				},
			},
		},
	}
	deployment, err := cli.AppsV1beta1().Deployments(namespace).Create(deployment)
	c.Assert(err, check.IsNil)
	ready := kube.WaitOnDeploymentReady(context.TODO(), cli, deployment)
	c.Assert(ready, check.Equals, true)
	return deployment
}

// DeleteDeployment deletes the specified deployment
func DeleteDeployment(c *check.C, cli kubernetes.Interface, namespace string, deployment *v1beta1.Deployment) {
	now := int64(0)
	err := cli.AppsV1beta1().Deployments(namespace).Delete(deployment.Name, &metav1.DeleteOptions{GracePeriodSeconds: &now})
	c.Assert(err, check.IsNil)

	bo := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    10 * time.Millisecond,
		Max:    10 * time.Minute,
	}

	for {
		_, err := cli.AppsV1beta1().Deployments(namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			// Wait for pods to get deleted
			sel := fmt.Sprintf("d=%s", deployment.Labels["d"])
			options := metav1.ListOptions{LabelSelector: sel}
			pods, err := cli.Core().Pods(namespace).List(options)
			c.Assert(err, check.IsNil)
			if len(pods.Items) == 0 {
				return
			}
			c.Logf("Found %d pods", len(pods.Items))
		}
		//Bail if we hit the max backoff
		if bo.ForAttempt(bo.Attempt()) == bo.Max {
			c.Fatalf("Failed to delete deployment %s", deployment.Name)
		}
		time.Sleep(bo.Duration())
	}
}

func newPVC(name string) v1.PersistentVolumeClaim {
	return v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: *resource.NewQuantity(1, resource.BinarySI),
				},
			},
		},
	}
}

// CreateStatefulSetFromYamlSpec creates a stateful set using the given yaml spec file
func CreateStatefulSetFromYamlSpec(ctx context.Context, c *check.C, cli kubernetes.Interface, namespace string, specFile string) *v1beta1.StatefulSet {
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	buf, err := ioutil.ReadFile(specFile)
	c.Assert(err, check.IsNil)
	spec := fmt.Sprintf(string(buf), name)
	ss, err := kube.CreateStatefulSet(ctx, cli, namespace, spec)
	c.Assert(err, check.IsNil)
	return ss
}
