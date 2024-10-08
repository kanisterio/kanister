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
	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

type GCPSecretCredsSuite struct{}

var _ = check.Suite(&GCPSecretCredsSuite{})

func (s *GCPSecretCredsSuite) TestValidateRepoServerGCPCredentials(c *check.C) {
	for i, tc := range []struct {
		secret        Secret
		errChecker    check.Checker
		expectedError error
	}{
		{ // Valid GCP Secret
			secret: NewGCPLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeGCS),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					BucketKey: []byte("bucket"),
					RegionKey: []byte("region"),
				},
			}),
			errChecker: check.IsNil,
		},
		{ // Missing required field - Bucket Key
			secret: NewGCPLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeGCS),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					RegionKey: []byte("region"),
				},
			}),
			errChecker:    check.NotNil,
			expectedError: errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, BucketKey, "ns", "sec"),
		},
		{ // Empty Secret
			secret: NewGCPLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeGCS),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			}),
			errChecker:    check.NotNil,
			expectedError: errkit.Wrap(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, "ns", "sec"),
		},
		{ // Nil Secret
			secret:        NewGCPLocation(nil),
			errChecker:    check.NotNil,
			expectedError: errkit.Wrap(secerrors.ErrValidate, secerrors.NilSecretErrorMessage),
		},
	} {
		err := tc.secret.Validate()
		c.Check(err, tc.errChecker)
		if err != nil {
			c.Check(err.Error(), check.Equals, tc.expectedError.Error(), check.Commentf("test number: %d", i))
		}
	}
}
