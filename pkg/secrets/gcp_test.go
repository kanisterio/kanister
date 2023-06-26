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

package secrets

import (
	"encoding/base64"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

type GCPSecretSuite struct{}

var _ = Suite(&GCPSecretSuite{})

func (s *GCPSecretSuite) TestValidateGCPCredentials(c *C) {
	var serviceKey []byte

	base64.StdEncoding.Encode(serviceKey, []byte("secret_key"))
	for i, tc := range []struct {
		secret        *v1.Secret
		expectedError error
		errChecker    Checker
	}{
		{
			secret: &v1.Secret{
				Type: v1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPProjectID:  []byte("key_id"),
					GCPServiceKey: serviceKey,
				},
			},
			expectedError: nil,
			errChecker:    IsNil,
		},
		{ // Incomatible secret type
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPProjectID:  []byte("key_id"),
					GCPServiceKey: serviceKey,
				},
			},
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.IncompatibleSecretTypeErrorMsg, GCPSecretType, "ns", "sec"),
			errChecker:    NotNil,
		},
		{ // missing field - GCPServiceKey
			secret: &v1.Secret{
				Type: v1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPProjectID: []byte("key_id"),
				},
			},
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, GCPServiceKey, "ns", "sec"),
			errChecker:    NotNil,
		},
		{ // missing field - GCPProjectID
			secret: &v1.Secret{
				Type: v1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPServiceKey: []byte("service_key"),
				},
			},
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, GCPProjectID, "ns", "sec"),
			errChecker:    NotNil,
		},
		{ // secret is Empty
			secret: &v1.Secret{
				Type: v1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, "ns", "sec"),
			errChecker:    NotNil,
		},
		{ // secret is nil
			secret:        nil,
			expectedError: errors.Wrapf(secerrors.ErrValidate, secerrors.NilSecretErrorMessage),
			errChecker:    NotNil,
		},
	} {
		err := ValidateGCPCredentials(tc.secret)
		c.Check(err, tc.errChecker)
		c.Check(err, DeepEquals, tc.expectedError, Commentf("test number: %d", i))
	}
}
