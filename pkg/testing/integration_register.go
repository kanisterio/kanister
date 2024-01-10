//go:build integration
// +build integration

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

package testing

import (
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/app"
)

// Register Applications to Integration Suite

// pitr-postgresql app
type PITRPostgreSQL struct {
	IntegrationSuite
}

var _ = Suite(&PITRPostgreSQL{
	IntegrationSuite{
		name:      "pitr-postgres",
		namespace: "pitr-postgres-test",
		app:       app.NewPostgresDB("pitr-postgres", ""),
		bp:        app.NewPITRBlueprint("pitr-postgres", "", true),
		profile:   newSecretProfile(),
	},
})

// postgres app
type PostgreSQL struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQL{
	IntegrationSuite{
		name:      "postgres",
		namespace: "postgres-test",
		app:       app.NewPostgresDB("postgres", ""),
		bp:        app.NewBlueprint("postgres", "", true),
		profile:   newSecretProfile(),
	},
})

// mysql app
type MySQL struct {
	IntegrationSuite
}

var _ = Suite(&MySQL{
	IntegrationSuite{
		name:      "mysql",
		namespace: "mysql-test",
		app:       app.NewMysqlDB("mysql"),
		bp:        app.NewBlueprint("mysql", "", true),
		profile:   newSecretProfile(),
	},
})

// cockroachdb app
type CockroachDB struct {
	IntegrationSuite
}

var _ = Suite(&CockroachDB{
	IntegrationSuite{
		name:      "cockroachdb",
		namespace: "cockroachdb-test",
		app:       app.NewCockroachDB("cockroachdb"),
		bp:        app.NewBlueprint("cockroachdb", "", false),
		profile:   newSecretProfile(),
	},
})

// time-log app for csi volumesnapshot
type TimeLogCSI struct {
	IntegrationSuite
}

var _ = Suite(&TimeLogCSI{
	IntegrationSuite{
		name:      "time-logger",
		namespace: "time-log",
		app:       app.NewTimeLogCSI("time-logger"),
		bp:        app.NewBlueprint("csi-snapshot", "", true),
		profile:   newSecretProfile(),
	},
})

// mariaDB app
type Maria struct {
	IntegrationSuite
}

var _ = Suite(&Maria{
	IntegrationSuite{
		name:      "mariadb",
		namespace: "mariadb-test",
		app:       app.NewMariaDB("maria"),
		bp:        app.NewBlueprint("maria", "", true),
		profile:   newSecretProfile(),
	},
})

// Elasticsearch app
type Elasticsearch struct {
	IntegrationSuite
}

var _ = Suite(&Elasticsearch{
	IntegrationSuite{
		name:      "elasticsearch",
		namespace: "es-test",
		app:       app.NewElasticsearchInstance("elasticsearch"),
		bp:        app.NewBlueprint("elasticsearch", "", true),
		profile:   newSecretProfile(),
	},
})

// MongoDB app
type MongoDB struct {
	IntegrationSuite
}

var _ = Suite(&MongoDB{
	IntegrationSuite{
		name:      "mongo",
		namespace: "mongo-test",
		app:       app.NewMongoDB("mongo"),
		bp:        app.NewBlueprint("mongo", "", true),
		profile:   newSecretProfile(),
	},
})

// Cassandra App
type Cassandra struct {
	IntegrationSuite
}

var _ = Suite(&Cassandra{IntegrationSuite{
	name:      "cassandra",
	namespace: "cassandra-test",
	app:       app.NewCassandraInstance("cassandra"),
	bp:        app.NewBlueprint("cassandra", "", true),
	profile:   newSecretProfile(),
},
})

// Couchbase app
type Couchbase struct {
	IntegrationSuite
}

var _ = Suite(&Couchbase{
	IntegrationSuite{
		name:      "couchbase",
		namespace: "couchbase-test",
		app:       app.NewCouchbaseDB("couchbase"),
		bp:        app.NewBlueprint("couchbase", "", true),
		profile:   newSecretProfile(),
	},
})

// rds-postgres app
type RDSPostgreSQL struct {
	IntegrationSuite
}

var _ = Suite(&RDSPostgreSQL{
	IntegrationSuite{
		name:      "rds-postgres",
		namespace: "rds-postgres-test",
		app:       app.NewRDSPostgresDB("rds-postgres", ""),
		bp:        app.NewBlueprint("rds-postgres", "", true),
		profile:   newSecretProfile(),
	},
})

type FoundationDB struct {
	IntegrationSuite
}

var _ = Suite(&FoundationDB{
	IntegrationSuite{
		name:      "foundationdb",
		namespace: "fdb-test",
		app:       app.NewFoundationDB("foundationdb"),
		bp:        app.NewBlueprint("foundationdb", "", true),
		profile:   newSecretProfile(),
	},
})

type RDSAuroraMySQL struct {
	IntegrationSuite
}

var _ = Suite(&RDSAuroraMySQL{
	IntegrationSuite{
		name:      "rds-aurora-mysql",
		namespace: "rds-aurora-mysql-test",
		app:       app.NewRDSAuroraMySQLDB("rds-aurora-dump", ""),
		bp:        app.NewBlueprint("rds-aurora-snap", "", true),
		profile:   newSecretProfile(),
	},
})

// rds-postgres-dump app
// Create snapshot, export data and restore from dump
type RDSPostgreSQLDump struct {
	IntegrationSuite
}

var _ = Suite(&RDSPostgreSQLDump{
	IntegrationSuite{
		name:      "rds-postgres-dump",
		namespace: "rds-postgres-dump-test",
		app:       app.NewRDSPostgresDB("rds-postgres-dump", ""),
		bp:        app.NewBlueprint("rds-postgres-dump", "", true),
		profile:   newSecretProfile(),
	},
})

