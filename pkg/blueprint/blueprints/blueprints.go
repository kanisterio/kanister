package blueprints

import (
	"os"
	"path/filepath"
	"runtime"
)

// these paths need to be kept in-sync with the symlinks in this directory
// (the build tag 'blueprints_test' enables tests that enforce this)
var blueprintPaths = map[string]string{
	"cassandra-blueprint.yaml":           "../../../examples/stable/cassandra/cassandra-blueprint.yaml",
	"couchbase-blueprint.yaml":           "../../../examples/stable/couchbase/couchbase-blueprint.yaml",
	"elasticsearch-blueprint.yaml":       "../../../examples/stable/elasticsearch/elasticsearch-blueprint.yaml",
	"foundationdb-blueprint.yaml":        "../../../examples/stable/foundationdb/foundationdb-blueprint.yaml",
	"mongo-blueprint.yaml":               "../../../examples/stable/mongodb/mongo-blueprint.yaml",
	"mongo-dep-config-blueprint.yaml":    "../../../examples/stable/mongodb-deploymentconfig/mongo-dep-config-blueprint.yaml",
	"mysql-blueprint.yaml":               "../../../examples/stable/mysql/mysql-blueprint.yaml",
	"mysql-dep-config-blueprint.yaml":    "../../../examples/stable/mysql-deploymentconfig/mysql-dep-config-blueprint.yaml",
	"pitr-postgres-blueprint.yaml":       "../../../examples/stable/postgresql-wale/postgresql-blueprint.yaml",
	"postgres-blueprint.yaml":            "../../../examples/stable/postgresql/postgres-blueprint.yaml",
	"postgres-dep-config-blueprint.yaml": "../../../examples/stable/postgresql-deploymentconfig/postgres-dep-config-blueprint.yaml",
	"rds-postgres-dump-blueprint.yaml":   "../../../examples/aws-rds/postgresql/rds-postgres-dump-blueprint.yaml",
	"rds-postgres-snap-blueprint.yaml":   "../../../examples/aws-rds/postgresql/rds-postgres-snap-blueprint.yaml",
}

// PathFor returns the well known path for a blueprint.
//
// Note: this function is only useful when the source is available while being
//       called (i.e. if the executable is moved away from the source, this
//       function will always return an error)
func PathFor(blueprint string) (string, error) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	var blueprintPath string
	path, present := blueprintPaths[blueprint]
	if present {
		blueprintPath = filepath.Join(dir, path)
	} else {
		// can't hurt to attempt to resolve it locally
		blueprintPath = filepath.Join(dir, blueprint)
	}

	_, err := os.Stat(blueprintPath)
	if err != nil {
		return "", err
	}

	return blueprintPath, nil
}
