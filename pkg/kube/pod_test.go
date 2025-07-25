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
	"os"
	"strings"
	"time"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
)

type PodSuite struct {
	cli       kubernetes.Interface
	namespace string
}

const (
	testSAName   = "test-sa"
	controllerSA = "controller-sa"
)

var _ = check.Suite(&PodSuite{})

func (s *PodSuite) SetUpSuite(c *check.C) {
	var err error
	s.cli, err = NewClient()
	c.Assert(err, check.IsNil)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podtest-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = ns.Name

	err = os.Setenv("POD_NAMESPACE", ns.Name)
	c.Assert(err, check.IsNil)

	err = os.Setenv("POD_SERVICE_ACCOUNT", controllerSA)
	c.Assert(err, check.IsNil)

	err = s.createServiceAccount(testSAName, s.namespace)
	c.Assert(err, check.IsNil)

	err = s.createServiceAccount(controllerSA, s.namespace)
	c.Assert(err, check.IsNil)
}

func (s *PodSuite) TearDownSuite(c *check.C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, check.IsNil)
	}
}

func (s *PodSuite) TestPod(c *check.C) {
	// get controllers's namespace
	cns, err := GetControllerNamespace()
	c.Assert(err, check.IsNil)

	// get controller's SA
	sa, err := GetControllerServiceAccount(fake.NewSimpleClientset())
	c.Assert(err, check.IsNil)

	testSec := s.createTestSecret(c)
	defer func() {
		err = s.cli.CoreV1().Secrets(testSec.Namespace).Delete(context.Background(), testSec.Name, metav1.DeleteOptions{})
		c.Log("Failed to delete test secret: ", testSec.Name)
	}()
	ctx := context.Background()

	nodes, err := s.cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	c.Assert(err, check.IsNil)

	if len(nodes.Items) < 1 {
		c.Log("Failed to list nodes from the cluster, to set to PodOptions")
		c.Fail()
	}
	nodeName := nodes.Items[0].Name

	podOptions := []*PodOptions{
		{
			Namespace:    s.namespace,
			GenerateName: "test-",
			Image:        consts.LatestKanisterToolsImage,
			Command:      []string{"sh", "-c", "tail -f /dev/null"},
			NodeName:     nodeName,
		},
		{
			Namespace:          s.namespace,
			GenerateName:       "test-",
			Image:              consts.LatestKanisterToolsImage,
			Command:            []string{"sh", "-c", "tail -f /dev/null"},
			ServiceAccountName: testSAName,
			RestartPolicy:      corev1.RestartPolicyAlways,
		},
		{
			Namespace:     cns,
			GenerateName:  "test-",
			Image:         consts.LatestKanisterToolsImage,
			Command:       []string{"sh", "-c", "tail -f /dev/null"},
			RestartPolicy: corev1.RestartPolicyOnFailure,
			NodeName:      nodeName,
		},
		{
			Namespace:          cns,
			GenerateName:       "test-",
			Image:              consts.LatestKanisterToolsImage,
			Command:            []string{"sh", "-c", "tail -f /dev/null"},
			ServiceAccountName: testSAName,
			RestartPolicy:      corev1.RestartPolicyNever,
		},
		{
			Namespace:    s.namespace,
			GenerateName: "test-",
			Image:        consts.LatestKanisterToolsImage,
			Command:      []string{"sh", "-c", "tail -f /dev/null"},
			Annotations: map[string]string{
				"test-annotation": "true",
			},
		},
		{
			Namespace:    s.namespace,
			GenerateName: "test-",
			Image:        consts.LatestKanisterToolsImage,
			Command:      []string{"sh", "-c", "tail -f /dev/null"},
			Labels: map[string]string{
				"run": "pod",
			},
			NodeName: nodeName,
		},
		{
			Namespace:    s.namespace,
			GenerateName: "test-",
			Image:        consts.LatestKanisterToolsImage,
			Command:      []string{"sh", "-c", "tail -f /dev/null"},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"memory": resource.MustParse("100Mi"),
					"cpu":    resource.MustParse("100m"),
				},
				Requests: corev1.ResourceList{
					"memory": resource.MustParse("100Mi"),
					"cpu":    resource.MustParse("100m"),
				},
			},
		},
		{
			Namespace:     s.namespace,
			GenerateName:  "test-",
			Image:         consts.LatestKanisterToolsImage,
			ContainerName: "test-container",
			Command:       []string{"sh", "-c", "tail -f /dev/null"},
			Labels: map[string]string{
				"run": "pod",
			},
		},
		{
			Namespace:     s.namespace,
			Name:          "testpod",
			GenerateName:  "test-",
			Image:         consts.LatestKanisterToolsImage,
			ContainerName: "test-container",
			Command:       []string{"sh", "-c", "tail -f /dev/null"},
			Labels: map[string]string{
				"run": "pod",
			},
		},
		{
			Namespace:    s.namespace,
			GenerateName: "test-",
			Image:        consts.LatestKanisterToolsImage,
			Command:      []string{"sh", "-c", "tail -f /dev/null"},
			EnvironmentVariables: []corev1.EnvVar{
				{
					Name:  "test-env",
					Value: "test-value",
				},
			},
			NodeName: nodeName,
		},
	}

	for _, po := range podOptions {
		pod, err := CreatePod(context.Background(), s.cli, po)

		// we have not specified the SA, if the pod is being created in the
		// same ns as controller's, controller's SA should have been set.
		if po.ServiceAccountName == "" && po.Namespace == cns {
			c.Assert(pod.Spec.ServiceAccountName, check.Equals, sa)
		} else {
			var expectedSA string
			if po.ServiceAccountName == "" {
				expectedSA = "default"
			} else {
				expectedSA = po.ServiceAccountName
			}
			c.Assert(pod.Spec.ServiceAccountName, check.Equals, expectedSA)
		}

		if po.Annotations != nil {
			c.Check(pod.ObjectMeta.Annotations, check.NotNil)
			c.Check(pod.ObjectMeta.Annotations, check.DeepEquals, po.Annotations)
		}

		if po.Name != "" {
			c.Assert(pod.ObjectMeta.Name, check.Equals, po.Name)
		}

		c.Check(len(pod.ObjectMeta.Labels), check.Equals, len(po.Labels)+1)
		c.Check(pod.ObjectMeta.Labels[consts.LabelKeyCreatedBy], check.Equals, consts.LabelValueKanister)
		for key, value := range po.Labels {
			c.Check(pod.ObjectMeta.Labels[key], check.Equals, value)
		}

		if po.Resources.Limits != nil {
			c.Assert(pod.Spec.Containers[0].Resources.Limits, check.NotNil)
			c.Assert(pod.Spec.Containers[0].Resources.Limits, check.DeepEquals, po.Resources.Limits)
		}
		if po.Resources.Requests != nil {
			c.Assert(pod.Spec.Containers[0].Resources.Requests, check.NotNil)
			c.Assert(pod.Spec.Containers[0].Resources.Requests, check.DeepEquals, po.Resources.Requests)
		}

		if po.ContainerName != "" {
			c.Assert(pod.Spec.Containers[0].Name, check.Equals, po.ContainerName)
		} else {
			c.Assert(pod.Spec.Containers[0].Name, check.Equals, DefaultContainerName)
		}

		if po.RestartPolicy == "" {
			c.Assert(pod.Spec.RestartPolicy, check.Equals, corev1.RestartPolicyNever)
		} else {
			c.Assert(pod.Spec.RestartPolicy, check.Equals, po.RestartPolicy)
		}

		if len(po.EnvironmentVariables) > 0 {
			c.Assert(pod.Spec.Containers[0].Env, check.DeepEquals, po.EnvironmentVariables)
		}

		c.Assert(err, check.IsNil)
		c.Assert(WaitForPodReady(ctx, s.cli, po.Namespace, pod.Name), check.IsNil)

		// make sure the nodeName set in podOptions is, actually assinged to the pod
		if po.NodeName != "" {
			c.Assert(po.NodeName, check.Equals, pod.Spec.NodeName)
		}

		c.Assert(DeletePod(context.Background(), s.cli, pod), check.IsNil)
	}
}

