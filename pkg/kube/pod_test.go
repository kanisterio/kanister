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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
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

var _ = Suite(&PodSuite{})

func (s *PodSuite) SetUpSuite(c *C) {
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podtest-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = ns.Name

	err = os.Setenv("POD_NAMESPACE", ns.Name)
	c.Assert(err, IsNil)

	err = os.Setenv("POD_SERVICE_ACCOUNT", controllerSA)
	c.Assert(err, IsNil)

	err = s.createServiceAccount(testSAName, s.namespace)
	c.Assert(err, IsNil)

	err = s.createServiceAccount(controllerSA, s.namespace)
	c.Assert(err, IsNil)
}

func (s *PodSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}

func (s *PodSuite) TestPod(c *C) {
	// get controllers's namespace
	cns, err := GetControllerNamespace()
	c.Assert(err, IsNil)

	// get controller's SA
	sa, err := GetControllerServiceAccount(fake.NewSimpleClientset())
	c.Assert(err, IsNil)

	testSec := s.createTestSecret(c)
	defer func() {
		err = s.cli.CoreV1().Secrets(testSec.Namespace).Delete(context.Background(), testSec.Name, metav1.DeleteOptions{})
		c.Log("Failed to delete test secret: ", testSec.Name)
	}()
	ctx := context.Background()
	podOptions := []*PodOptions{
		{
			Namespace:    s.namespace,
			GenerateName: "test-",
			Image:        consts.LatestKanisterToolsImage,
			Command:      []string{"sh", "-c", "tail -f /dev/null"},
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
		},
	}

	for _, po := range podOptions {
		pod, err := CreatePod(context.Background(), s.cli, po)

		// we have not specified the SA, if the pod is being created in the
		// same ns as controller's, controller's SA should have been set.
		if po.ServiceAccountName == "" && po.Namespace == cns {
			c.Assert(pod.Spec.ServiceAccountName, Equals, sa)
		} else {
			var expectedSA string
			if po.ServiceAccountName == "" {
				expectedSA = "default"
			} else {
				expectedSA = po.ServiceAccountName
			}
			c.Assert(pod.Spec.ServiceAccountName, Equals, expectedSA)
		}

		if po.Annotations != nil {
			c.Check(pod.ObjectMeta.Annotations, NotNil)
			c.Check(pod.ObjectMeta.Annotations, DeepEquals, po.Annotations)
		}

		if po.Name != "" {
			c.Assert(pod.ObjectMeta.Name, Equals, po.Name)
		}

		c.Check(len(pod.ObjectMeta.Labels), Equals, len(po.Labels)+1)
		c.Check(pod.ObjectMeta.Labels[consts.LabelKeyCreatedBy], Equals, consts.LabelValueKanister)
		for key, value := range po.Labels {
			c.Check(pod.ObjectMeta.Labels[key], Equals, value)
		}

		if po.Resources.Limits != nil {
			c.Assert(pod.Spec.Containers[0].Resources.Limits, NotNil)
			c.Assert(pod.Spec.Containers[0].Resources.Limits, DeepEquals, po.Resources.Limits)
		}
		if po.Resources.Requests != nil {
			c.Assert(pod.Spec.Containers[0].Resources.Requests, NotNil)
			c.Assert(pod.Spec.Containers[0].Resources.Requests, DeepEquals, po.Resources.Requests)
		}

		switch {
		case po.ContainerName != "":
			c.Assert(pod.Spec.Containers[0].Name, Equals, po.ContainerName)
		default:
			c.Assert(pod.Spec.Containers[0].Name, Equals, DefaultContainerName)
		}

		switch {
		case po.RestartPolicy == "":
			c.Assert(pod.Spec.RestartPolicy, Equals, corev1.RestartPolicyNever)
		default:
			c.Assert(pod.Spec.RestartPolicy, Equals, po.RestartPolicy)
		}

		if po.EnvironmentVariables != nil && len(po.EnvironmentVariables) > 0 {
			c.Assert(pod.Spec.Containers[0].Env, DeepEquals, po.EnvironmentVariables)
		}

		c.Assert(err, IsNil)
		c.Assert(WaitForPodReady(ctx, s.cli, po.Namespace, pod.Name), IsNil)
		c.Assert(DeletePod(context.Background(), s.cli, pod), IsNil)
	}
}

