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

package volume

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test(t *testing.T) { TestingT(t) }

type TestVolSuite struct{}

var _ = Suite(&TestVolSuite{})

func (s *TestVolSuite) TestCreatePVC(c *C) {
	// Create PVC
	ctx := context.Background()
	pvcSize := int64(1024)
	ns := "kanister-pvc-test"
	targetVolID := "testVolID"
	annotations := map[string]string{"a1": "foo"}
	cli := fake.NewSimpleClientset()
	pvcName, err := CreatePVC(ctx, cli, ns, NoPVCNameSpecified, pvcSize, targetVolID, annotations, []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}, nil)
	c.Assert(err, IsNil)
	pvc, err := cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
	c.Assert(err, IsNil)

	c.Assert(len(pvc.Spec.AccessModes) >= 1, Equals, true)
	accessMode := pvc.Spec.AccessModes[0]
	c.Assert(accessMode, Equals, v1.ReadWriteOnce)
	capacity, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	c.Assert(ok, Equals, true)
	c.Assert(capacity.Value() >= int64(pvcSize), Equals, true)
	eq := reflect.DeepEqual(annotations, pvc.ObjectMeta.Annotations)
	c.Assert(eq, Equals, true)
	c.Assert(len(pvc.Spec.Selector.MatchLabels) >= 1, Equals, true)
	label := pvc.Spec.Selector.MatchLabels[pvMatchLabelName]
	c.Assert(label, Equals, filepath.Base(targetVolID))

	volumeMode := v1.PersistentVolumeBlock
	_, err = CreatePVC(ctx, cli, ns, "pvc2", pvcSize, targetVolID, annotations, nil, &volumeMode)
	c.Assert(err, IsNil)
	pvc2, err := cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "pvc2", metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(len(pvc2.Spec.AccessModes) >= 1, Equals, true)
	c.Assert(*pvc2.Spec.VolumeMode, Equals, v1.PersistentVolumeBlock)
}
