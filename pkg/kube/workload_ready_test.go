package kube

import (
	"context"
	"time"

	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
	observedGeneration      int64
	podStatus               v1.PodPhase
}

// These tests can be used to force the various error states
func (s *WorkloadReadySuite) TestWaitOnStatefulSetReady(c *C) {
	cp := cliParams{"ss", "default", true, 1, 1, 1, 1, 2, "Running"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second/4)
	defer cancel()
	err := WaitOnStatefulSetReady(ctx, getCli(cp), cp.namespace, cp.name)
	c.Assert(err, IsNil)
}

func (s *WorkloadReadySuite) TestStatefulSetReady(c *C) {
	cp := cliParams{"ss", "default", true, 1, 1, 1, 1, 2, "Running"}
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
			input: cliParams{"dep", "default", false, 5, 2, 5, 5, 2, "Running"},
			want:  "Specified 5 replicas and only have 2.*",
		}, {
			input: cliParams{"dep", "default", false, 5, 5, 2, 5, 2, "Running"},
			want:  "Specified 5 replicas and only have 2 updated replicas.*",
		}, {
			input: cliParams{"dep", "default", false, 5, 5, 5, 2, 2, "Running"},
			want:  "Specified 5 replicas and only have 2 available replicas.*",
		},
	}
	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/4)
		defer cancel()
		err := WaitOnDeploymentReady(ctx, getCli(tc.input), tc.input.namespace, tc.input.name)
		c.Assert(err, ErrorMatches, tc.want)
	}
}

func (s *WorkloadReadySuite) TestDeploymentReady(c *C) {
	cp := cliParams{"dep", "default", false, 1, 1, 1, 1, 2, "Running"}
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
				Name:      cp.name,
				Namespace: cp.namespace,
				UID:       dUID,
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
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "pod",
				Namespace:       cp.namespace,
				OwnerReferences: []metav1.OwnerReference{{UID: pUID}},
			},
			Status: v1.PodStatus{
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
