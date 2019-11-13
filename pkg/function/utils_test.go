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
	v1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

type UtilsTestSuite struct {
}

var _ = Suite(&UtilsTestSuite{})

func (s *UtilsTestSuite) TestValidateProfile(c *C) {
	testCases := []struct {
		name       string
		profile    *param.Profile
		errChecker Checker
	}{
		{"Valid Profile", newValidProfile(), IsNil},
		{"Valid Profile with Secret Credentials", newValidProfileWithSecretCredentials(), IsNil},
		{"Invalid Profile", newInvalidProfile(), NotNil},
		{"Invalid Profile with Secret Credentials", newInvalidProfileWithSecretCredentials(), NotNil},
		{"Nil Profile", nil, NotNil},
	}
	for _, tc := range testCases {
		err := ValidateProfile(tc.profile)
		c.Check(err, tc.errChecker, Commentf("Test %s Failed", tc.name))
	}
}

func (s *UtilsTestSuite) TestFetchPodVolumes(c *C) {
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
		vols, err := FetchPodVolumes(tc.pod, tc.tp)
		c.Check(err, tc.errChecker, Commentf("Test: %s Failed!", tc.name))
		c.Check(vols, DeepEquals, tc.vols, Commentf("Test: %s Failed!", tc.name))
	}
}

func (s *UtilsTestSuite) TestResolveArtifactPrefix(c *C) {
	for _, tc := range []struct {
		prefix   string
		expected string
	}{
		{
			prefix:   "test-bucket/prefix",
			expected: "test-bucket/prefix",
		},
		{
			prefix:   "test-bucket/pre/fix",
			expected: "test-bucket/pre/fix",
		},
		{
			prefix:   "prefix",
			expected: "test-bucket/prefix",
		},
		{
			prefix:   "pre/fix",
			expected: "test-bucket/pre/fix",
		},
		{
			prefix:   "",
			expected: "test-bucket",
		},
		{
			prefix:   "test-bucket",
			expected: "test-bucket",
		},
	} {
		res := ResolveArtifactPrefix(tc.prefix, newValidProfile())
		c.Check(res, Equals, tc.expected)
	}
}

func newValidProfile() *param.Profile {
	return &param.Profile{
		Location: crv1alpha1.Location{
			Type:     crv1alpha1.LocationTypeS3Compliant,
			Bucket:   "test-bucket",
			Endpoint: "",
			Prefix:   "",
			Region:   "us-west-1",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     "test-id",
				Secret: "test-secret",
			},
		},
		SkipSSLVerify: false,
	}
}

func newValidProfileWithSecretCredentials() *param.Profile {
	return &param.Profile{
		Location: crv1alpha1.Location{
			Type:     crv1alpha1.LocationTypeS3Compliant,
			Bucket:   "test-bucket",
			Endpoint: "",
			Prefix:   "",
			Region:   "us-west-1",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeSecret,
			Secret: &v1.Secret{
				Type: v1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte("key-id"),
					secrets.AWSSecretAccessKey: []byte("access-key"),
					secrets.ConfigRole:         []byte("role"),
				},
			},
		},
	}
}

func newInvalidProfile() *param.Profile {
	return &param.Profile{
		Location: crv1alpha1.Location{
			Type:     "foo-type",
			Bucket:   "test-bucket",
			Endpoint: "",
			Prefix:   "",
			Region:   "us-west-1",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     "test-id",
				Secret: "test-secret",
			},
		},
		SkipSSLVerify: false,
	}
}

func newInvalidProfileWithSecretCredentials() *param.Profile {
	return &param.Profile{
		Location: crv1alpha1.Location{
			Type:     crv1alpha1.LocationTypeS3Compliant,
			Bucket:   "test-bucket",
			Endpoint: "",
			Prefix:   "",
			Region:   "us-west-1",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeSecret,
			Secret: &v1.Secret{
				Type: v1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte("key-id"),
					secrets.AWSSecretAccessKey: []byte("access-key"),
					secrets.ConfigRole:         []byte("role"),
					"InvalidSecretKey":         []byte("InvalidValue"),
				},
			},
		},
	}
}

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