func (s *PodSuite) createTestSecret(c *C) *corev1.Secret {
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"key": "value",
		},
	}
	testSecret, err := s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), testSecret, metav1.CreateOptions{})
	c.Assert(err, IsNil)
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

func (s *PodSuite) TestPodWithFilesystemModeVolumes(c *C) {
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
	c.Assert(err, IsNil)
	vols := map[string]VolumeMountOptions{pvc.Name: {MountPath: "/mnt/data1", ReadOnly: PVCContainsReadOnlyAccessMode(pvc)}}
	ctx := context.Background()
	var p *corev1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*corev1.Pod)
		if len(p.Spec.Volumes[0].Name) > 63 {
			return true, nil, errors.New("spec.volumes[0].name must be no more than 63 characters")
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
	c.Assert(err, IsNil)
	c.Assert(WaitForPodReady(ctx, cli, s.namespace, pod.Name), IsNil)
	c.Assert(pod.Spec.Volumes, HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, Equals, pvcName)
	c.Assert(pod.Spec.Containers[0].VolumeMounts[0].MountPath, Equals, "/mnt/data1")
	c.Assert(len(pod.Spec.Containers[0].VolumeDevices), Equals, 0)
}

func (s *PodSuite) TestPodWithFilesystemModeReadOnlyVolumes(c *C) {
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
	c.Assert(err, IsNil)
	vols := map[string]VolumeMountOptions{pvc.Name: {MountPath: "/mnt/data1", ReadOnly: PVCContainsReadOnlyAccessMode(pvc)}}
	ctx := context.Background()
	var p *corev1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*corev1.Pod)
		if len(p.Spec.Volumes[0].Name) > 63 {
			return true, nil, errors.New("spec.volumes[0].name must be no more than 63 characters")
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
	c.Assert(err, IsNil)
	c.Assert(WaitForPodReady(ctx, cli, s.namespace, pod.Name), IsNil)
	c.Assert(pod.Spec.Volumes, HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, Equals, pvcName)
	c.Assert(pod.Spec.Containers[0].VolumeMounts[0].MountPath, Equals, "/mnt/data1")
	c.Assert(len(pod.Spec.Containers[0].VolumeDevices), Equals, 0)
}

func (s *PodSuite) TestPodWithBlockModeVolumes(c *C) {
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
	c.Assert(err, IsNil)
	vols := map[string]string{pvc.Name: "/mnt/data1"}
	ctx := context.Background()
	var p *corev1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*corev1.Pod)
		if len(p.Spec.Volumes[0].Name) > 63 {
			return true, nil, errors.New("spec.volumes[0].name must be no more than 63 characters")
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
	c.Assert(err, IsNil)
	c.Assert(WaitForPodReady(ctx, cli, s.namespace, pod.Name), IsNil)
	c.Assert(pod.Spec.Volumes, HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, Equals, pvcName)
	c.Assert(len(pod.Spec.Containers[0].VolumeMounts), Equals, 0)
	c.Assert(pod.Spec.Containers[0].VolumeDevices[0].DevicePath, Equals, "/mnt/data1")
}

func (s *PodSuite) TestGetPodLogs(c *C) {
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
	c.Assert(err, IsNil)
	c.Assert(WaitForPodCompletion(ctx, s.cli, s.namespace, pod.Name), IsNil)
	logs, err := GetPodLogs(ctx, s.cli, s.namespace, pod.Name, pod.Spec.Containers[0].Name)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(logs, "hello"), Equals, true)
	c.Assert(DeletePod(context.Background(), s.cli, pod), IsNil)
}

func (s *PodSuite) TestPatchDefaultPodSpecs(c *C) {
	defaultSpecs := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "container",
				Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
						Image:           "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
		c.Assert(err, IsNil)
		podSpec, err := patchDefaultPodSpecs(defaultSpecs, override)
		c.Assert(err, IsNil)
		c.Assert(podSpec, DeepEquals, test.Expected)
	}
}

