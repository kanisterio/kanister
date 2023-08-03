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

type AWSSecretCredsSuite struct{}

var _ = Suite(&AWSSecretCredsSuite{})

func (s *AWSSecretCredsSuite) TestValidateRepoServerAWSCredentials(c *C) {
	for i, tc := range []struct {
		secret        Secret
		errChecker    Checker
		expectedError error
	}{
		{ // Valid AWS Secret
			secret: NewAWSLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeS3),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					BucketKey: []byte("bucket"),
					RegionKey: []byte("region"),
				},
			}),
			errChecker: IsNil,
		},
		{ // Missing required field - Region Key
			secret: NewAWSLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeS3),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					BucketKey: []byte("bucket"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, RegionKey, "ns", "sec"),
		},
		{ // Missing required field - Bucket Key
			secret: NewAWSLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeS3),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					RegionKey: []byte("region"),
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, BucketKey, "ns", "sec"),
		},
		{ // Empty Secret
			secret: NewAWSLocation(&corev1.Secret{
				Type: corev1.SecretType(LocTypeS3),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			}),
			errChecker:    NotNil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, "ns", "sec"),
		},
		{ // Nil Secret
			secret:        NewAWSLocation(nil),
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
