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

package errors

import "fmt"

var ErrValidate = fmt.Errorf("validation Failed")

const (
	// Error msg for missing required field in the secret
	MissingRequiredFieldErrorMsg = "Missing required field %s in the secret '%s:%s'"
	// Error msg for unknown in the secret
	UnknownFieldErrorMsg = "'%s:%s' secret has an unknown field"
	// Unsupported location type in the secret
	UnsupportedLocationTypeErrorMsg = "Unsupported location type '%s' for secret '%s:%s'"
	// Invalid Secret type error msg
	IncompatibleSecretTypeErrorMsg = "Incompatible secret type. Expected type %s in the secret '%s:%s'"

	// Nil Secret error message
	NilSecretErrorMessage = "Secret is Nil"
	// Empty Secret error message
	EmptySecretErrorMessage = "Empty secret. Expected at least one key value pair in the secret '%s:%s'"
)
