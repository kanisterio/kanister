# Examples

## Helm Examples

To make it easier to experiment with Kanister, we have modified a few upstream Helm charts to add Kanister Blueprints as well as easily configure the application via Helm. More information on these is available in the [kanister docs](https://docs.kanister.io/helm.html) with a brief summary available below:


* `kanister-mongodb-replicaset/`

     Dynamically scaleable MongoDB replica set protected using mongodump

* `kanister-mysql/`

     Single-node MySQL deployment protected using mysqldump.

* `kanister-postgresql/`

    PostgreSQL deployment protected using continuous archiving of PostgreSQL WAL files and base backups. Supports advanced features such as Point-In-Time-Restore (PITR).

## Non-Helm Examples

* `mongo-sidecar/`

    MongoDB statefulset with a Kanister sidecar. Uses monogdump.

* `postgres-basic-pgdump/`

    Unmodified PostgreSQL deployment (deployed via the Patroni operator) and protected using `pg_dumpall`.

* `time-log/`

    Kanister tutorial. Demonstrates Kanister features using a simple time-logger deployment.
