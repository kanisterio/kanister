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

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

type GCPSecretSuite struct{}

var _ = check.Suite(&GCPSecretSuite{})

func (s *GCPSecretSuite) TestValidateGCPCredentials(c *check.C) {
	serviceAccountJSON := make([]byte, base64.StdEncoding.EncodedLen(len([]byte("service_account_json"))))
	base64.StdEncoding.Encode(serviceAccountJSON, []byte("service_account_json"))
	for i, tc := range []struct {
		secret      *corev1.Secret
		errChecker  check.Checker
		expectedErr error
	}{
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPProjectID:             []byte("key_id"),
					GCPServiceAccountJSONKey: serviceAccountJSON,
				},
			},
			errChecker:  check.IsNil,
			expectedErr: nil,
		},
		{ // Incompatible secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPProjectID:             []byte("key_id"),
					GCPServiceAccountJSONKey: serviceAccountJSON,
				},
			},
			errChecker:  check.NotNil,
			expectedErr: errkit.Wrap(secerrors.ErrValidate, secerrors.IncompatibleSecretTypeErrorMsg, GCPSecretType, "ns", "sec"),
		},
		{ // missing field - GCPServiceKey
			secret: &corev1.Secret{
				Type: corev1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPProjectID: []byte("key_id"),
				},
			},
			expectedErr: errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, GCPServiceAccountJSONKey, "ns", "sec"),
			errChecker:  check.NotNil,
		},
		{ // missing field - GCPProjectID
			secret: &corev1.Secret{
				Type: corev1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					GCPServiceAccountJSONKey: []byte("service_account_json"),
				},
			},
			expectedErr: errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, GCPProjectID, "ns", "sec"),
			errChecker:  check.NotNil,
		},
		{ // secret is Empty
			secret: &corev1.Secret{
				Type: corev1.SecretType(GCPSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			expectedErr: errkit.Wrap(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, "ns", "sec"),
			errChecker:  check.NotNil,
		},
		{ // secret is nil
			secret:      nil,
			expectedErr: errkit.Wrap(secerrors.ErrValidate, secerrors.NilSecretErrorMessage),
			errChecker:  check.NotNil,
		},
	} {
		err := ValidateGCPCredentials(tc.secret)
		if err != nil {
			c.Check(err.Error(), check.Equals, tc.expectedErr.Error(), check.Commentf("test number: %d", i))
		}
	}
}
