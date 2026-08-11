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
	"fmt"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
)

// ValidateCredentials returns error if secret is failed at validation.
// Currently supports following:
// - AWS typed secret with required AWS secret fields.
func ValidateCredentials(secret *corev1.Secret) error {
	if secret == nil {
		return errkit.New("Nil secret")
	}
	switch string(secret.Type) {
	case AWSSecretType:
		return ValidateAWSCredentials(secret)
	case AzureSecretType:
		return ValidateAzureCredentials(secret)
	case GCPSecretType:
		return ValidateGCPCredentials(secret)
	case FilestoreSecretType:
		// returning nil currently since we
		// dont need credentials for file store
		return nil
	default:
		return errkit.New(fmt.Sprintf("Unsupported type '%s' for secret '%s:%s'", string(secret.Type), secret.Namespace, secret.Name))
	}
}