func (s *PodSuite) createTestSecret(c *check.C) *corev1.Secret {
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"key": "value",
		},
	}
	testSecret, err := s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), testSecret, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	return testSecret
}

func (s *PodSuite) createServiceAccount(name, ns string) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if _, err := s.cli.CoreV1().ServiceAccounts(ns).Create(context.TODO(), &sa, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (s *PodSuite) TestPodWithFilesystemModeVolumes(c *check.C) {
	cli := fake.NewSimpleClientset()
	pvcName := "prometheus-ibm-monitoring-prometheus-db-prometheus-ibm-monitoring-prometheus-0"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
	pvc, err := cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	vols := map[string]VolumeMountOptions{pvc.Name: {MountPath: "/mnt/data1", ReadOnly: PVCContainsReadOnlyAccessMode(pvc)}}
	ctx := context.Background()
	var p *corev1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*corev1.Pod)
		if len(p.Spec.Volumes[0].Name) > 63 {
			return true, nil, errkit.New("spec.volumes[0].name must be no more than 63 characters")
		}
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p.Status.Phase = corev1.PodRunning
		return true, p, nil
	})
	pod, err := CreatePod(ctx, cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      vols,
	})
	c.Assert(err, check.IsNil)
	c.Assert(WaitForPodReady(ctx, cli, s.namespace, pod.Name), check.IsNil)
	c.Assert(pod.Spec.Volumes, check.HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, check.Equals, pvcName)
	c.Assert(pod.Spec.Containers[0].VolumeMounts[0].MountPath, check.Equals, "/mnt/data1")
	c.Assert(len(pod.Spec.Containers[0].VolumeDevices), check.Equals, 0)
}

