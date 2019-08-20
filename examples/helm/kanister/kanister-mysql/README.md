# MySQL

[MySQL](https://MySQL.org) is one of the most popular database servers in the world. Notable users include Wikipedia, Facebook and Google.

## Introduction

This chart bootstraps a single node MySQL deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.6+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kanister version 0.21.0 with `profiles.cr.kanister.io` CRD installed

## Installing the Chart

Installing Kanister Enabled MySQL
For basic installation, you can install using the provided Helm chart that will install an instance of MySQL (a deployment with a persistent volume) as well as a Kanister blueprint to be used with it.

Prior to install you will need to have the Kanister Helm repository added to your local setup.

```bash
$ helm repo add kanister https://charts.kanister.io
```
You can use following command to install Kanister if not installed already

```bash
$ helm install --name kanister --namespace kasten-io kanister/kanister-operator --set image.tag=0.21.0
```

Then install the sample MySQL application in its own namespace.

```bash
$ helm install kanister/kanister-mysql -n my-release --namespace mysql-test \
    --set profile.create='true' \
    --set profile.profileName='mysql-test-profile' \
    --set profile.location.type='s3Compliant' \
    --set profile.location.bucket='kanister-bucket' \
    --set profile.location.endpoint='https://my-custom-s3-provider:9000' \
    --set profile.aws.accessKey='AKIAIOSFODNN7EXAMPLE' \
    --set profile.aws.secretKey='wJalrXUtnFEMI%K7MDENG%bPxRfiCYEXAMPLEKEY' \
    --set mysqlRootPassword='asd#45@mysqlEXAMPLE' \
    --set persistence.size=10Gi
```

The settings in the command above represent the minimum recommended set for your installation.
The command deploys MySQL on the Kubernetes cluster in the default configuration. The
[configuration](#configuration) section lists the parameters that can be configured during
installation.

The command will also configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference whether one created as part of the application install or
not. Support for creating an ActionSet as part of install is simply for convenience.
This CR can be shared between Kanister-enabled application instances so one option is to
only create as part of the first instance.

If not creating a Profile CR, it is possible to use an even simpler command.

```bash
$ helm install kanister/kanister-mysql -n my-release --namespace mysql-test \
     --set mysqlRootPassword='asd#45@mysqlEXAMPLE' \
     --set persistence.size=10Gi
```

By default a random password will be generated for the root user. If you'd like to set your own password change the mysqlRootPassword in the values.yaml.

You can retrieve your root password by running the following command:

    kubectl get secret my-release-kanister-mysql -n mysql-test -o jsonpath="{.data.mysql-root-password}" | base64 -d

> **Tip**: List all releases using `helm list`

Once MySQL is running, you can populate it with some data. Let's add a table called "pets" to a test database:

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-kanister-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

# From inside the shell, use the mysql CLI to insert some data into the test database
# Create "test" db

$ mysql --user=root --password=asd#45@mysqlEXAMPLE

mysql> CREATE DATABASE test;
Query OK, 1 row affected (0.00 sec)

mysql> USE test;
Database changed

# Create "pets" table
mysql> CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);
Query OK, 0 rows affected (0.02 sec)

# Insert row to the table
mysql> INSERT INTO pets VALUES ('Puffball','Diane','hamster','f','1999-03-30',NULL);
Query OK, 1 row affected (0.01 sec)

# View data in "pets" table
mysql> SELECT * FROM pets;
+----------+-------+---------+------+------------+-------+
| name     | owner | species | sex  | birth      | death |
+----------+-------+---------+------+------------+-------+
| Puffball | Diane | hamster | f    | 1999-03-30 | NULL  |
+----------+-------+---------+------+------------+-------+
1 row in set (0.00 sec)

```

## Protect the Application

You can now take a backup of the MySQL data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kanctl create actionset --action backup --namespace kasten-io --blueprint my-release-kanister-mysql-blueprint --deployment mysql-test/my-release-kanister-mysql --profile mysql-test/mysql-test-profile
actionset backup-rslmb created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-rslmb         1m

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-rslmb
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:
```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-kanister-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

# Drop the test database
$ mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
| test               |
+--------------------+
5 rows in set (0.00 sec)

mysql> DROP DATABASE test;
Query OK, 1 row affected (0.03 sec)

mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
+--------------------+
4 rows in set (0.00 sec)

```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-llfb8"
actionset restore-backup-62vxm-2hdsz created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-62vxm-2hdsz
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to MySQL

```bash
mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
| test               |
+--------------------+
5 rows in set (0.00 sec)

mysql> USE test;
Reading table information for completion of table and column names
You can turn off this feature to get a quicker startup with -A

Database changed
mysql> SHOW TABLES;
+----------------+
| Tables_in_test |
+----------------+
| pets           |
+----------------+
1 row in set (0.00 sec)

mysql> SELECT * FROM pets;
+----------+-------+---------+------+------------+-------+
| name     | owner | species | sex  | birth      | death |
+----------+-------+---------+------+------------+-------+
| Puffball | Diane | hamster | f    | 1999-03-30 | NULL  |
+----------+-------+---------+------+------------+-------+
1 row in set (0.00 sec)

```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.
To completely remove the release include the `--purge` flag.

## Configuration

The following table lists the configurable MySQL Kanister blueprint and Profile CR parameters and their
default values. The Profile CR parameters are passed to the profile sub-chart. 

| Parameter | Description | Default |
| --- | --- | --- |
| `profile.create` | (Optional) Specify if a Profile CR should be created as part of install. | ``false`` |
| `profile.defaultProfile` | (Optional if not creating a default Profile) Set to ``true`` to create a profile with name `default-profile` | ``false`` | 
| `profile.profileName` | (Required if not creating a default Profile) Name for the profile that is created | `nil` |
| `profile.aws.accessKey` | (Required if creating profile) API Key for an s3 compatible object store. | `nil`|
| `profile.aws.secretKey` | (Required if creating profile) Corresponding secret for `accessKey`. | `nil` |
| `profile.location.bucket` | (Required if creating profile) A bucket that will be used to store Kanister artifacts. <br><br>The bucket must already exist and the account with the above API key and secret needs to have sufficient permissions to list, get, put, delete. | `nil` |
| `profile.location.region` | (Optional if creating profile) Region to be used for the bucket. | `nil` |
| `profile.location.endpoint` | (Optional if creating profile) The URL for an s3 compatible object store provider. Can be omitted if provider is AWS. Required for any other provider. | `nil` |
| `profile.verifySSL` | (Optional if creating profile) Set to ``false`` to disable SSL verification on the s3 endpoint. | `true` |
| `kanister.controller_namespace` | (Optional) Specify the namespace where the Kanister controller is running. | kasten-io |

The following table lists the configurable parameters of the MySQL chart and their default values.

| Parameter                                    | Description                                                                                  | Default                                              |
| -------------------------------------------- | -------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| `args`                                       | Additional arguments to pass to the MySQL container.                                         | `[]`                                                 |
| `initContainer.resources`                    | initContainer resource requests/limits                                                       | Memory: `10Mi`, CPU: `10m`                           |
| `image`                                      | `mysql` image repository.                                                                    | `mysql`                                              |
| `imageTag`                                   | `mysql` image tag.                                                                           | `5.7.14`                                             |
| `busybox.image`                              | `busybox` image repository.                                                                  | `busybox`                                            |
| `busybox.tag`                                | `busybox` image tag.                                                                         | `1.29.3`                                             |
| `testFramework.image`                        | `test-framework` image repository.                                                           | `dduportal/bats`                                     |
| `testFramework.tag`                          | `test-framework` image tag.                                                                  | `0.4.0`                                              |
| `imagePullPolicy`                            | Image pull policy                                                                            | `IfNotPresent`                                       |
| `existingSecret`                             | Use Existing secret for Password details                                                     | `nil`                                                |
| `extraVolumes`                               | Additional volumes as a string to be passed to the `tpl` function                            |                                                      |
| `extraVolumeMounts`                          | Additional volumeMounts as a string to be passed to the `tpl` function                       |                                                      |
| `extraInitContainers`                        | Additional init containers as a string to be passed to the `tpl` function                    |                                                      |
| `mysqlRootPassword`                          | Password for the `root` user. Ignored if existing secret is provided                         | Random 10 characters                                 |
| `mysqlUser`                                  | Username of new user to create.                                                              | `nil`                                                |
| `mysqlPassword`                              | Password for the new user. Ignored if existing secret is provided                            | Random 10 characters                                 |
| `mysqlDatabase`                              | Name for new database to create.                                                             | `nil`                                                |
| `livenessProbe.initialDelaySeconds`          | Delay before liveness probe is initiated                                                     | 30                                                   |
| `livenessProbe.periodSeconds`                | How often to perform the probe                                                               | 10                                                   |
| `livenessProbe.timeoutSeconds`               | When the probe times out                                                                     | 5                                                    |
| `livenessProbe.successThreshold`             | Minimum consecutive successes for the probe to be considered successful after having failed. | 1                                                    |
| `livenessProbe.failureThreshold`             | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3                                                    |
| `readinessProbe.initialDelaySeconds`         | Delay before readiness probe is initiated                                                    | 5                                                    |
| `readinessProbe.periodSeconds`               | How often to perform the probe                                                               | 10                                                   |
| `readinessProbe.timeoutSeconds`              | When the probe times out                                                                     | 1                                                    |
| `readinessProbe.successThreshold`            | Minimum consecutive successes for the probe to be considered successful after having failed. | 1                                                    |
| `readinessProbe.failureThreshold`            | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3                                                    |
| `schedulerName`                              | Name of the k8s scheduler (other than default)                                               | `nil`                                                |
| `persistence.enabled`                        | Create a volume to store data                                                                | true                                                 |
| `persistence.size`                           | Size of persistent volume claim                                                              | 8Gi RW                                               |
| `persistence.storageClass`                   | Type of persistent volume claim                                                              | nil                                                  |
| `persistence.accessMode`                     | ReadWriteOnce or ReadOnly                                                                    | ReadWriteOnce                                        |
| `persistence.existingClaim`                  | Name of existing persistent volume                                                           | `nil`                                                |
| `persistence.subPath`                        | Subdirectory of the volume to mount                                                          | `nil`                                                |
| `persistence.annotations`                    | Persistent Volume annotations                                                                | {}                                                   |
| `nodeSelector`                               | Node labels for pod assignment                                                               | {}                                                   |
| `tolerations`                                | Pod taint tolerations for deployment                                                         | {}                                                   |
| `metrics.enabled`                            | Start a side-car prometheus exporter                                                         | `false`                                              |
| `metrics.image`                              | Exporter image                                                                               | `prom/mysqld-exporter`                               |
| `metrics.imageTag`                           | Exporter image                                                                               | `v0.10.0`                                            |
| `metrics.imagePullPolicy`                    | Exporter image pull policy                                                                   | `IfNotPresent`                                       |
| `metrics.resources`                          | Exporter resource requests/limit                                                             | `nil`                                                |
| `metrics.livenessProbe.initialDelaySeconds`  | Delay before metrics liveness probe is initiated                                             | 15                                                   |
| `metrics.livenessProbe.timeoutSeconds`       | When the probe times out                                                                     | 5                                                    |
| `metrics.readinessProbe.initialDelaySeconds` | Delay before metrics readiness probe is initiated                                            | 5                                                    |
| `metrics.readinessProbe.timeoutSeconds`      | When the probe times out                                                                     | 1                                                    |
| `metrics.flags`                              | Additional flags for the mysql exporter to use                                               | `[]`                                                 |
| `metrics.serviceMonitor.enabled`             | Set this to `true` to create ServiceMonitor for Prometheus operator                          | `false`                                              |
| `metrics.serviceMonitor.additionalLabels`    | Additional labels that can be used so ServiceMonitor will be discovered by Prometheus        | `{}`                                                 |
| `resources`                                  | CPU/Memory resource requests/limits                                                          | Memory: `256Mi`, CPU: `100m`                         |
| `configurationFiles`                         | List of mysql configuration files                                                            | `nil`                                                |
| `configurationFilesPath`                     | Path of mysql configuration files                                                            | `/etc/mysql/conf.d/`                                 |
| `securityContext.enabled`                    | Enable security context (mysql pod)                                                          | `false`                                              |
| `securityContext.fsGroup`                    | Group ID for the container (mysql pod)                                                       | 999                                                  |
| `securityContext.runAsUser`                  | User ID for the container (mysql pod)                                                        | 999                                                  |
| `service.annotations`                        | Kubernetes annotations for mysql                                                             | {}                                                   |
| `service.type`                               | Kubernetes service type                                                                      | ClusterIP                                            |
| `service.loadBalancerIP`                     | LoadBalancer service IP                                                                      | `""`                                                 |
| `ssl.enabled`                                | Setup and use SSL for MySQL connections                                                      | `false`                                              |
| `ssl.secret`                                 | Name of the secret containing the SSL certificates                                           | mysql-ssl-certs                                      |
| `ssl.certificates[0].name`                   | Name of the secret containing the SSL certificates                                           | `nil`                                                |
| `ssl.certificates[0].ca`                     | CA certificate                                                                               | `nil`                                                |
| `ssl.certificates[0].cert`                   | Server certificate (public key)                                                              | `nil`                                                |
| `ssl.certificates[0].key`                    | Server key (private key)                                                                     | `nil`                                                |
| `imagePullSecrets`                           | Name of Secret resource containing private registry credentials                              | `nil`                                                |
| `initializationFiles`                        | List of SQL files which are run after the container started                                  | `nil`                                                |
| `timezone`                                   | Container and mysqld timezone (TZ env)                                                       | `nil` (UTC depending on image)                       |
| `podAnnotations`                             | Map of annotations to add to the pods                                                        | `{}`                                                 |
| `podLabels`                                  | Map of labels to add to the pods                                                             | `{}`                                                 |
| `priorityClassName`                          | Set pod priorityClassName                                                                    | `{}`                                                 |
| `deploymentAnnotations`		       | Map of annotations for deployment							      | `{}`						     |

Some of the parameters above map to the env variables defined in the [MySQL DockerHub image](https://hub.docker.com/_/mysql/).

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install --name my-release \
  --set mysqlRootPassword=secretpassword,mysqlUser=my-user,mysqlPassword=my-password,mysqlDatabase=my-database \
    kanister/kanister-mysql
```

The above command sets the MySQL `root` account password to `secretpassword`. Additionally it creates a standard database user named `my-user`, with the password `my-password`, who has access to a database named `my-database`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
$ helm install --name my-release -f values.yaml kanister/kanister-mysql
```

> **Tip**: You can use the default [values.yaml](values.yaml)

## Persistence

The [MySQL](https://hub.docker.com/_/mysql/) image stores the MySQL data and configurations at the `/var/lib/mysql` path of the container.

By default a PersistentVolumeClaim is created and mounted into that directory. In order to disable this functionality
you can change the values.yaml to disable persistence and use an emptyDir instead.

> *"An emptyDir volume is first created when a Pod is assigned to a Node, and exists as long as that Pod is running on that node. When a Pod is removed from a node for any reason, the data in the emptyDir is deleted forever."*

**Notice**: You may need to increase the value of `livenessProbe.initialDelaySeconds` when enabling persistence by using PersistentVolumeClaim from PersistentVolume with varying properties. Since its IO performance has impact on the database initialization performance. The default limit for database initialization is `60` seconds (`livenessProbe.initialDelaySeconds` + `livenessProbe.periodSeconds` * `livenessProbe.failureThreshold`). Once such initialization process takes more time than this limit, kubelet will restart the database container, which will interrupt database initialization then causing persisent data in an unusable state.

## Custom MySQL configuration files

The [MySQL](https://hub.docker.com/_/mysql/) image accepts custom configuration files at the path `/etc/mysql/conf.d`. If you want to use a customized MySQL configuration, you can create your alternative configuration files by passing the file contents on the `configurationFiles` attribute. Note that according to the MySQL documentation only files ending with `.cnf` are loaded.

```yaml
configurationFiles:
  mysql.cnf: |-
    [mysqld]
    skip-host-cache
    skip-name-resolve
    sql-mode=STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION
  mysql_custom.cnf: |-
    [mysqld]
```

## MySQL initialization files

The [MySQL](https://hub.docker.com/_/mysql/) image accepts *.sh, *.sql and *.sql.gz files at the path `/docker-entrypoint-initdb.d`.
These files are being run exactly once for container initialization and ignored on following container restarts.
If you want to use initialization scripts, you can create initialization files by passing the file contents on the `initializationFiles` attribute.


```yaml
initializationFiles:
  first-db.sql: |-
    CREATE DATABASE IF NOT EXISTS first DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;
  second-db.sql: |-
    CREATE DATABASE IF NOT EXISTS second DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;
```

## SSL

This chart supports configuring MySQL to use [encrypted connections](https://dev.mysql.com/doc/refman/5.7/en/encrypted-connections.html) with TLS/SSL certificates provided by the user. This is accomplished by storing the required Certificate Authority file, the server public key certificate, and the server private key as a Kubernetes secret. The SSL options for this chart support the following use cases:

* Manage certificate secrets with helm
* Manage certificate secrets outside of helm

## Manage certificate secrets with helm

Include your certificate data in the `ssl.certificates` section. For example:

```
ssl:
  enabled: false
  secret: mysql-ssl-certs
  certificates:
  - name: mysql-ssl-certs
    ca: |-
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    cert: |-
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    key: |-
      -----BEGIN RSA PRIVATE KEY-----
      ...
      -----END RSA PRIVATE KEY-----
```

> **Note**: Make sure your certificate data has the correct formatting in the values file.

## Manage certificate secrets outside of helm

1. Ensure the certificate secret exist before installation of this chart.
2. Set the name of the certificate secret in `ssl.secret`.
3. Make sure there are no entries underneath `ssl.certificates`.

To manually create the certificate secret from local files you can execute:
```
kubectl create secret generic mysql-ssl-certs \
  --from-file=ca.pem=./ssl/certificate-authority.pem \
  --from-file=server-cert.pem=./ssl/server-public-key.pem \
  --from-file=server-key.pem=./ssl/server-private-key.pem
```
> **Note**: `ca.pem`, `server-cert.pem`, and `server-key.pem` **must** be used as the key names in this generic secret.

If you are using a certificate your configurationFiles must include the three ssl lines under [mysqld]

```
[mysqld]
    ssl-ca=/ssl/ca.pem
    ssl-cert=/ssl/server-cert.pem
    ssl-key=/ssl/server-key.pem
```
