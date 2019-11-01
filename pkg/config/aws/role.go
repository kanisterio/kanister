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

package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
)

// SwitchRole func uses credentials API to automatically generates New Credentials for a given role.
func SwitchRole(ctx context.Context, accessKeyID string, secretAccessKey string, role string, duration time.Duration) (*credentials.Credentials, error) {
	creds := credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	sess, err := session.NewSession(aws.NewConfig().WithCredentials(creds))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}
	creds = stscreds.NewCredentials(sess, role, func(p *stscreds.AssumeRoleProvider) {
		p.Duration = duration
	})
	return creds, nil
}