func (s *PodSuite) TestPodWithFilesystemModeReadOnlyVolumes(c *check.C) {
	cli := fake.NewSimpleClientset()
	pvcName := "pvc-with-read-only-mount"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
	pvc, err := cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	vols := map[string]VolumeMountOptions{pvc.Name: {MountPath: "/mnt/data1", ReadOnly: PVCContainsReadOnlyAccessMode(pvc)}}
	ctx := context.Background()
	var p *corev1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*corev1.Pod)
		if len(p.Spec.Volumes[0].Name) > 63 {
			return true, nil, errkit.New("spec.volumes[0].name must be no more than 63 characters")
		}
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p.Status.Phase = corev1.PodRunning
		return true, p, nil
	})
	pod, err := CreatePod(ctx, cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      vols,
	})
	c.Assert(err, check.IsNil)
	c.Assert(WaitForPodReady(ctx, cli, s.namespace, pod.Name), check.IsNil)
	c.Assert(pod.Spec.Volumes, check.HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, check.Equals, pvcName)
	c.Assert(pod.Spec.Containers[0].VolumeMounts[0].MountPath, check.Equals, "/mnt/data1")
	c.Assert(len(pod.Spec.Containers[0].VolumeDevices), check.Equals, 0)
}

func (s *PodSuite) TestPodWithBlockModeVolumes(c *check.C) {
	cli := fake.NewSimpleClientset()
	pvcName := "block-mode-volume"
	blockMode := corev1.PersistentVolumeBlock
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			VolumeMode:  &blockMode,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
	pvc, err := cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	vols := map[string]string{pvc.Name: "/mnt/data1"}
	ctx := context.Background()
	var p *corev1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*corev1.Pod)
		if len(p.Spec.Volumes[0].Name) > 63 {
			return true, nil, errkit.New("spec.volumes[0].name must be no more than 63 characters")
		}
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p.Status.Phase = corev1.PodRunning
		return true, p, nil
	})
	pod, err := CreatePod(ctx, cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		BlockVolumes: vols,
	})
	c.Assert(err, check.IsNil)
	c.Assert(WaitForPodReady(ctx, cli, s.namespace, pod.Name), check.IsNil)
	c.Assert(pod.Spec.Volumes, check.HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, check.Equals, pvcName)
	c.Assert(len(pod.Spec.Containers[0].VolumeMounts), check.Equals, 0)
	c.Assert(pod.Spec.Containers[0].VolumeDevices[0].DevicePath, check.Equals, "/mnt/data1")
}

func (s *PodSuite) TestGetPodLogs(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pod, err := CreatePod(context.Background(), s.cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "echo hello"},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []corev1.Container{{
				Name:    "sidecar",
				Image:   consts.LatestKanisterToolsImage,
				Command: []string{"sh", "-c", "echo sidecar"},
			}},
		},
	})
	c.Assert(err, check.IsNil)
	c.Assert(WaitForPodCompletion(ctx, s.cli, s.namespace, pod.Name), check.IsNil)
	logs, err := GetPodLogs(ctx, s.cli, s.namespace, pod.Name, pod.Spec.Containers[0].Name)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(logs, "hello"), check.Equals, true)
	c.Assert(DeletePod(context.Background(), s.cli, pod), check.IsNil)
}

