// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repositoryserver

import (
	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

type ServerAdminCredentialsSecretSuite struct{}

var _ = Suite(&ServerAdminCredentialsSecretSuite{})

func (s *GCPSecretCredsSuite) TestValidateRepositoryServerAdminCredentials(c *C) {
	for i, tc := range []struct {
		secret        Secret
		errChecker    Checker
		expectedError error
	}{
		{ // Valid Repository Server Admin Credentials Secret
			secret: NewRepositoryServerAdminCredentials(&corev1.Secret{
				Type: corev1.SecretType(AdminCredentialsSecret),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					AdminUsernameKey: []byte("adminusername"),
					AdminPasswordKey: []byte("adminpassword"),
				},
			}),
			errChecker: IsNil,
		},
		{ // Missing required field - AdminUsernameKey
			secret: NewRepositoryServerAdminCredentials(&corev1.Secret{
				Type: corev1.SecretType(AdminCredentialsSecret),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					AdminPasswordKey: []byte("adminpassword"),
					BucketKey:        []byte("bucketkey"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, AdminUsernameKey, "ns", "sec"),
		},
		{ // Missing required field - AdminPasswordKey
			secret: NewRepositoryServerAdminCredentials(&corev1.Secret{
				Type: corev1.SecretType(AdminCredentialsSecret),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					AdminUsernameKey: []byte("adminusername"),
					BucketKey:        []byte("bucketkey"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, AdminPasswordKey, "ns", "sec"),
		},
		{ // Secret should contain only 2 key value pairs
			secret: NewRepositoryServerAdminCredentials(&corev1.Secret{
				Type: corev1.SecretType(AdminCredentialsSecret),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					AdminUsernameKey: []byte("adminusername"),
					AdminPasswordKey: []byte("adminpassword"),
					BucketKey:        []byte("invalidkey"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.UnknownFieldErrorMsg, "ns", "sec"),
		},
		{ // Empty Secret
			secret: NewRepositoryServerAdminCredentials(&corev1.Secret{
				Type: corev1.SecretType(AdminCredentialsSecret),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, "ns", "sec"),
		},
		{ // Nil Secret
			secret:        NewRepositoryServerAdminCredentials(nil),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.NilSecretErrorMessage),
		},
	} {
		err := tc.secret.Validate()
		c.Check(err, tc.errChecker)
		if err != nil {
			c.Check(err.Error(), Equals, tc.expectedError.Error(), Commentf("test number: %d", i))
		}
	}
}
