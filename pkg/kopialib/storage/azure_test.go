// Copyright 2024 The Kanister Authors.
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

package storage

import (
	"context"
	"fmt"

	"github.com/kanisterio/kanister/pkg/kopialib"
	"github.com/kopia/kopia/repo/blob/azure"
	. "gopkg.in/check.v1"
)

type azureStorageTestSuite struct{}

var _ = Suite(&azureStorageTestSuite{})

func (s *azureStorageTestSuite) TestSetOptions(c *C) {
	for i, tc := range []struct {
		name            string
		options         map[string]string
		expectedOptions azure.Options
		expectedErr     string
		errChecker      Checker
	}{
		{
			name:        "options not set",
			options:     map[string]string{},
			expectedErr: fmt.Sprintf(ErrStorageOptionsCannotBeNilMsg, TypeAzure),
			errChecker:  NotNil,
		},
		{
			name: "set correct options",
			options: map[string]string{
				kopialib.BucketKey:                    "test-bucket",
				kopialib.AzureStorageAccount:          "test-azure-storage-account",
				kopialib.AzureStorageAccountAccessKey: "test-azure-storage-account-access-key",
			},
			expectedOptions: azure.Options{
				Container:      "test-bucket",
				StorageAccount: "storage-account",
				StorageKey:     "storage-account-access-key",
			},
			errChecker: IsNil,
		},
	} {
		azureStorage := azureStorage{}
		err := azureStorage.SetOptions(context.Background(), tc.options)
		c.Check(err, tc.errChecker)
		if err != nil {
			c.Check(err.Error(), Equals, tc.expectedErr, Commentf("test number: %d", i))
		}

		c.Assert(azureStorage.Options, DeepEquals, tc.expectedOptions)
	}
}
