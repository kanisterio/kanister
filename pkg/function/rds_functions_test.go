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

package function

import (
	"github.com/kanisterio/kanister/pkg/param"
	. "gopkg.in/check.v1"
)

type RDSFunctionsTest struct{}

var _ = Suite(&RDSFunctionsTest{})

func (s *RDSFunctionsTest) TestPrepareCommand(c *C) {
	testCases := []struct {
		name             string
		dbEngine         RDSDBEngine
		action           RDSAction
		instanceID       string
		dbEndpoint       string
		username         string
		password         string
		backupPrefix     string
		backupID         string
		errChecker       Checker
		deepEqualChecker Checker
		tp               param.TemplateParams
	}{
		{
			name:             "PostgreS restore command",
			dbEngine:         PostgrSQLEngine,
			action:           RestoreAction,
			instanceID:       "some-instance-id",
			dbEndpoint:       "db-endpoint",
			username:         "dummy-user",
			password:         "secret-pass",
			backupPrefix:     "/backup/postgres-backup",
			backupID:         "backup-id",
			errChecker:       IsNil,
			deepEqualChecker: DeepEquals,
		},
		{
			name:             "PostgreS backup command",
			dbEngine:         PostgrSQLEngine,
			action:           BackupAction,
			instanceID:       "some-instance-id",
			dbEndpoint:       "db-endpoint",
			username:         "dummy-user",
			password:         "secret-pass",
			backupPrefix:     "/backup/postgres-backup",
			backupID:         "backup-id",
			errChecker:       IsNil,
			deepEqualChecker: DeepEquals,
		},
	}

	for _, tc := range testCases {
		var command []string
		var err error
		if tc.dbEngine == PostgrSQLEngine {
			if tc.action == RestoreAction {
				command, err = getPostgreSQLRestoreCommand(tc.dbEndpoint, tc.password, tc.backupPrefix, tc.backupID, tc.username, tc.tp.Profile)
				c.Check(err, tc.errChecker, Commentf("Case %s failed", tc.name))
			} else if tc.action == BackupAction {
				command, err = postgresBackupCommand(tc.instanceID, tc.dbEndpoint, tc.username, tc.password, tc.backupPrefix, tc.backupID, tc.tp.Profile)
				c.Check(err, tc.errChecker, Commentf("Case %s failed", tc.name))
			}
		}

		outCommand, _, err := prepareCommand(tc.dbEngine, tc.action, tc.instanceID, tc.dbEndpoint, tc.username, tc.password, tc.backupPrefix, tc.backupID, tc.tp.Profile)
		c.Check(err, tc.errChecker, Commentf("Case %s failed", tc.name))
		c.Assert(command, tc.deepEqualChecker, outCommand)
	}
}
