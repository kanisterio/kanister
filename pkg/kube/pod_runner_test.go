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

package kube

import (
	"context"
	"os"
	"path"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
)

type PodRunnerTestSuite struct{}

var _ = Suite(&PodRunnerTestSuite{})

const (
	podRunnerNS = "pod-runner-test"
	podName     = "test-pod"
)

func (s *PodRunnerTestSuite) SetUpSuite(c *C) {
	os.Setenv("POD_NAMESPACE", podRunnerNS)
}

func (s *PodRunnerTestSuite) TestPodRunnerContextCanceled(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	cli := fake.NewSimpleClientset()
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p := &corev1.Pod{
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}
		return true, p, nil
	})
	deleted := make(chan struct{})
	cli.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		c.Log("Pod deleted due to Context Cancelled")
		close(deleted)
		return true, nil, nil
	})
	pr := NewPodRunner(cli, &PodOptions{
		Namespace: podRunnerNS,
		Name:      podName,
	})
	returned := make(chan struct{})
	go func() {
		_, err := pr.Run(ctx, makePodRunnerTestFunc(deleted))
		c.Assert(err, IsNil)
		close(returned)
	}()
	cancel()
	<-deleted
	<-returned
}

func (s *PodRunnerTestSuite) TestPodRunnerForSuccessCase(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	cli := fake.NewSimpleClientset()
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p := &corev1.Pod{
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}
		return true, p, nil
	})
	deleted := make(chan struct{})
	cli.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		c.Log("Pod deleted due to Context Cancelled")
		close(deleted)
		return true, nil, nil
	})
	pr := NewPodRunner(cli, &PodOptions{
		Namespace: podRunnerNS,
		Name:      podName,
		Command:   []string{"sh", "-c", "tail -f /dev/null"},
	})
	returned := make(chan struct{})
	go func() {
		_, err := pr.Run(ctx, makePodRunnerTestFunc(deleted))
		c.Assert(err, IsNil)
		close(returned)
	}()
	deleted <- struct{}{}
	<-returned
	cancel()
}

// TestPodRunnerWithJobIDDebugLabelForSuccessCase: This test adds a debug entry (kanister.io/JobID) into the context and verifies the
// pod got created with corresponding label using the entry or not.
func (s *PodRunnerTestSuite) TestPodRunnerWithJobIDDebugLabelForSuccessCase(c *C) {
	randomUUID := "xyz123"
	ctx, cancel := context.WithCancel(context.Background())
	ctx = field.Context(ctx, path.Join(consts.LabelPrefix, "JobID"), randomUUID)
	cli := fake.NewSimpleClientset()
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p := &corev1.Pod{
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}
		return true, p, nil
	})
	po := &PodOptions{
		Namespace: podRunnerNS,
		Name:      podName,
		Command:   []string{"sh", "-c", "tail -f /dev/null"},
	}
	deleted := make(chan struct{})
	cli.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		c.Log("Pod deleted due to Context Cancelled")
		close(deleted)
		return true, nil, nil
	})
	var targetKey = path.Join(consts.LabelPrefix, "JobID")
	AddLabelsToPodOptions(po, targetKey, randomUUID)
	pr := NewPodRunner(cli, po)
	errorCh := make(chan error)
	go func() {
		_, err := pr.Run(ctx, afterPodRunTestKeyPresentFunc(targetKey, randomUUID, deleted))
		errorCh <- err
	}()
	deleted <- struct{}{}
	c.Assert(<-errorCh, IsNil)
	cancel()
}

func makePodRunnerTestFunc(ch chan struct{}) func(ctx context.Context, pc PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc PodController) (map[string]interface{}, error) {
		<-ch
		return nil, nil
	}
}

func afterPodRunTestKeyPresentFunc(labelKey, labelValue string, ch chan struct{}) func(ctx context.Context, pc PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc PodController) (map[string]interface{}, error) {
		<-ch
		value, ok := pc.Pod().Labels[labelKey]
		if !ok {
			return nil, errors.New("Key not present")
		}
		if value != labelValue {
			return nil, errors.New("Value mismatch")
		}
		return nil, nil
	}
}
