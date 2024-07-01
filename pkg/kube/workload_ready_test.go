package kube

import (
	"context"
	"time"

	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/errorchecker"
)

type WorkloadReadySuite struct{}

var _ = Suite(&WorkloadReadySuite{})

type cliParams struct {
	name                    string
	namespace               string
	stateful                bool
	specReplicas            int32
	statusReplicas          int32
	statusUpdatedReplicas   int32
	statusAvailableReplicas int32
	generation              int64
	observedGeneration      int64
	podStatus               corev1.PodPhase
}

// These tests can be used to force the various error states
func (s *WorkloadReadySuite) TestWaitOnStatefulSetReady(c *C) {
	testCases := []struct {
		input cliParams
		want  string
	}{
		{
			input: cliParams{"ss", "default", true, 1, 1, 1, 1, 1, 1, "Running"},
			want:  "",
		}, {
			input: cliParams{"ss", "default", true, 5, 2, 5, 5, 1, 1, "Running"},
			want:  "Specified 5 replicas and only 2 are ready.*",
		}, {
			input: cliParams{"ss", "default", true, 1, 1, 1, 1, 1, 1, "Failed"},
			want:  "Specified 1 replicas and only 0 are running.*",
		},
	}
	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/4)
		defer cancel()
		err := WaitOnStatefulSetReady(ctx, getCli(tc.input), tc.input.namespace, tc.input.name)
		if tc.want != "" {
			errorchecker.AssertErrorMessage(c, err, tc.want)
		} else {
			c.Assert(err, IsNil)
		}
	}
}

func (s *WorkloadReadySuite) TestStatefulSetReady(c *C) {
	cp := cliParams{"ss", "default", true, 1, 1, 1, 1, 1, 1, "Running"}
	ctx := context.Background()
	ready, status, err := StatefulSetReady(ctx, getCli(cp), cp.namespace, cp.name)
	c.Assert(status, DeepEquals, "")
	c.Assert(ready, DeepEquals, true)
	c.Assert(err, IsNil)
}

func (s *WorkloadReadySuite) TestWaitOnDeploymentReady(c *C) {
	testCases := []struct {
		input cliParams
		want  string
	}{
		{
			input: cliParams{"dep", "default", false, 1, 1, 1, 1, 1, 1, "Running"},
			want:  "",
		}, {
			input: cliParams{"dep", "default", false, 5, 2, 5, 5, 1, 1, "Running"},
			want:  "Specified 5 replicas and only have 2.*",
		}, {
			input: cliParams{"dep", "default", false, 5, 5, 2, 5, 1, 1, "Running"},
			want:  "Specified 5 replicas and only have 2 updated replicas.*",
		}, {
			input: cliParams{"dep", "default", false, 5, 5, 5, 2, 1, 1, "Running"},
			want:  "Specified 5 replicas and only have 2 available replicas.*",
		}, {
			input: cliParams{"dep", "default", false, 1, 1, 1, 1, 1, 0, "Running"},
			want:  "Need generation of at least 1 and observed 0.*",
		}, {
			input: cliParams{"dep", "default", false, 1, 1, 1, 1, 1, 1, "Failed"},
			want:  "0 out of 1 available pods are running.*",
		},
	}
	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/4)
		defer cancel()
		err := WaitOnDeploymentReady(ctx, getCli(tc.input), tc.input.namespace, tc.input.name)
		if tc.want != "" {
			errorchecker.AssertErrorMessage(c, err, tc.want)
		} else {
			c.Assert(err, IsNil)
		}
	}
}

func (s *WorkloadReadySuite) TestDeploymentReady(c *C) {
	cp := cliParams{"dep", "default", false, 1, 1, 1, 1, 1, 1, "Running"}
	ctx := context.Background()
	ready, status, err := DeploymentReady(ctx, getCli(cp), cp.namespace, cp.name)
	c.Assert(ready, DeepEquals, true)
	c.Assert(status, DeepEquals, "")
	c.Assert(err, IsNil)
}

// Returns a fake k8s cli that contains a Deployment, ReplicaSet or StatefulSet, and Pod
func getCli(cp cliParams) kubernetes.Interface {
	const dUID types.UID = "1234"
	const pUID types.UID = "4567"

	kubeObjects := []runtime.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:       cp.name,
				Namespace:  cp.namespace,
				UID:        dUID,
				Generation: cp.generation,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &cp.specReplicas,
			},
			Status: appsv1.DeploymentStatus{
				Replicas:           cp.statusReplicas,
				UpdatedReplicas:    cp.statusUpdatedReplicas,
				AvailableReplicas:  cp.statusAvailableReplicas,
				ObservedGeneration: cp.observedGeneration,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "pod",
				Namespace:       cp.namespace,
				OwnerReferences: []metav1.OwnerReference{{UID: pUID}},
			},
			Status: corev1.PodStatus{
				Phase: cp.podStatus,
			},
		},
	}

	var podSet runtime.Object
	if cp.stateful {
		podSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:            cp.name,
				Namespace:       cp.namespace,
				UID:             pUID,
				OwnerReferences: []metav1.OwnerReference{{UID: dUID}},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &cp.specReplicas,
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: cp.statusReplicas,
			},
		}
	} else {
		podSet = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "repset",
				Namespace:       cp.namespace,
				UID:             pUID,
				OwnerReferences: []metav1.OwnerReference{{UID: dUID}},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: &cp.specReplicas,
			},
			Status: appsv1.ReplicaSetStatus{},
		}
	}
	kubeObjects = append(kubeObjects, podSet)

	return fake.NewSimpleClientset(kubeObjects...)
}
