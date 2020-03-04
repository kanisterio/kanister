// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package role

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
)

// Switch func uses credentials API to automatically generates New Credentials for a given role.
func Switch(ctx context.Context, creds *credentials.Credentials, role string, duration time.Duration) (*credentials.Credentials, error) {
	sess, err := session.NewSession(aws.NewConfig().WithCredentials((creds)))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	return stscreds.NewCredentials(sess, role, func(p *stscreds.AssumeRoleProvider) {
		p.Duration = duration
	}), nil
}
