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

package function

import (
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/param"
	v1 "k8s.io/api/core/v1"
)

type RestoreDataTestSuite struct{}

var _ = Suite(&RestoreDataTestSuite{})

func newValidDeploymentTP() param.TemplateParams {
	return param.TemplateParams{
		Deployment: &param.DeploymentParams{
			Name:      "test-deployment",
			Namespace: "test-namespace",
			Pods: []string{
				"pod1",
				"pod2",
			},
			Containers: [][]string{{"test-container"}},
			PersistentVolumeClaims: map[string]map[string]string{
				"pod1": map[string]string{
					"pvc1": "path1",
				},
				"pod2": map[string]string{
					"pvc2": "path2",
				},
			},
		},
	}
}

func newInvalidDeploymentTP() param.TemplateParams {
	return param.TemplateParams{
		Deployment: &param.DeploymentParams{
			Name:      "test-deployment",
			Namespace: "test-namespace",
			Pods: []string{
				"pod1",
				"pod2",
			},
			Containers: [][]string{{"test-container"}},
			PersistentVolumeClaims: map[string]map[string]string{
				"pod1": map[string]string{
					"pvc1": "path1",
				},
			},
		},
	}
}

func newValidStatefulSetTP() param.TemplateParams {
	return param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Name:      "test-ss",
			Namespace: "test-namespace",
			Pods: []string{
				"pod1",
				"pod2",
			},
			Containers: [][]string{{"test-container"}},
			PersistentVolumeClaims: map[string]map[string]string{
				"pod1": map[string]string{
					"pvc1": "path1",
				},
				"pod2": map[string]string{
					"pvc2": "path2",
					"pvc3": "path3",
				},
			},
		},
	}
}

func (s *RestoreDataTestSuite) TestFetchPodVolumesTest(c *C) {
	testCases := []struct {
		name       string
		tp         param.TemplateParams
		pod        string
		vols       map[string]string
		errChecker Checker
	}{
		{"Valid Deployment Pod", newValidDeploymentTP(), "pod1", map[string]string{"pvc1": "path1"}, IsNil},
		{"Valid StatefulSet Pod", newValidStatefulSetTP(), "pod2", map[string]string{"pvc2": "path2", "pvc3": "path3"}, IsNil},
		{"Invalid Deployment Pod", newValidDeploymentTP(), "pod3", nil, NotNil},
		{"Invalid StatefulSet Pod", newValidStatefulSetTP(), "pod4", nil, NotNil},
		{"Deployment Pod with no volumes", newInvalidDeploymentTP(), "pod2", nil, NotNil},
		{"Invalid Template Params", param.TemplateParams{}, "pod1", nil, NotNil},
	}
	for _, tc := range testCases {
		vols, err := fetchPodVolumes(tc.pod, tc.tp)
		c.Check(err, tc.errChecker, Commentf("Test: %s Failed!", tc.name))
		c.Check(vols, DeepEquals, tc.vols, Commentf("Test: %s Failed!", tc.name))
	}
}

func (s *RestoreDataTestSuite) TestValidateAndGetOptArgs(c *C) {
	testCases := []struct {
		name       string
		args       map[string]interface{}
		errChecker Checker
	}{
		{
			name: "Args with Pod",
			args: map[string]interface{}{
				RestoreDataPodArg:       "some-pod",
				RestoreDataBackupTagArg: "backup123",
			},
			errChecker: IsNil,
		},
		{
			name: "Args with Vols",
			args: map[string]interface{}{
				RestoreDataVolsArg:      map[string]string{"pvc": "mount"},
				RestoreDataBackupTagArg: "backup123",
			},
			errChecker: IsNil,
		},
		{
			name: "Args with Pod and Vols",
			args: map[string]interface{}{
				RestoreDataPodArg:       "some-pod",
				RestoreDataVolsArg:      map[string]string{"pvc": "mount"},
				RestoreDataBackupTagArg: "backup123",
			},
			errChecker: NotNil,
		},
		{
			name:       "Empty Args",
			args:       map[string]interface{}{},
			errChecker: NotNil,
		},
		{
			name: "Args with backupTag",
			args: map[string]interface{}{
				RestoreDataPodArg:       "some-pod",
				RestoreDataBackupTagArg: "backup123",
			},
			errChecker: IsNil,
		},
		{
			name: "Args with ID",
			args: map[string]interface{}{
				RestoreDataPodArg:              "some-pod",
				RestoreDataBackupIdentifierArg: "backup123",
			},
			errChecker: IsNil,
		},
		{
			name: "Args with backupTag and ID",
			args: map[string]interface{}{
				RestoreDataPodArg:              "some-pod",
				RestoreDataBackupTagArg:        "backup123",
				RestoreDataBackupIdentifierArg: "backup123",
			},
			errChecker: NotNil,
		},
		{
			name: "Args with podOverride",
			args: map[string]interface{}{
				RestoreDataPodArg:              "some-pod",
				RestoreDataBackupIdentifierArg: "backup123",
				RestoreDataPodOverrideArg: v1.PodSpec{
					Containers: []v1.Container{
						{
							Command: []string{"echo", "in unit tests"},
						},
					},
				},
			},
			errChecker: IsNil,
		},
	}
	for _, tc := range testCases {
		_, _, _, _, _, _, _, err := validateAndGetOptArgs(tc.args)
		c.Check(err, tc.errChecker, Commentf("Case %s failed", tc.name))
	}
}
