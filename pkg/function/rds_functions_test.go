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
	"context"
	"fmt"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/postgres"
)

type RDSFunctionsTest struct{}

var _ = Suite(&RDSFunctionsTest{})

func (s *RDSFunctionsTest) TestPrepareCommand(c *C) {
	testCases := []struct {
		name            string
		dbEngine        RDSDBEngine
		dbList          []string
		action          RDSAction
		dbEndpoint      string
		username        string
		password        string
		backupPrefix    string
		backupID        string
		dbEngineVersion string
		errChecker      Checker
		tp              param.TemplateParams
		command         []string
	}{
		{
			name:            "PostgreS restore command",
			dbEngine:        PostgrSQLEngine,
			action:          RestoreAction,
			dbEndpoint:      "db-endpoint",
			username:        "test-user",
			password:        "secret-pass",
			backupPrefix:    "/backup/postgres-backup",
			backupID:        "backup-id",
			dbEngineVersion: "12.7",
			errChecker:      IsNil,
			dbList:          []string{"template1"},
			command: []string{"bash", "-o", "errexit", "-o", "pipefail", "-c",
				fmt.Sprintf(`
		export PGHOST=%s
		kando location pull --profile '%s' --path "%s" - | gunzip -c -f | sed 's/"LOCALE"/"LC_COLLATE"/' | psql -q -U "${PGUSER}" %s
		`, "db-endpoint", "null", fmt.Sprintf("%s/%s", "/backup/postgres-backup", "backup-id"), postgres.DefaultConnectDatabase),
			},
		},
		{
			name:            "PostgreS restore command",
			dbEngine:        PostgrSQLEngine,
			action:          RestoreAction,
			dbEndpoint:      "db-endpoint",
			username:        "test-user",
			password:        "secret-pass",
			backupPrefix:    "/backup/postgres-backup",
			backupID:        "backup-id",
			dbEngineVersion: "13.3",
			errChecker:      IsNil,
			dbList:          []string{"template1"},
			command: []string{"bash", "-o", "errexit", "-o", "pipefail", "-c",
				fmt.Sprintf(`
		export PGHOST=%s
		kando location pull --profile '%s' --path "%s" - | gunzip -c -f | psql -q -U "${PGUSER}" %s
		`, "db-endpoint", "null", fmt.Sprintf("%s/%s", "/backup/postgres-backup", "backup-id"), postgres.DefaultConnectDatabase),
			},
		},
		{
			name:            "PostgreS backup command",
			dbEngine:        PostgrSQLEngine,
			action:          BackupAction,
			dbEndpoint:      "db-endpoint",
			username:        "test-user",
			password:        "secret-pass",
			backupPrefix:    "/backup/postgres-backup",
			backupID:        "backup-id",
			dbEngineVersion: "12.7",
			errChecker:      IsNil,
			dbList:          []string{"template1"},
			command: []string{"bash", "-o", "errexit", "-o", "pipefail", "-c",
				fmt.Sprintf(`
			export PGHOST=%s
			BACKUP_PREFIX=%s
			BACKUP_ID=%s

			mkdir /backup
			dblist=( %s )
			for db in "${dblist[@]}";
			  do echo "backing up $db db" && pg_dump $db -C --inserts > /backup/$db.sql;
			done
			tar -zc backup | kando location push --profile '%s' --path "${BACKUP_PREFIX}/${BACKUP_ID}" -
			kando output %s ${BACKUP_ID}`,
					"db-endpoint", "/backup/postgres-backup", "backup-id", strings.Join([]string{"template1"}, " "), "null", ExportRDSSnapshotToLocBackupID),
			},
		},
		{
			name:            "PostgreS backup command",
			dbEngine:        "MySQLDBEngine",
			action:          BackupAction,
			dbEndpoint:      "db-endpoint",
			username:        "test-user",
			password:        "secret-pass",
			backupPrefix:    "/backup/postgres-backup",
			backupID:        "backup-id",
			dbEngineVersion: "12.7",
			errChecker:      NotNil,
			dbList:          []string{"template1"},
			command:         nil,
		},
	}

	for _, tc := range testCases {
		outCommand, err := prepareCommand(context.Background(), tc.dbEngine, tc.action, tc.dbEndpoint, tc.username, tc.password, tc.dbList, tc.backupPrefix, tc.backupID, tc.tp.Profile, tc.dbEngineVersion)

		c.Check(err, tc.errChecker, Commentf("Case %s failed", tc.name))
		c.Assert(outCommand, DeepEquals, tc.command)
	}
}
