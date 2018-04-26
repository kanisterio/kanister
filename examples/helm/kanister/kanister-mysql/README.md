# MySQL

[MySQL](https://MySQL.org) is one of the most popular database servers in the world. Notable users include Wikipedia, Facebook and Google.

## Introduction

This chart bootstraps a single node MySQL deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.6+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure

## Installing the Chart

Installing Kanister Enabled MySQL
For basic installation, you can install using a Kasten-provided Helm chart that will install an instance of MySQL (a deployment with a persistent volume) as well as a Kanister blueprint to be used with it.

Prior to install you will need to have the Kasten Helm repository added to your local setup.
```bash
$ helm repo add kasten https://charts.kasten.io/
```
Then install the sample MySQL application in its own namespace.

```bash
$ helm install kasten/kanister-mysql -n my-release --namespace mysql-test \
     --set kanister.s3_endpoint="https://my-custom-s3-provider:9000" \
     --set kanister.s3_api_key="AKIAIOSFODNN7EXAMPLE" \
     --set kanister.s3_api_secret="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
     --set kanister.s3_bucket="kanister-bucket" \
     --set mysqlRootPassword="asd#45@mysqlEXAMPLE" \
     --set persistence.size=10Gi
```

The settings in the command above represent the minimum recommended set for your installation.
The command deploys MySQL on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

By default a random password will be generated for the root user. If you'd like to set your own password change the mysqlRootPassword
in the values.yaml.

You can retrieve your root password by running the following command. Make sure to replace [YOUR_RELEASE_NAME]:

    printf $(printf '\%o' `kubectl get secret [YOUR_RELEASE_NAME]-mysql -o jsonpath="{.data.mysql-root-password[*]}"`)

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable MySQL Kanister blueprint parameters and their default values.

| Parameter | Description | Default |
| --- | --- | --- |
| `kanister.s3_api_key` | (Required) API Key for an s3 compatible object store. | `nil`|
| `kanister.s3_api_secret` | (Required) Corresponding secret for `s3_api_key`. | `nil` |
| `kanister.s3_bucket` | (Required) A bucket that will be used to store Kanister artifacts. <br><br>The bucket must already exist and the account with the above API key and secret needs to have sufficient permissions to list, get, put, delete. | `None` |
| `kanister.s3_endpoint` | (Optional) The URL for an s3 compatible object store provider. Can be omitted if provider is AWS. Required for any other provider. | `nil` |
| `kanister.s3_verify_ssl` | (Optional) Set to ``false`` to disable SSL verification on the s3 endpoint. | `true` |
| `kanister.controller_namespace` | (Optional) Specify the namespace where the Kanister controller is running. | kasten-io |


The following table lists the configurable parameters of the MySQL chart and their default values.

| Parameter                            | Description                               | Default                                              |
| ------------------------------------ | ----------------------------------------- | ---------------------------------------------------- |
| `imageTag`                           | `mysql` image tag.                        | Most recent release                                  |
| `imagePullPolicy`                    | Image pull policy                         | `IfNotPresent`                                       |
| `mysqlRootPassword`                  | Password for the `root` user.             | `nil`                                                |
| `mysqlUser`                          | Username of new user to create.           | `nil`                                                |
| `mysqlPassword`                      | Password for the new user.                | `nil`                                                |
| `mysqlDatabase`                      | Name for new database to create.          | `nil`                                                |
| `livenessProbe.initialDelaySeconds`  | Delay before liveness probe is initiated  | 30                                                   |
| `livenessProbe.periodSeconds`        | How often to perform the probe            | 10                                                   |
| `livenessProbe.timeoutSeconds`       | When the probe times out                  | 5                                                    |
| `livenessProbe.successThreshold`     | Minimum consecutive successes for the probe to be considered successful after having failed. | 1 |
| `livenessProbe.failureThreshold`     | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3 |
| `readinessProbe.initialDelaySeconds` | Delay before readiness probe is initiated | 5                                                    |
| `readinessProbe.periodSeconds`       | How often to perform the probe            | 10                                                   |
| `readinessProbe.timeoutSeconds`      | When the probe times out                  | 1                                                    |
| `readinessProbe.successThreshold`    | Minimum consecutive successes for the probe to be considered successful after having failed. | 1 |
| `readinessProbe.failureThreshold`    | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3 |
| `persistence.enabled`                | Create a volume to store data             | true                                                 |
| `persistence.size`                   | Size of persistent volume claim           | 8Gi RW                                               |
| `persistence.storageClass`           | Type of persistent volume claim           | nil (uses alpha storage class annotation)      |
| `persistence.accessMode`             | ReadWriteOnce or ReadOnly                 | ReadWriteOnce                                        |
| `persistence.existingClaim`          | Name of existing persistent volume        | `nil`                                                |
| `persistence.subPath`                | Subdirectory of the volume to mount       | `nil`                                                |
| `resources`                          | CPU/Memory resource requests/limits       | Memory: `256Mi`<br>CPU: `100m`                       |
| `configurationFiles`                 | List of mysql configuration files         | `nil`                                                |

## Using Configuration Parameters
Some of the parameters above map to the env variables defined in the [MySQL DockerHub image](https://hub.docker.com/_/mysql/).

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install --name my-release \
  --set mysqlRootPassword=secretpassword,mysqlUser=my-user,mysqlPassword=my-password,mysqlDatabase=my-database \
    kasten/kanister-mysql
```

The above command sets the MySQL `root` account password to `secretpassword`. Additionally it creates a standard database user named `my-user`, with the password `my-password`, who has access to a database named `my-database`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
$ helm install --name my-release -f values.yaml kasten/kanister-mysql
```

> **Tip**: You can use the default [values.yaml](values.yaml)

## Persistence

The [MySQL](https://hub.docker.com/_/mysql/) image stores the MySQL data and configurations at the `/var/lib/mysql` path of the container.

By default a PersistentVolumeClaim is created and mounted into that directory. In order to disable this functionality
you can change the values.yaml to disable persistence and use an emptyDir instead.

> *"An emptyDir volume is first created when a Pod is assigned to a Node, and exists as long as that Pod is running on that node. When a Pod is removed from a node for any reason, the data in the emptyDir is deleted forever."*

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