// rds-postgres-snap app
// Create snapshot and restore from snapshot
type RDSPostgreSQLSnap struct {
	IntegrationSuite
}

var _ = Suite(&RDSPostgreSQLSnap{
	IntegrationSuite{
		name:      "rds-postgres-snap",
		namespace: "rds-postgres-snap-test",
		app:       app.NewRDSPostgresDB("rds-postgres-snap", ""),
		bp:        app.NewBlueprint("rds-postgres-snap", "", true),
		profile:   newSecretProfile(),
	},
})

type MSSQL struct {
	IntegrationSuite
}

var _ = Suite(&MSSQL{
	IntegrationSuite{
		name:      "mssql",
		namespace: "mssql-test",
		app:       app.NewMssqlDB("mssql"),
		bp:        app.NewBlueprint("mssql", "", true),
		profile:   newSecretProfile(),
	},
})

// OpenShift apps for version 3.11
// Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP3_11, app.EphemeralStorage, "5.7"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MongoDB deployed on openshift cluster
type MongoDBDepConfig struct {
	IntegrationSuite
}

var _ = Suite(&MongoDBDepConfig{
	IntegrationSuite{
		name:      "mongodb",
		namespace: "mongodb-test",
		app:       app.NewMongoDBDepConfig("mongodeploymentconfig", app.TemplateVersionOCP3_11, app.EphemeralStorage),
		bp:        app.NewBlueprint("mongo-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP3_11, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// OpenShift apps for version 4.4
// Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_4 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_4{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-4-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_4, app.EphemeralStorage, "5.7"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MongoDB deployed on openshift cluster
type MongoDBDepConfig4_4 struct {
	IntegrationSuite
}

var _ = Suite(&MongoDBDepConfig4_4{
	IntegrationSuite{
		name:      "mongodb",
		namespace: "mongodb4-4-test",
		app:       app.NewMongoDBDepConfig("mongodeploymentconfig", app.TemplateVersionOCP4_4, app.EphemeralStorage),
		bp:        app.NewBlueprint("mongo-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_4 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_4{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-4-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_4, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// OpenShift apps for version 4.5
// Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_5 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_5{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-5-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_5, app.EphemeralStorage, "5.7"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MongoDB deployed on openshift cluster
type MongoDBDepConfig4_5 struct {
	IntegrationSuite
}

var _ = Suite(&MongoDBDepConfig4_5{
	IntegrationSuite{
		name:      "mongodb",
		namespace: "mongodb4-5-test",
		app:       app.NewMongoDBDepConfig("mongodeploymentconfig", app.TemplateVersionOCP4_5, app.EphemeralStorage),
		bp:        app.NewBlueprint("mongo-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_5 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_5{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-5-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_5, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// Kafka deployed on kubernetes cluster
type Kafka struct {
	IntegrationSuite
}

var _ = Suite(&Kafka{
	IntegrationSuite{
		name:      "kafka",
		namespace: "kafka-test",
		app:       app.NewKafkaCluster("kafka", ""),
		bp:        app.NewBlueprint("kafka", "", true),
		profile:   newSecretProfile(),
	},
})

// Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_10 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_10{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-10-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_10, app.EphemeralStorage, "8.0"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MongoDB deployed on openshift cluster
type MongoDBDepConfig4_10 struct {
	IntegrationSuite
}

var _ = Suite(&MongoDBDepConfig4_10{
	IntegrationSuite{
		name:      "mongodb",
		namespace: "mongodb4-10-test",
		app:       app.NewMongoDBDepConfig("mongodeploymentconfig", app.TemplateVersionOCP4_10, app.EphemeralStorage),
		bp:        app.NewBlueprint("mongo-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_10 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_10{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-10-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_10, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MysqlDBDepConfig4_11 for Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_11 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_11{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-11-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_11, app.EphemeralStorage, "8.0"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQLDepConfig4_11 for PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_11 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_11{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-11-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_11, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MysqlDBDepConfig4_12 for Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_12 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_12{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-12-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_12, app.EphemeralStorage, "8.0"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQLDepConfig4_12 for PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_12 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_12{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-12-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_12, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MysqlDBDepConfig4_13 for Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_13 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_13{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-13-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_13, app.EphemeralStorage, "8.0"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQLDepConfig4_13 for PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_13 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_13{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-13-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_13, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// MysqlDBDepConfig4_14 for Mysql Instance that is deployed through DeploymentConfig on OpenShift cluster
type MysqlDBDepConfig4_14 struct {
	IntegrationSuite
}

var _ = Suite(&MysqlDBDepConfig4_14{
	IntegrationSuite{
		name:      "mysqldc",
		namespace: "mysqldc4-14-test",
		app:       app.NewMysqlDepConfig("mysqldeploymentconfig", app.TemplateVersionOCP4_14, app.EphemeralStorage, "8.0"),
		bp:        app.NewBlueprint("mysql-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})

// PostgreSQLDepConfig4_14 for PostgreSQL deployed on openshift cluster
type PostgreSQLDepConfig4_14 struct {
	IntegrationSuite
}

var _ = Suite(&PostgreSQLDepConfig4_14{
	IntegrationSuite{
		name:      "postgresdepconf",
		namespace: "postgresdepconf4-14-test",
		app:       app.NewPostgreSQLDepConfig("postgresdepconf", app.TemplateVersionOCP4_14, app.EphemeralStorage),
		bp:        app.NewBlueprint("postgres-dep-config", "", true),
		profile:   newSecretProfile(),
	},
})
