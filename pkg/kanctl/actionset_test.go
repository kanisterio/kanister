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
