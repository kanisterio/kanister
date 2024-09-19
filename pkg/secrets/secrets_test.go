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

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func Test(t *testing.T) { TestingT(t) }

type SecretUtilsSuite struct{}

var _ = Suite(&SecretUtilsSuite{})

func (s *SecretUtilsSuite) TestGetLocationSecret(c *C) {
	for i, tc := range []struct {
		secret                *corev1.Secret
		errChecker            Checker
		locationSecretChecker Checker
		expectedError         error
	}{
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeGCS),
				},
			},
			errChecker:            IsNil,
			locationSecretChecker: NotNil,
			expectedError:         nil,
		},
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeAzure),
				},
			},
			errChecker:            IsNil,
			locationSecretChecker: NotNil,
			expectedError:         nil,
		},
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeS3),
				},
			},
			errChecker:            IsNil,
			locationSecretChecker: NotNil,
			expectedError:         nil,
		},
		{ // Valid secret type
			secret: &corev1.Secret{
				Type: corev1.SecretType(repositoryserver.Location),
				Data: map[string][]byte{
					repositoryserver.TypeKey: []byte(repositoryserver.LocTypeFilestore),
				},
			},
			errChecker:            IsNil,
			locationSecretChecker: NotNil,
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
			errChecker:            NotNil,
			locationSecretChecker: IsNil,
			expectedError:         errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, repositoryserver.TypeKey, "ns", "sec"),
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
			errChecker:            NotNil,
			locationSecretChecker: IsNil,
			expectedError:         errors.Wrapf(secerrors.ErrValidate, secerrors.UnsupportedLocationTypeErrorMsg, "invalid", "ns", "sec"),
		},
	} {
		rsecret, err := getLocationSecret(tc.secret)
		c.Check(err, tc.errChecker)
		c.Check(rsecret, tc.locationSecretChecker)
		if err != nil {
			c.Check(err.Error(), Equals, tc.expectedError.Error(), Commentf("test number: %d", i))
		}
	}
}
