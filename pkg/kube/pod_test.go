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

// +build !unit

package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type PodSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&PodSuite{})

func (s *PodSuite) SetUpSuite(c *C) {
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podtest-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = ns.Name
}

func (s *PodSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(s.namespace, nil)
		c.Assert(err, IsNil)
	}
}

func (s *PodSuite) TestPod(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pod, err := CreatePod(context.Background(), s.cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        "kanisterio/kanister-tools:0.23.0",
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
	})
	c.Assert(err, IsNil)
	c.Assert(WaitForPodReady(ctx, s.cli, s.namespace, pod.Name), IsNil)
	c.Assert(DeletePod(context.Background(), s.cli, pod), IsNil)
}

func (s *PodSuite) TestPodWithVolumes(c *C) {
	cli := fake.NewSimpleClientset()
	vols := map[string]string{"pvc-test": "/mnt/data1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var p *v1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*v1.Pod)
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p.Status.Phase = v1.PodRunning
		return true, p, nil
	})
	pod, err := CreatePod(ctx, cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        "kanisterio/kanister-tools:0.23.0",
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      vols,
	})
	c.Assert(err, IsNil)
	c.Assert(pod.Spec.Volumes, HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, Equals, "pvc-test")
	c.Assert(pod.Spec.Containers[0].VolumeMounts[0].MountPath, Equals, "/mnt/data1")
}

func (s *PodSuite) TestGetPodLogs(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pod, err := CreatePod(context.Background(), s.cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        "kanisterio/kanister-tools:0.23.0",
		Command:      []string{"sh", "-c", "echo hello"},
	})
	c.Assert(err, IsNil)
	c.Assert(WaitForPodCompletion(ctx, s.cli, s.namespace, pod.Name), IsNil)
	logs, err := GetPodLogs(ctx, s.cli, s.namespace, pod.Name)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(logs, "hello"), Equals, true)
	c.Assert(DeletePod(context.Background(), s.cli, pod), IsNil)
}

func (s *PodSuite) TestPatchDefaultPodSpecs(c *C) {
	defaultSpecs := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:            "container",
				Image:           "kanisterio/kanister-tools:0.23.0",
				Command:         []string{"sh", "-c", "echo in default specs"},
				ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "data",
						MountPath: "/var/lib/data",
					},
				},
			},
		},
		RestartPolicy: v1.RestartPolicyOnFailure,
		Volumes: []v1.Volume{
			{
				Name: "data",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: "default-pvc",
					},
				},
			},
		},
	}

	tests := []struct {
		BlueprintPodSpecs crv1alpha1.JSONMap
		ActionsetPodSpecs crv1alpha1.JSONMap
		Expected          v1.PodSpec
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
			Expected: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyAlways,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
			Expected: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyOnFailure,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
			Expected: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
						VolumeMounts: []v1.VolumeMount{
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
				RestartPolicy: v1.RestartPolicyOnFailure,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
			Expected: v1.PodSpec{
				NodeSelector: map[string]string{
					"selector-key": "selector-value",
				},
				Tolerations: []v1.Toleration{
					{
						Key:      "taint-key",
						Operator: v1.TolerationOpEqual,
						Value:    "taint-value",
						Effect:   v1.TaintEffectNoSchedule,
					},
				},
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyOnFailure,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
			Expected: v1.PodSpec{
				NodeSelector: map[string]string{
					"selector-key": "selector-value",
				},
				Tolerations: []v1.Toleration{
					{
						Key:      "taint-key",
						Operator: v1.TolerationOpEqual,
						Value:    "taint-value",
						Effect:   v1.TaintEffectNoSchedule,
					},
				},
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"echo", "override command"},
						ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyOnFailure,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
			Expected: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"echo", "override command"},
						ImagePullPolicy: v1.PullPolicy(v1.PullAlways),
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyOnFailure,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
			Expected: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "container",
						Image:           "kanisterio/kanister-tools:0.23.0",
						Command:         []string{"sh", "-c", "echo in default specs"},
						ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "data",
								MountPath: "/var/lib/data",
							},
						},
					},
				},
				RestartPolicy: v1.RestartPolicyOnFailure,
				Volumes: []v1.Volume{
					{
						Name: "data",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default-pvc",
							},
						},
					},
				},
				DNSPolicy: v1.DNSClusterFirst,
			},
		},
	}

	// Run tests
	for _, test := range tests {
		override, err := CreateAndMergeJsonPatch(test.BlueprintPodSpecs, test.ActionsetPodSpecs)
		c.Assert(err, IsNil)
		podSpec, err := patchDefaultPodSpecs(defaultSpecs, override)
		c.Assert(err, IsNil)
		c.Assert(podSpec, DeepEquals, test.Expected)
	}

}
