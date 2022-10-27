// Copyright 2022 The Kanister Authors.
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
	"bytes"
	"context"
	"time"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type FileReaderSuite struct {
	cli       kubernetes.Interface
	namespace string
	pod       *v1.Pod
}

var _ = check.Suite(&FileReaderSuite{})

func (s *FileReaderSuite) SetUpSuite(c *check.C) {
	ctx := context.Background()
	var err error
	s.cli, err = NewClient()
	c.Assert(err, check.IsNil)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "exectest-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = ns.Name
	s.setupTestPod(ctx, c)
}

func (s *FileReaderSuite) TearDownSuite(c *check.C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, check.IsNil)
	}
}

func (s *FileReaderSuite) TestReadFile(c *check.C) {
	path := "/testfile"
	data := "data"
	writer := NewPodWriter(s.cli, path, bytes.NewBufferString(data))
	err := writer.Write(context.Background(), s.namespace, s.pod.Name, s.pod.Spec.Containers[0].Name)
	c.Assert(err, check.IsNil)
	reader := NewPodFileReader(s.cli, s.pod.Name, s.namespace, s.pod.Spec.Containers[0].Name)
	output, err := reader.ReadFile(context.Background(), path)
	c.Assert(err, check.IsNil)
	c.Assert(output, check.Equals, data)
}

func (s *FileReaderSuite) TestReadDir(c *check.C) {
	dir := "/mnt/secrets/locationSecret"

	reader := NewPodFileReader(s.cli, s.pod.Name, s.namespace, s.pod.Spec.Containers[0].Name)
	output, err := reader.ReadDir(context.Background(), dir)
	c.Assert(err, check.IsNil)
	c.Assert(output, check.DeepEquals, map[string]string{"prefix": "test-prefix", "type": "filestore"})
}

func (s *FileReaderSuite) setupTestPod(ctx context.Context, c *check.C) {
	ls := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "location-secret-",
		},
		StringData: map[string]string{
			"prefix": "test-prefix",
			"type":   "filestore",
		},
	}
	locSecret, err := s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), ls, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	volMounts := []v1.VolumeMount{
		{
			Name:      "location-secret",
			MountPath: "/mnt/secrets/locationSecret",
		},
	}
	vols := []v1.Volume{
		{
			Name: "location-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: locSecret.Name,
				},
			},
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "testcontainer",
					Image:           "busybox",
					Command:         []string{"sh", "-c", "tail -f /dev/null"},
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts:    volMounts,
				},
			},
			Volumes: vols,
		},
	}
	s.pod, err = s.cli.CoreV1().Pods(s.namespace).Create(ctx, pod, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	c.Assert(WaitForPodReady(ctxTimeout, s.cli, s.namespace, s.pod.Name), check.IsNil)
	s.pod, err = s.cli.CoreV1().Pods(s.namespace).Get(ctx, s.pod.Name, metav1.GetOptions{})
	c.Assert(err, check.IsNil)
}
