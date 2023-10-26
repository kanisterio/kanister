// Copyright 2020 The Kanister Authors.
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

package kanctl

import (
	"testing"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type KanctlTestSuite struct{}

var _ = Suite(&KanctlTestSuite{})

func Test(t *testing.T) { TestingT(t) }

func (k *KanctlTestSuite) TestParseGenericObjectReference(c *C) {
	for _, tc := range []struct {
		objectFlag string
		expected   crv1alpha1.ObjectReference
		err        Checker
	}{
		// not core group
		{
			objectFlag: "apps/v1/deployments/namespace/name",
			expected: crv1alpha1.ObjectReference{
				APIVersion: "v1",
				Group:      "apps/",
				Resource:   "deployments",
				Name:       "name",
				Namespace:  "namespace",
			},
			err: IsNil,
		},
		// core group
		{
			objectFlag: "v1/pods/kube-system/etcd-minikube ",
			expected: crv1alpha1.ObjectReference{
				APIVersion: "v1",
				Group:      "",
				Resource:   "pods",
				Name:       "etcd-minikube",
				Namespace:  "kube-system",
			},
			err: IsNil,
		},
		// CRs
		{
			objectFlag: "cr.kanister.io/v1alpha1/profiles/kanister/s3-profile-5fx9w",
			expected: crv1alpha1.ObjectReference{
				APIVersion: "v1alpha1",
				Group:      "cr.kanister.io/",
				Resource:   "profiles",
				Name:       "s3-profile-5fx9w",
				Namespace:  "kanister",
			},
			err: IsNil,
		},
	} {
		a, err := parseGenericObjectReference(tc.objectFlag)
		c.Check(err, tc.err)
		c.Assert(a, DeepEquals, tc.expected)
	}
}

func (k *KanctlTestSuite) TestGenerateActionSetName(c *C) {
	var testCases = []struct {
		actionName    string
		actionSetName string
		parentName    string
		expected      string
		expectedErr   error
	}{
		{actionName: "", actionSetName: "", parentName: "", expected: "", expectedErr: errMissingFieldActionName},
		{actionName: "my-action", actionSetName: "", parentName: "", expected: "my-action-"},
		{actionName: "my-action", actionSetName: "", parentName: "parent", expected: "my-action-parent-"},
		{actionName: "", actionSetName: "", parentName: "parent", expected: "parent-"},
		{actionName: "my-action", actionSetName: "my-override", parentName: "parent", expected: "my-override"},
		{actionName: "", actionSetName: "my-override", parentName: "", expected: "my-override"},
	}

	for _, tc := range testCases {
		params := &PerformParams{
			ActionName:    tc.actionName,
			ActionSetName: tc.actionSetName,
			ParentName:    tc.parentName,
		}

		actual, err := generateActionSetName(params)
		c.Assert(err, DeepEquals, tc.expectedErr)
		if tc.actionSetName != "" || tc.expected == "" {
			// if --name is provided we just use that we dont derive name
			c.Assert(actual, DeepEquals, tc.expected)
		} else {
			// random 5 chars are added at the end if name is derived by us
			c.Assert(actual[0:len(actual)-5], DeepEquals, tc.expected)
		}
	}
}

func (k *KanctlTestSuite) TestParseLabels(c *C) {
	for _, tc := range []struct {
		flagValue      string
		expectedLabels map[string]string
		expectedErr    error
	}{
		{
			flagValue:      "a=b",
			expectedLabels: map[string]string{"a": "b"},
		},
		{
			flagValue:      "a=b,c=d",
			expectedLabels: map[string]string{"a": "b", "c": "d"},
		},
		{
			flagValue:      "a=b,c=d,e=f",
			expectedLabels: map[string]string{"a": "b", "c": "d", "e": "f"},
		},
		{
			flagValue:      "a=b,c=d,",
			expectedLabels: nil,
			expectedErr:    errInvalidFieldLabels,
		},
		{
			flagValue:      ",a=b,c=d,",
			expectedLabels: nil,
			expectedErr:    errInvalidFieldLabels,
		},
		{
			flagValue:      ",a=b,c=d",
			expectedLabels: nil,
			expectedErr:    errInvalidFieldLabels,
		},
		{
			flagValue:      "a",
			expectedLabels: nil,
			expectedErr:    errInvalidFieldLabels,
		},
		{
			flagValue:      "",
			expectedLabels: nil,
		},
		{
			flagValue:      "a,=b,c=d",
			expectedLabels: nil,
			expectedErr:    errInvalidFieldLabels,
		},
		{
			flagValue:      "a=b ,c =d",
			expectedLabels: map[string]string{"a": "b", "c": "d"},
		},
		{
			flagValue:      "  a= b ,c = d ",
			expectedLabels: map[string]string{"a": "b", "c": "d"},
		},
		{
			flagValue:      "  a= b ",
			expectedLabels: map[string]string{"a": "b"},
		},
		{
			flagValue:      "a=",
			expectedLabels: map[string]string{"a": ""},
		},
	} {
		op, err := parseLabels(tc.flagValue)
		c.Assert(err, DeepEquals, tc.expectedErr)
		c.Assert(op, DeepEquals, tc.expectedLabels)
	}
}