func (s *PodSuite) TestGetPodReadyWaitTimeout(c *C) {
	// Setup ENV to change the default timeout
	err := os.Setenv(PodReadyWaitTimeoutEnv, "5")
	c.Assert(err, IsNil)
	c.Assert(GetPodReadyWaitTimeout(), Equals, time.Minute*5)
	err = os.Unsetenv(PodReadyWaitTimeoutEnv)
	c.Assert(err, IsNil)
	// Check without ENV set
	c.Assert(GetPodReadyWaitTimeout(), Equals, DefaultPodReadyWaitTimeout)
}

func (s *PodSuite) TestSetPodSecurityContext(c *C) {
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
	c.Assert(err, IsNil)
	runAsNonRootExpected := true
	c.Assert(pod.Spec.SecurityContext.RunAsNonRoot, DeepEquals, &runAsNonRootExpected)
	var uidAndGIDExpected int64 = 1000
	c.Assert(*pod.Spec.SecurityContext.RunAsUser, DeepEquals, uidAndGIDExpected)
	c.Assert(*pod.Spec.SecurityContext.RunAsGroup, DeepEquals, uidAndGIDExpected)
}

func (s *PodSuite) TestSetPodSecurityContextOverridesPodOverride(c *C) {
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
	c.Assert(err, IsNil)
	runAsNonRootExpected := true
	c.Assert(pod.Spec.SecurityContext.RunAsNonRoot, DeepEquals, &runAsNonRootExpected)
	var uidAndGIDExpected int64 = 1000
	c.Assert(*pod.Spec.SecurityContext.RunAsUser, DeepEquals, uidAndGIDExpected)
	c.Assert(*pod.Spec.SecurityContext.RunAsGroup, DeepEquals, uidAndGIDExpected)
}

func (s *PodSuite) TestSetLifecycleHook(c *C) {
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
	c.Assert(err, IsNil)
	c.Assert(pod.Spec.Containers[0].Lifecycle, DeepEquals, lch)
}

func (s *PodSuite) TestGetRedactedOptions(c *C) {
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

	c.Assert(po1.Namespace, Equals, opts.Namespace)
	c.Assert(po1.GenerateName, Equals, opts.GenerateName)
	c.Assert(po1.Image, Equals, opts.Image)
	c.Assert(po1.Command, DeepEquals, []string{redactedValue, redactedValue, redactedValue})
	c.Assert(po1.EnvironmentVariables, DeepEquals, []corev1.EnvVar{
		{Name: "abc", Value: redactedValue},
		{Name: "ooo", Value: redactedValue},
	})
	c.Assert(po1.PodOverride, DeepEquals, crv1alpha1.JSONMap{
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

	c.Assert(po2.Namespace, Equals, s.namespace)
	c.Assert(po2.Image, Equals, consts.LatestKanisterToolsImage)
	c.Assert(po2.Command, IsNil)
	c.Assert(po2.EnvironmentVariables, IsNil)
	c.Assert(po2.PodOverride, DeepEquals, crv1alpha1.JSONMap{
		"volumes":    []corev1.Volume{{Name: "Fake volume"}},
		"containers": 123,
	})
}

func (s *PodSuite) TestGetRedactedPod(c *C) {
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

	c.Assert(p1.TypeMeta, DeepEquals, pod.TypeMeta)
	c.Assert(len(p1.Spec.Containers), Equals, len(pod.Spec.Containers))
	c.Assert(p1.Spec.Containers, DeepEquals, []corev1.Container{
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

func (s *PodControllerTestSuite) TestContainerNameFromPodOptsOrDefault(c *C) {
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
		c.Assert(name, Equals, tc.expectedContainerName)
	}

	name := ContainerNameFromPodOptsOrDefault(&PodOptions{})
	c.Assert(name, Equals, DefaultContainerName)

	name = ContainerNameFromPodOptsOrDefault(nil)
	c.Assert(name, Equals, DefaultContainerName)
}

func (s *PodSuite) TestAddLabels(c *C) {
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
		c.Assert(tc.podOptions, DeepEquals, tc.expectedPodOptions)
	}
}

func (s *PodSuite) TestAddAnnotations(c *C) {
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
		c.Assert(tc.podOptions, DeepEquals, tc.expectedPodOptions)
	}
}