func (s *PodSuite) TestPatchDefaultPodSpecs(c *check.C) {
	defaultSpecs := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "container",
				Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
				Command:         []string{"sh", "-c", "echo in default specs"},
				ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "data",
						MountPath: "/var/lib/data",
					},
				},
			},
		},
		RestartPolicy: corev1.RestartPolicyOnFailure,
		Volumes: []corev1.Volume{
			{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "default-pvc",
					},
				},
			},
		},
	}

	tests := []struct {
		BlueprintPodSpecs crv1alpha1.JSONMap
		ActionsetPodSpecs crv1alpha1.JSONMap
		Expected          corev1.PodSpec
	}{
		// Blueprint and Actionset PodOverride specs are nil
		{
			BlueprintPodSpecs: nil,
			ActionsetPodSpecs: nil,
			Expected:          defaultSpecs,
		},

		// Blueprint PodOverride specs are nil
		{
			BlueprintPodSpecs: nil,
			ActionsetPodSpecs: crv1alpha1.JSONMap{
				"restartPolicy": "Always",
			},
			Expected: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyAlways,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
			},
		},

		// Actionset PodOverride specs are nil
		{
			BlueprintPodSpecs: crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name":            "container",
						"imagePullPolicy": "IfNotPresent",
					},
				},
			},
			ActionsetPodSpecs: nil,
			Expected: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
			},
		},

		// Modify volume mounts
		{
			BlueprintPodSpecs: crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name": "container",
						"volumeMounts": []map[string]interface{}{
							{
								"mountPath": "/var/lib/other",
								"name":      "data",
							},
						},
					},
				},
			},
			ActionsetPodSpecs: crv1alpha1.JSONMap{
				"volumes": []map[string]interface{}{
					{
						"name": "data",
						"persistentVolumeClaim": map[string]interface{}{
							"claimName": "other-claim",
						},
					},
				},
			},
			Expected: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/other",
							},
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "other-claim",
							},
						},
					},
				},
			},
		},

		// Add NodeSelector and Tolerations
		{
			BlueprintPodSpecs: crv1alpha1.JSONMap{
				"nodeSelector": map[string]interface{}{
					"selector-key": "selector-value",
				},
			},
			ActionsetPodSpecs: crv1alpha1.JSONMap{
				"tolerations": []map[string]interface{}{
					{
						"key":      "taint-key",
						"operator": "Equal",
						"value":    "taint-value",
						"effect":   "NoSchedule",
					},
				},
			},
			Expected: corev1.PodSpec{
				NodeSelector: map[string]string{
					"selector-key": "selector-value",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "taint-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "taint-value",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
			},
		},

		// Add NodeSelector and Tolerations. Override container command
		{
			BlueprintPodSpecs: crv1alpha1.JSONMap{
				"nodeSelector": map[string]interface{}{
					"selector-key": "selector-value",
				},
				"tolerations": []map[string]interface{}{
					{
						"key":      "taint-key",
						"operator": "Equal",
						"value":    "taint-value",
						"effect":   "NoSchedule",
					},
				},
			},
			ActionsetPodSpecs: crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name":    "container",
						"command": []string{"echo", "override command"},
					},
				},
			},
			Expected: corev1.PodSpec{
				NodeSelector: map[string]string{
					"selector-key": "selector-value",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "taint-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "taint-value",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"echo", "override command"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
			},
		},

		// Override container command
		{
			BlueprintPodSpecs: crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name":    "container",
						"command": []string{"echo", "override command"},
					},
				},
			},
			ActionsetPodSpecs: nil,
			Expected: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"echo", "override command"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
			},
		},

		// Override blueprint specs with actionset
		{
			BlueprintPodSpecs: crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name":            "container",
						"imagePullPolicy": "IfNotPresent",
					},
				},
				"dnsPolicy": "Default",
			},
			ActionsetPodSpecs: crv1alpha1.JSONMap{
				"dnsPolicy": "ClusterFirst",
			},
			Expected: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "container",
						Image:           "ghcr.io/kanisterio/kanister-tools:0.114.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
				DNSPolicy: corev1.DNSClusterFirst,
			},
		},
	}

	// Run tests
	for _, test := range tests {
		override, err := CreateAndMergeJSONPatch(test.BlueprintPodSpecs, test.ActionsetPodSpecs)
		c.Assert(err, check.IsNil)
		podSpec, err := PatchDefaultPodSpecs(defaultSpecs, override)
		c.Assert(err, check.IsNil)
		c.Assert(podSpec, check.DeepEquals, test.Expected)
	}
}

