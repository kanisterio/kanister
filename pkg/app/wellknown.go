// Copyright 2020 The Kanister Authors.
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

package app

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// Stable Cassandra app
	CassandraApp = "cassandra"
	// Stable Couchbase app
	CouchbaseApp = "couchbase"
	// Stable Elasticsearch app
	ElasticsearchApp = "elasticsearch"
	// Stable FoundationDB app
	FoundationDBApp = "foundationdb"
	// Stable MongoDB app
	MongoDBApp = "mongo"
	// Stable MySQL app
	MySQLApp = "mysql"
	// Stable MongoDB app on OpenShift
	OpenShiftMongoDBApp = "mongo-dep-config"
	// Stable MySQL app on OpenShift
	OpenShiftMySQLApp = "mysql-dep-config"
	// Stable PostgreSQL app on OpenShift
	OpenShiftPostgreSQLApp = "postgres-dep-config"
	// Stable PITR PostgreSQL app
	PITRPostgreSQLApp = "pitr-postgres"
	// Stable PostgreSQL app
	PostgreSQLApp = "postgres"
	// RDS PostgreSQL instance
	RDSPostgreSQLApp = "rds-postgres"
	// RDS PostgreSQL instance dump
	RDSPostgreSQLDumpApp = "rds-postgres-dump"
	// RDS PostgreSQL instance snapshot
	RDSPostgreSQLSnapApp = "rds-postgres-snap"
)

var (
	ErrUnknownBlueprint = errors.New("unkown blueprint")
)

// wellknownBlueprintPaths is essentially a list of symlinks, matching those in
// the blueprints/ subdirectory.
//
// This must be maintained manually as Go modules elides symlinks from the
// downloaded module, making the actual symlinks useless in that context. Tests
// are used to enforce that this list stays in sync with the symlinks.
var wellknownBlueprintPaths = map[string]string{
	CassandraApp:           "../../../examples/stable/cassandra/cassandra-blueprint.yaml",
	CouchbaseApp:           "../../../examples/stable/couchbase/couchbase-blueprint.yaml",
	ElasticsearchApp:       "../../../examples/stable/elasticsearch/elasticsearch-blueprint.yaml",
	FoundationDBApp:        "../../../examples/stable/foundationdb/foundationdb-blueprint.yaml",
	MongoDBApp:             "../../../examples/stable/mongodb/mongo-blueprint.yaml",
	MySQLApp:               "../../../examples/stable/mysql/mysql-blueprint.yaml",
	OpenShiftMongoDBApp:    "../../../examples/stable/mongodb-deploymentconfig/mongo-dep-config-blueprint.yaml",
	OpenShiftMySQLApp:      "../../../examples/stable/mysql-deploymentconfig/mysql-dep-config-blueprint.yaml",
	OpenShiftPostgreSQLApp: "../../../examples/stable/postgresql-deploymentconfig/postgres-dep-config-blueprint.yaml",
	PITRPostgreSQLApp:      "../../../examples/stable/postgresql-wale/postgresql-blueprint.yaml",
	PostgreSQLApp:          "../../../examples/stable/postgresql/postgres-blueprint.yaml",
	RDSPostgreSQLApp:       "rds-postgres-blueprint.yaml",
	RDSPostgreSQLDumpApp:   "../../../examples/aws-rds/postgresql/rds-postgres-dump-blueprint.yaml",
	RDSPostgreSQLSnapApp:   "../../../examples/aws-rds/postgresql/rds-postgres-snap-blueprint.yaml",
}

func blueprintsPath() string {
	_, goSource, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(goSource), "..", "blueprint", "blueprints")
}

// Wellknown returns the named well-known blueprint.
//
// Note: this function is only useful when source is available. If source is
// not available, this function will always return an error.
func WellknownApp(name string) (*AppBlueprint, error) {
	symlinkPath, present := wellknownBlueprintPaths[name]
	if !present {
		return nil, ErrUnknownBlueprint
	}

	path := filepath.Join(blueprintsPath(), filepath.FromSlash(symlinkPath))
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, os.ErrNotExist
	} else if err != nil {
		return nil, err
	}

	return &AppBlueprint{
		App:  name,
		Path: path,
	}, nil
}
