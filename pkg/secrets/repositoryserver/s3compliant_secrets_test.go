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

type S3CompliantSecretTestSuite struct{}

var _ = Suite(&S3CompliantSecretTestSuite{})

func (s *S3CompliantSecretTestSuite) TestValidateRepoServerS3CompliantCredentials(c *C) {
	for i, tc := range []struct {
		secret        Secret
		errChecker    Checker
		expectedError error
	}{
		{ // Valid S3 Compatible Secret
			secret: NewS3CompliantLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypes3Compliant),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					BucketKey:   []byte("bucket"),
					RegionKey:   []byte("region"),
					EndpointKey: []byte("endpoint"),
				},
			}),
			errChecker: IsNil,
		},
		{ // Missing required field - Bucket Key
			secret: NewS3CompliantLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypes3Compliant),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					RegionKey:   []byte("region"),
					EndpointKey: []byte("endpoint"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, BucketKey, "ns", "sec"),
		},
		{ // Missing required field - Region Key
			secret: NewS3CompliantLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypes3Compliant),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					BucketKey:   []byte("bucket"),
					EndpointKey: []byte("endpoint"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, RegionKey, "ns", "sec"),
		},
		{ // Missing required field - Endpoint Key
			secret: NewS3CompliantLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypes3Compliant),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					RegionKey: []byte("region"),
					BucketKey: []byte("bucket"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, EndpointKey, "ns", "sec"),
		},
		{ // Empty Secret
			secret: NewS3CompliantLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypes3Compliant),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, "ns", "sec"),
		},
		{ // Nil Secret
			secret:        NewS3CompliantLocation(nil),
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