func (s *PodSuite) TestGetPodReadyWaitTimeout(c *check.C) {
	// Setup ENV to change the default timeout
	err := os.Setenv(PodReadyWaitTimeoutEnv, "5")
	c.Assert(err, check.IsNil)
	c.Assert(GetPodReadyWaitTimeout(), check.Equals, time.Minute*5)
	err = os.Unsetenv(PodReadyWaitTimeoutEnv)
	c.Assert(err, check.IsNil)
	// Check without ENV set
	c.Assert(GetPodReadyWaitTimeout(), check.Equals, DefaultPodReadyWaitTimeout)
}

func (s *PodSuite) TestSetPodSecurityContext(c *check.C) {
	po := &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodSecurityContext: &corev1.PodSecurityContext{
			RunAsUser:    &[]int64{1000}[0],
			RunAsGroup:   &[]int64{1000}[0],
			RunAsNonRoot: &[]bool{true}[0],
		},
	}

	pod, err := CreatePod(context.Background(), s.cli, po)
	c.Assert(err, check.IsNil)
	runAsNonRootExpected := true
	c.Assert(pod.Spec.SecurityContext.RunAsNonRoot, check.DeepEquals, &runAsNonRootExpected)
	var uidAndGIDExpected int64 = 1000
	c.Assert(*pod.Spec.SecurityContext.RunAsUser, check.DeepEquals, uidAndGIDExpected)
	c.Assert(*pod.Spec.SecurityContext.RunAsGroup, check.DeepEquals, uidAndGIDExpected)
}

func (s *PodSuite) TestSetPodSecurityContextOverridesPodOverride(c *check.C) {
	po := &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodSecurityContext: &corev1.PodSecurityContext{
			RunAsUser:    &[]int64{1000}[0],
			RunAsGroup:   &[]int64{1000}[0],
			RunAsNonRoot: &[]bool{true}[0],
		},
		PodOverride: crv1alpha1.JSONMap{
			"securityContext": map[string]interface{}{
				"runAsUser":    2000,
				"runAsGroup":   2000,
				"runAsNonRoot": false,
			},
		},
	}

	pod, err := CreatePod(context.Background(), s.cli, po)
	c.Assert(err, check.IsNil)
	runAsNonRootExpected := true
	c.Assert(pod.Spec.SecurityContext.RunAsNonRoot, check.DeepEquals, &runAsNonRootExpected)
	var uidAndGIDExpected int64 = 1000
	c.Assert(*pod.Spec.SecurityContext.RunAsUser, check.DeepEquals, uidAndGIDExpected)
	c.Assert(*pod.Spec.SecurityContext.RunAsGroup, check.DeepEquals, uidAndGIDExpected)
}

func (s *PodSuite) TestSetLifecycleHook(c *check.C) {
	lch := &corev1.Lifecycle{
		PostStart: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/bash", "-c", "echo 1"},
			},
		},
	}

	po := &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Lifecycle:    lch,
	}

	pod, err := CreatePod(context.Background(), s.cli, po)
	c.Assert(err, check.IsNil)
	c.Assert(pod.Spec.Containers[0].Lifecycle, check.DeepEquals, lch)
}

