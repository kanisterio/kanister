// Copyright 2026 The Kanister Authors.
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

package function

import (
	"context"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/param"
)

var _ = check.Suite(&KubeTaskWithRestorePVCSuite{})

type KubeTaskWithRestorePVCSuite struct{}

func (s *KubeTaskWithRestorePVCSuite) TestParseArgsDefaultSelectorFromWorkload(c *check.C) {
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"},
	}
	args := map[string]interface{}{
		KubeTaskWithRestorePVCImageArg:   "img",
		KubeTaskWithRestorePVCCommandArg: []string{"true"},
	}
	parsed, err := (&kubeTaskWithRestorePVCFunc{}).parseArgs(tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.namespace, check.Equals, "demo")
	c.Check(parsed.mountPath, check.Equals, defaultRestorePVCMountPath)
	c.Check(parsed.storageClass, check.Equals, defaultRestorePVCStorageClass)
	c.Check(parsed.cleanupPVC, check.Equals, true)
	c.Check(parsed.pvcSelector.MatchLabels[LabelKeyStagingPVC], check.Equals, "true")
	c.Check(parsed.pvcSelector.MatchLabels[LabelKeyWorkloadName], check.Equals, "pg")
	c.Check(parsed.pvcSelector.MatchLabels[LabelKeyWorkloadNamespace], check.Equals, "demo")
}

func (s *KubeTaskWithRestorePVCSuite) TestParseArgsUserSelectorRespected(c *check.C) {
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"},
	}
	args := map[string]interface{}{
		KubeTaskWithRestorePVCImageArg:   "img",
		KubeTaskWithRestorePVCCommandArg: []string{"true"},
		KubeTaskWithRestorePVCPVCSelectorArg: metav1.LabelSelector{
			MatchLabels: map[string]string{"my/key": "my/value"},
		},
		KubeTaskWithRestorePVCCleanupPVCArg: false,
	}
	parsed, err := (&kubeTaskWithRestorePVCFunc{}).parseArgs(tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.pvcSelector.MatchLabels, check.DeepEquals, map[string]string{"my/key": "my/value"})
	c.Check(parsed.cleanupPVC, check.Equals, false)
}

func (s *KubeTaskWithRestorePVCSuite) TestParseArgsNoWorkloadAndNoSelector(c *check.C) {
	// No workload, no selector → must error rather than match every PVC.
	args := map[string]interface{}{
		KubeTaskWithRestorePVCImageArg:     "img",
		KubeTaskWithRestorePVCCommandArg:   []string{"true"},
		KubeTaskWithRestorePVCNamespaceArg: "explicit-ns",
	}
	_, err := (&kubeTaskWithRestorePVCFunc{}).parseArgs(param.TemplateParams{}, args)
	c.Assert(err, check.NotNil)
}

func (s *KubeTaskWithRestorePVCSuite) TestFindStagingPVCExactlyOneMatch(c *check.C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset(
		stagedPVC("matching", map[string]string{
			LabelKeyStagingPVC:        "true",
			LabelKeyWorkloadName:      "pg",
			LabelKeyWorkloadNamespace: "demo",
		}),
		stagedPVC("other", map[string]string{
			"unrelated": "label",
		}),
	)
	a := &kubeTaskWithRestorePVCArgs{
		namespace: "demo",
		pvcSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				LabelKeyStagingPVC:        "true",
				LabelKeyWorkloadName:      "pg",
				LabelKeyWorkloadNamespace: "demo",
			},
		},
	}
	pvc, err := (&kubeTaskWithRestorePVCFunc{}).findStagingPVC(ctx, cli, a)
	c.Assert(err, check.IsNil)
	c.Check(pvc.Name, check.Equals, "matching")
}

func (s *KubeTaskWithRestorePVCSuite) TestFindStagingPVCZeroMatches(c *check.C) {
	cli := fake.NewSimpleClientset(
		stagedPVC("p", map[string]string{"unrelated": "x"}),
	)
	a := &kubeTaskWithRestorePVCArgs{
		namespace: "demo",
		pvcSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{LabelKeyStagingPVC: "true"},
		},
	}
	_, err := (&kubeTaskWithRestorePVCFunc{}).findStagingPVC(context.Background(), cli, a)
	c.Assert(err, check.NotNil)
}

func (s *KubeTaskWithRestorePVCSuite) TestFindStagingPVCMultipleMatches(c *check.C) {
	cli := fake.NewSimpleClientset(
		stagedPVC("a", map[string]string{LabelKeyStagingPVC: "true"}),
		stagedPVC("b", map[string]string{LabelKeyStagingPVC: "true"}),
	)
	a := &kubeTaskWithRestorePVCArgs{
		namespace: "demo",
		pvcSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{LabelKeyStagingPVC: "true"},
		},
	}
	_, err := (&kubeTaskWithRestorePVCFunc{}).findStagingPVC(context.Background(), cli, a)
	c.Assert(err, check.NotNil)
}

func (s *KubeTaskWithRestorePVCSuite) TestFindStagingPVCEmptySelector(c *check.C) {
	cli := fake.NewSimpleClientset(stagedPVC("a", map[string]string{}))
	a := &kubeTaskWithRestorePVCArgs{
		namespace:   "demo",
		pvcSelector: metav1.LabelSelector{},
	}
	_, err := (&kubeTaskWithRestorePVCFunc{}).findStagingPVC(context.Background(), cli, a)
	c.Assert(err, check.NotNil, check.Commentf("empty selector must error rather than match every PVC"))
}

func (s *KubeTaskWithRestorePVCSuite) TestBuildPodOptionsReadOnlyMount(c *check.C) {
	a := &kubeTaskWithRestorePVCArgs{
		image:     "img",
		command:   []string{"true"},
		mountPath: "/restore",
		namespace: "demo",
	}
	opts := (&kubeTaskWithRestorePVCFunc{}).buildPodOptions(a, "vol")
	v, ok := opts.Volumes["vol"]
	c.Assert(ok, check.Equals, true)
	c.Check(v.ReadOnly, check.Equals, true)
	c.Check(v.MountPath, check.Equals, "/restore")
}

func (s *KubeTaskWithRestorePVCSuite) TestRequiredAndArguments(c *check.C) {
	f := &kubeTaskWithRestorePVCFunc{}
	c.Check(f.Name(), check.Equals, KubeTaskWithRestorePVCFuncName)
	required := f.RequiredArgs()
	c.Check(required, check.DeepEquals, []string{KubeTaskWithRestorePVCImageArg, KubeTaskWithRestorePVCCommandArg})
}

func stagedPVC(name string, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "demo",
			Labels:    labels,
		},
	}
}
