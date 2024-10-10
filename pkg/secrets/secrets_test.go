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
	"testing"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func Test(t *testing.T) { check.TestingT(t) }

type SecretUtilsSuite struct{}

var _ = check.Suite(&SecretUtilsSuite{})

func (s *SecretUtilsSuite) TestGetLocationSecret(c *check.C) {
	for i, tc := range []struct {
		secret                *corev1.Secret
		errChecker            check.Checker
		locationSecretChecker check.Checker
		expectedError         error
	}{
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeGCS),
				},
			},
			errChecker:            check.IsNil,
			locationSecretChecker: check.NotNil,
			expectedError:         nil,
		},
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeAzure),
				},
			},
			errChecker:            check.IsNil,
			locationSecretChecker: check.NotNil,
			expectedError:         nil,
		},
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeS3),
				},
			},
			errChecker:            check.IsNil,
			locationSecretChecker: check.NotNil,
			expectedError:         nil,
		},
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeFilestore),
				},
			},
			errChecker:            check.IsNil,
			locationSecretChecker: check.NotNil,
			expectedError:         nil,
		},
		{ // Missing location type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
			},
			errChecker:            check.NotNil,
			locationSecretChecker: check.IsNil,
			expectedError:         errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, repositoryserver.TypeKey, "ns", "sec"),
		},
		{ // Unsupported location type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte("invalid"),
				},
			},
			errChecker:            check.NotNil,
			locationSecretChecker: check.IsNil,
			expectedError:         errkit.Wrap(secerrors.ErrValidate, secerrors.UnsupportedLocationTypeErrorMsg, "invalid", "ns", "sec"),
		},
	} {
		rsecret, err := getLocationSecret(tc.secret)
		c.Check(err, tc.errChecker)
		c.Check(rsecret, tc.locationSecretChecker)
		if err != nil {
			c.Check(err.Error(), check.Equals, tc.expectedError.Error(), check.Commentf("test number: %d", i))
		}
	}
}
