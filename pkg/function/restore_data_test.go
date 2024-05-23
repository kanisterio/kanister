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

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

type RestoreDataTestSuite struct{}

var _ = Suite(&RestoreDataTestSuite{})

func (s *RestoreDataTestSuite) TestValidateAndGetOptArgs(c *C) {
	testCases := []struct {
		name       string
		args       map[string]interface{}
		errChecker Checker
		tp         param.TemplateParams
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
				RestoreDataPodOverrideArg: crv1alpha1.JSONMap{
					"containers": []map[string]interface{}{
						{
							"name":    "container",
							"command": []string{"echo", "in unit tests"},
						},
					},
				},
			},
			errChecker: IsNil,
			tp: param.TemplateParams{
				PodOverride: crv1alpha1.JSONMap{
					"dnsPolicy": "ClusterFirst",
				},
			},
		},
	}
	for _, tc := range testCases {
		_, _, _, _, _, _, _, _, err := validateAndGetOptArgs(tc.args, tc.tp)
		c.Check(err, tc.errChecker, Commentf("Case %s failed", tc.name))
	}
}
