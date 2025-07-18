// Copyright 2024 The Kanister Authors.
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

package ephemeral_test

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type EphemeralSuite struct {
	OriginalContainer  ephemeral.ApplierList[corev1.Container]
	OriginalPod        ephemeral.ApplierList[corev1.Pod]
	OriginalPodOptions ephemeral.ApplierList[kube.PodOptions]
}

var _ = check.Suite(&EphemeralSuite{})

func (s *EphemeralSuite) SetUpTest(c *check.C) {
	s.OriginalContainer = ephemeral.Container
	ephemeral.Container = ephemeral.ApplierList[corev1.Container]{}

	s.OriginalPod = ephemeral.Pod
	ephemeral.Pod = ephemeral.ApplierList[corev1.Pod]{}

	s.OriginalPodOptions = ephemeral.PodOptions
	ephemeral.PodOptions = ephemeral.ApplierList[kube.PodOptions]{}
}

func (s *EphemeralSuite) TearDownTest(c *check.C) {
	ephemeral.Container = s.OriginalContainer
	ephemeral.Pod = s.OriginalPod
	ephemeral.PodOptions = s.OriginalPodOptions
}

type TestContainerApplier struct{}

func (TestContainerApplier) Apply(*corev1.Container) error { return nil }

func (s *EphemeralSuite) TestRegisterContainerApplier(c *check.C) {
	var applier TestContainerApplier

	c.Assert(len(ephemeral.Container), check.Equals, 0)
	ephemeral.Register(applier)

	if c.Check(len(ephemeral.Container), check.Equals, 1) {
		c.Check(ephemeral.Container[0], check.Equals, applier)
	}
}

type TestPodOptionsApplier struct{}

func (TestPodOptionsApplier) Apply(*kube.PodOptions) error { return nil }

func (s *EphemeralSuite) TestRegisterPodOptionsApplier(c *check.C) {
	var applier TestPodOptionsApplier

	c.Assert(len(ephemeral.PodOptions), check.Equals, 0)
	ephemeral.Register(applier)

	if c.Check(len(ephemeral.PodOptions), check.Equals, 1) {
		c.Check(ephemeral.PodOptions[0], check.Equals, applier)
	}
}

type TestPodApplier struct{}

func (TestPodApplier) Apply(*corev1.Pod) error { return nil }

func (s *EphemeralSuite) TestRegisterPodApplier(c *check.C) {
	var applier TestPodApplier

	c.Assert(ephemeral.Pod, check.HasLen, 0)
	ephemeral.Register(applier)

	if c.Check(len(ephemeral.Pod), check.Equals, 1) {
		c.Check(ephemeral.Pod[0], check.Equals, applier)
	}
}

func (s *EphemeralSuite) TestRegisterSet(c *check.C) {
	set := ephemeral.ApplierSet{
		Container:  TestContainerApplier{},
		Pod:        TestPodApplier{},
		PodOptions: TestPodOptionsApplier{},
	}

	c.Assert(ephemeral.Container, check.HasLen, 0)
	c.Assert(ephemeral.Pod, check.HasLen, 0)
	c.Assert(ephemeral.PodOptions, check.HasLen, 0)
	ephemeral.RegisterSet(set)

	if c.Check(ephemeral.Container, check.HasLen, 1) {
		c.Check(ephemeral.Container[0], check.Equals, set.Container)
	}
	if c.Check(ephemeral.Pod, check.HasLen, 1) {
		c.Check(ephemeral.Pod[0], check.Equals, set.Pod)
	}
	if c.Check(ephemeral.PodOptions, check.HasLen, 1) {
		c.Check(ephemeral.PodOptions[0], check.Equals, set.PodOptions)
	}
}

func (s *EphemeralSuite) TestFilter(c *check.C) {
	applier := ephemeral.Filter(
		ephemeral.PodOptionsNameFilter("matches"),
		ephemeral.ApplierFunc[kube.PodOptions](func(options *kube.PodOptions) error {
			options.Image = "applied-image"
			return nil
		}),
	)

	var options kube.PodOptions

	options.Name = "nomatch"
	applier.Apply(&options)
	c.Check(options.Image, check.Equals, "")

	options.Name = "matches"
	applier.Apply(&options)
	c.Check(options.Image, check.Equals, "applied-image")
}

func (s *EphemeralSuite) TestContainerNameFilter(c *check.C) {
	filter := ephemeral.ContainerNameFilter("matches")

	c.Check(filter.Filter(&corev1.Container{Name: "matches"}), check.Equals, true)
	c.Check(filter.Filter(&corev1.Container{Name: "nomatch"}), check.Equals, false)
}

func (s *EphemeralSuite) TestPodNameFilter(c *check.C) {
	filter := ephemeral.PodNameFilter("matches")

	c.Check(filter.Filter(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "matches"}}), check.Equals, true)
	c.Check(filter.Filter(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "nomatch"}}), check.Equals, false)
}

func (s *EphemeralSuite) TestPodOptionsNameFilter(c *check.C) {
	filter := ephemeral.PodOptionsNameFilter("matches")

	c.Check(filter.Filter(&kube.PodOptions{Name: "matches"}), check.Equals, true)
	c.Check(filter.Filter(&kube.PodOptions{Name: "nomatch"}), check.Equals, false)
}
