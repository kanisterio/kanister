# Examples

* Helm examples in `helm/kanister/`

     To make it easier to experiment with Kanister, we have modified a few upstream Helm charts to add Kanister Blueprints as well as easily configure the application via Helm. More information on these is available in the [kanister docs](https://docs.kanister.io/helm.html)


  * `kanister-mongodb-replicaset/`

     Dynamically scaleable MongoDB replica set protected using mongodump
  * `kanister-mysql/`

     Single node MySQL deployment protected using mysqldump
  * `kanister-postgresql/`

    PostgreSQL deployment protected using continuous archiving of PostgreSQL WAL files and base backups.
    Supports point-in-time restore.

 * `mongo-sidecar/`

    MongoDB statefulset with a kanister sidecar. Uses monogdump.
 * `postgres-basic-pgdump/`

    Unmodified PostgreSQL deployment protected using `pg_dumpall`
 * `time-log/`

    Kanister tutorial. Demonstrates Kanister features using a simple time-logger deployment