func (s *PodSuite) TestGetRedactedOptions(c *check.C) {
	opts := &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		EnvironmentVariables: []corev1.EnvVar{
			{Name: "abc", Value: "def", ValueFrom: &corev1.EnvVarSource{}},
			{Name: "ooo", Value: "aaa", ValueFrom: &corev1.EnvVarSource{}},
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []corev1.Container{{
				Name:    "sidecar",
				Image:   consts.LatestKanisterToolsImage,
				Command: []string{"sh", "-c", "echo sidecar"},
				Env: []corev1.EnvVar{
					{Name: "a1", Value: "v1"},
					{Name: "a2", Value: "v2"},
				},
			}},
		},
	}

	po1 := getRedactedOptions(opts)

	c.Assert(po1.Namespace, check.Equals, opts.Namespace)
	c.Assert(po1.GenerateName, check.Equals, opts.GenerateName)
	c.Assert(po1.Image, check.Equals, opts.Image)
	c.Assert(po1.Command, check.DeepEquals, []string{redactedValue, redactedValue, redactedValue})
	c.Assert(po1.EnvironmentVariables, check.DeepEquals, []corev1.EnvVar{
		{Name: "abc", Value: redactedValue},
		{Name: "ooo", Value: redactedValue},
	})
	c.Assert(po1.PodOverride, check.DeepEquals, crv1alpha1.JSONMap{
		"containers": []corev1.Container{{
			Name:    "sidecar",
			Image:   consts.LatestKanisterToolsImage,
			Command: []string{redactedValue, redactedValue, redactedValue},
			Env: []corev1.EnvVar{
				{Name: "a1", Value: redactedValue},
				{Name: "a2", Value: redactedValue},
			},
		}},
	})

	po2 := getRedactedOptions(&PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        consts.LatestKanisterToolsImage,
		PodOverride: crv1alpha1.JSONMap{
			"volumes":    []corev1.Volume{{Name: "Fake volume"}},
			"containers": 123, // Check that non []corev1.Container value will not break anything
		},
	})

	c.Assert(po2.Namespace, check.Equals, s.namespace)
	c.Assert(po2.Image, check.Equals, consts.LatestKanisterToolsImage)
	c.Assert(po2.Command, check.IsNil)
	c.Assert(po2.EnvironmentVariables, check.IsNil)
	c.Assert(po2.PodOverride, check.DeepEquals, crv1alpha1.JSONMap{
		"volumes":    []corev1.Volume{{Name: "Fake volume"}},
		"containers": 123,
	})
}

func (s *PodSuite) TestGetRedactedPod(c *check.C) {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Some kind",
			APIVersion: "FakeAPI-1.0",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "c1",
					Image: "img1",
					Args:  []string{"a", "b", "c"},
					Env: []corev1.EnvVar{
						{Name: "ev1", Value: "23", ValueFrom: &corev1.EnvVarSource{}},
						{Name: "ev2", Value: "dd", ValueFrom: &corev1.EnvVarSource{}},
					},
				},
				{
					Name:    "c2",
					Image:   "img2",
					Command: []string{"sh", "-c", "tail -f /dev/null"},
					Env: []corev1.EnvVar{
						{Name: "a1", Value: "v1", ValueFrom: &corev1.EnvVarSource{}},
						{Name: "a2", Value: "v2", ValueFrom: &corev1.EnvVarSource{}},
					},
				},
			},
		},
	}

	p1 := getRedactedPod(pod)

	c.Assert(p1.TypeMeta, check.DeepEquals, pod.TypeMeta)
	c.Assert(len(p1.Spec.Containers), check.Equals, len(pod.Spec.Containers))
	c.Assert(p1.Spec.Containers, check.DeepEquals, []corev1.Container{
		{
			Name:  "c1",
			Image: "img1",
			Args:  []string{redactedValue, redactedValue, redactedValue},
			Env: []corev1.EnvVar{
				{Name: "ev1", Value: redactedValue},
				{Name: "ev2", Value: redactedValue},
			},
		},
		{
			Name:    "c2",
			Image:   "img2",
			Command: []string{redactedValue, redactedValue, redactedValue},
			Env: []corev1.EnvVar{
				{Name: "a1", Value: redactedValue},
				{Name: "a2", Value: redactedValue},
			},
		},
	})
}

func (s *PodControllerTestSuite) TestContainerNameFromPodOptsOrDefault(c *check.C) {
	for _, tc := range []struct {
		podOptsContainerName  string
		expectedContainerName string
	}{
		{
			podOptsContainerName:  "conone",
			expectedContainerName: "conone",
		},
		{
			podOptsContainerName:  "",
			expectedContainerName: DefaultContainerName,
		},
	} {
		name := ContainerNameFromPodOptsOrDefault(&PodOptions{
			ContainerName: tc.podOptsContainerName,
		})
		c.Assert(name, check.Equals, tc.expectedContainerName)
	}

	name := ContainerNameFromPodOptsOrDefault(&PodOptions{})
	c.Assert(name, check.Equals, DefaultContainerName)

	name = ContainerNameFromPodOptsOrDefault(nil)
	c.Assert(name, check.Equals, DefaultContainerName)
}

