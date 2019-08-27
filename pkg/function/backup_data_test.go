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

type BackupDataSuite struct {
}

var _ = Suite(&BackupDataSuite{})

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

func (s *BackupDataSuite) TestValidateProfile(c *C) {
	testCases := []struct {
		name       string
		profile    *param.Profile
		errChecker Checker
	}{
		{"Valid Profile", newValidProfile(), IsNil},
		{"Invalid Profile", newInvalidProfile(), NotNil},
		{"Nil Profile", nil, NotNil},
	}
	for _, tc := range testCases {
		err := validateProfile(tc.profile)
		c.Check(err, tc.errChecker, Commentf("Test %s Failed", tc.name))
	}
}