func (s *PodSuite) TestAddLabels(c *check.C) {
	for _, tc := range []struct {
		podOptions         *PodOptions
		labels             map[string]string
		expectedPodOptions *PodOptions
	}{
		{
			podOptions: &PodOptions{},
			labels: map[string]string{
				"keyOne": "valOne",
			},
			expectedPodOptions: &PodOptions{
				Labels: map[string]string{
					"keyOne": "valOne",
				},
			},
		},
		{
			podOptions: nil,
			labels: map[string]string{
				"keyOne": "valOne",
			},
			expectedPodOptions: nil,
		},
		{
			podOptions: &PodOptions{
				Labels: map[string]string{
					"key": "val",
				},
			},
			labels: map[string]string{
				"keyOne": "valOne",
			},
			expectedPodOptions: &PodOptions{
				Labels: map[string]string{
					"key":    "val",
					"keyOne": "valOne",
				},
			},
		},
		{
			podOptions: &PodOptions{
				Labels: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
				},
			},
			labels: map[string]string{
				"keyOne": "valOne",
				"keyTwo": "valTwo",
			},
			expectedPodOptions: &PodOptions{
				Labels: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
					"keyOne":  "valOne",
					"keyTwo":  "valTwo",
				},
			},
		},
		{
			podOptions: &PodOptions{
				Labels: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
				},
			},
			labels: nil,
			expectedPodOptions: &PodOptions{
				Labels: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
				},
			},
		},
	} {
		tc.podOptions.AddLabels(tc.labels)
		c.Assert(tc.podOptions, check.DeepEquals, tc.expectedPodOptions)
	}
}

func (s *PodSuite) TestAddAnnotations(c *check.C) {
	for _, tc := range []struct {
		podOptions         *PodOptions
		annotations        map[string]string
		expectedPodOptions *PodOptions
	}{
		{
			podOptions: &PodOptions{},
			annotations: map[string]string{
				"keyOne": "valOne",
			},
			expectedPodOptions: &PodOptions{
				Annotations: map[string]string{
					"keyOne": "valOne",
				},
			},
		},
		{
			podOptions: nil,
			annotations: map[string]string{
				"keyOne": "valOne",
			},
			expectedPodOptions: nil,
		},
		{
			podOptions: &PodOptions{
				Annotations: map[string]string{
					"key": "val",
				},
			},
			annotations: map[string]string{
				"keyOne": "valOne",
			},
			expectedPodOptions: &PodOptions{
				Annotations: map[string]string{
					"key":    "val",
					"keyOne": "valOne",
				},
			},
		},
		{
			podOptions: &PodOptions{
				Annotations: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
				},
			},
			annotations: map[string]string{
				"keyOne": "valOne",
				"keyTwo": "valTwo",
			},
			expectedPodOptions: &PodOptions{
				Annotations: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
					"keyOne":  "valOne",
					"keyTwo":  "valTwo",
				},
			},
		},
		{
			podOptions: &PodOptions{
				Annotations: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
				},
			},
			annotations: nil,
			expectedPodOptions: &PodOptions{
				Annotations: map[string]string{
					"key":     "val",
					"keyZero": "valZero",
				},
			},
		},
	} {
		tc.podOptions.AddAnnotations(tc.annotations)
		c.Assert(tc.podOptions, check.DeepEquals, tc.expectedPodOptions)
	}
}

// TestErrkitApiErrorsWrapping verifies that apierrors wrapped with errkit.Wrap are still matchable using apierrors matchers
func (s *PodSuite) TestErrkitApiErrorsWrapping(c *check.C) {
	// Create the fake client
	fakeClient := fake.NewSimpleClientset()

	// Add a reactor to simulate an error when trying to get a PVC
	fakeClient.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errkit.Wrap(apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetSubresource()), "Some context")
	})

	_, err := fakeClient.CoreV1().PersistentVolumeClaims("abc").Get(context.TODO(), "def", metav1.GetOptions{})
	if err != nil {
		c.Assert(apierrors.IsNotFound(err), check.Equals, true)
	}
}
