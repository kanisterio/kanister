# MySQL

[MySQL](https://MySQL.org) is one of the most popular database servers in the world. Notable users include Wikipedia, Facebook and Google.

## Introduction

This chart bootstraps a single node MySQL deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.23+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.92.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## Installing the Chart

To install the MySQL database using the `bitnami` chart with the release name `mysql-release`:

```bash
# Add bitnami in your local chart repository
$ helm repo add bitnami https://charts.bitnami.com/bitnami

# Update your local chart repository
$ helm repo update

# Install the MySQL database

> **Note**: The custom image for MySQL is required because the Kopia Repository Server based functions (used in Blueprint), require `Kopia` binary to be available on application pod.

$ helm install mysql-release bitnami/mysql \
  --namespace mysql \
  --create-namespace \
  --set auth.rootPassword='<mysql-root-password>'
```

The command deploys a MySQL instance in the `mysql` namespace.

By default, a random password will be generated for the root user. For setting your own password, use the `auth.rootPassword` param as shown above.

You can retrieve your root password by running the following command. Make sure to replace [YOUR_RELEASE_NAME] and [YOUR_NAMESPACE]:

    `kubectl get secret [YOUR_RELEASE_NAME] --namespace [YOUR_NAMESPACE] -o jsonpath="{.data.mysql-root-password}" | base64 --decode`

> **Tip**: List all releases using `helm list --all-namespaces`, using Helm Version 3.

## Integrating with Kanister

If you have deployed MySQL application with name other than `mysql-release` and namespace other than `mysql`, you need to modify the commands(backup, restore and delete) used below to use the correct release name and namespace

### Create [Kopia Repository](https://kopia.io/docs/repositories/) using S3 as the location storage

```bash
$ kopia --log-level=error --config-file=/tmp/kopia-repository.config \
--log-dir=/tmp/kopia-cache repository create --no-check-for-updates \
--cache-directory=/tmp/cache.dir --content-cache-size-mb=0 \
--metadata-cache-size-mb=500 --override-hostname=mysql.app \
--override-username=kanisterAdmin s3 --bucket=<s3_bucket_name> \
--prefix=/repo-controller/ --region=<s3_bucket_region> \
--access-key=<aws_access_key> --secret-access-key=<aws_secret_access_key> --password=<repository_password>
```

### Create Repository Server CR

**NOTE:**

All the secrets mentioned below are required to be created in the same namespace as the Kanister controller.
And all the secrets would be used while creating the Repository Server CR. 

- Generate TLS Certificates and create TLS secret for Kopia Repository Server for secure communication between Kopia Repository Server and Client

```bash
$ openssl req -newkey rsa:2048 -nodes -keyout key.pem -x509 -days 365 -out certificate.pem

$ kubectl create secret tls repository-server-tls-cert --cert=certificate.pem --key=key.pem -n kanister
```

- Create Location Secrets for Kopia Repository

```bash
# The following file s3_location_creds.yaml is a sample file for creating s3 credentials secrets. It contains the credentials for accessing the s3 bucket.
$ vi s3_location_creds.yaml

apiVersion: v1
kind: Secret
metadata:
   name: s3-creds
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: secrets.kanister.io/aws
data:
   # required: base64 encoded value for key with proper permissions for the bucket
   aws_access_key_id: <base64_encoded_aws_access_key>
   # required: base64 encoded value for the secret corresponding to the key above
   aws_secret_access_key: <base64_encoded_aws_secret_access_key>
```

```
$ kubectl create -f s3_location_creds.yaml -n kanister
```

```bash
# The following file s3_location.yaml is a sample file for creating s3 location secrets. It contains the details of the s3 bucket.
$ vi s3_location.yaml

apiVersion: v1
kind: Secret
metadata:
   name: s3-location
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   # required: specify the type of the store
   # supported values are s3, gcs, azure, and file-store
   type: czM=
   bucket: <base64_encoded_s3_bucket_name>
   # optional: used as a sub path in the bucket for all backups
   path: <base_64_encoded_prefix_provided_when_creating_kopia_repository>
   # required, if supported by the provider
   region: <base64_encoded_s3_bucket_region>
   # optional: if set to true, do not verify SSL cert.
   # Default, when omitted, is false
   #skipSSLVerify: false
   # required: if type is `file-store`
   # optional, otherwise
   #claimName: store-pvc
```

```
$ kubectl create -f s3_location.yaml -n kanister
```

- Apply Secrets for Kopia Repository Server User Access, Admin Access and Repository Access

```bash
# The following command creates secret for kopia repository server user access.
kubectl create secret generic repository-server-user-access -n kanister --from-literal=localhost=<suitable_password_for_repository_server_user>

# The following command creates secret for kopia repository server admin access.
kubectl create secret generic repository-admin-user -n kanister --from-literal=username=<suitable_admin_username_for_repository_server> --from-literal=password=<suitable_password_for_repository_server_admin>

# The following command creates secret for kopia repository access.
kubectl create secret generic repo-pass -n kanister --from-literal=repo-password=<repository_password_set_while_creating_kopia_repository>
```

- Create Repository Server CR

```bash
vi repo-server-cr.yaml 
```
```
apiVersion: cr.kanister.io/v1alpha1
kind: RepositoryServer
metadata:
  labels:
    app.kubernetes.io/name: repositoryserver
    app.kubernetes.io/instance: repositoryserver-sample
    app.kubernetes.io/part-of: kanister
    app.kuberentes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kanister
  name: kopia-repo-server-1
  namespace: kanister
spec:
  storage:
    secretRef:
      name: s3-location
      namespace: kanister
    credentialSecretRef:
      name: s3-creds
      namespace: kanister
  repository:
    rootPath: /repo-controller/
    passwordSecretRef:
      name: repo-pass
      namespace: kanister
    username: kanisterAdmin
    hostname: mysql.app
  server:
    adminSecretRef:
      name: repository-admin-user
      namespace: kanister
    tlsSecretRef:
      name: repository-server-tls-cert
      namespace: kanister
    userAccess:
      userAccessSecretRef:
        name: repository-server-user-access
        namespace: kanister
      username: kanisteruser
```
```bash
$ kubectl create -f repo-server-cr.yaml -n kanister
```
**NOTE:**

Make Sure the Repository Server is in ServerReady State before creating actionsets.
You could check the status of the Repository Server CR by running following command
```bash
$ kubectl get repositoryservers.cr.kanister.io kopia-repo-server-1 -n kanister -o yaml
```

**NOTE:**

The above command will configure a kopia repository server, which manages artifacts resulting from Kanister
data operations such as backup.
This is stored as a `repositoryservers.cr.kanister.io` *CustomResource (CR)* which is then referenced in Kanister ActionSets.

**NOTE:**

Do not delete the secrets that were created above in order to ensure the proper functioning of the repository server
and to avoid any errors while executing actionsets.

### Create Blueprint

Create Blueprint in the same namespace as the Kanister controller

```bash
$ kubectl create -f ./mysql-blueprint.yaml -n kanister
```

Once MySQL is running, you can populate it with some data. Let's add a table called "pets" to a test database:

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql -- bash

# From inside the shell, use the mysql CLI to insert some data into the test database
# Create "test" db

# Replace mysql-root-password with the password that you have set while installing MySQL
$ mysql --user=root --password=<mysql-root-password>

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
# Find Repository name
$ kubectl get repositoryservers -n kanister
NAME                  AGE
kopia-repo-server-1   2m

# Create Actionset
# Please make sure the value of repository-server and blueprint matches with the names of repository-server and blueprint that we have created already
kanctl create actionset --action backup --namespace kanister --blueprint mysql-blueprint --statefulset mysql/mysql-release --secrets mysql=mysql/mysql-release --repository-server=kanister/kopia-repo-server-1
actionset backup-rslmb created

# View the status of the actionset
$ kubectl --namespace kanister get actionsets.cr.kanister.io backup-rslmb
NAME           PROGRESS   LAST TRANSITION TIME   STATE
backup-rslmb   100.00     2022-12-15T09:56:49Z   complete
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql -- bash

$ mysql --user=root --password=<mysql-root-password>

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
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action restore --from "backup-rslmb"
Warning: Neither --profile nor --repository-server flag is provided.
Action might fail if blueprint is using these resources.
actionset restore-backup-rslmb-2hdsz created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io restore-backup-rslmb-2hdsz
NAME                         PROGRESS   LAST TRANSITION TIME   STATE
restore-backup-rslmb-2hdsz   100.00     2022-12-15T10:00:05Z   complete
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to MySQL

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql -- bash

mysql --user=root --password=<mysql-root-password>

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

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-rslmb
Warning: Neither --profile nor --repository-server flag is provided.
Action might fail if blueprint is using these resources.
actionset delete-backup-rslmb-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-rslmb-cq6bw
NAME                        PROGRESS   LAST TRANSITION TIME   STATE
delete-backup-rslmb-cq6bw   100.00     2022-12-15T10:05:38Z   complete
```


## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-rslmb-2hdsz -n kanister
```


## Cleanup

### Uninstalling the Chart

To uninstall/delete the `mysql-release` deployment:

```bash
# Helm Version 3
$ helm delete mysql-release -n mysql
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint, RepositoryServer CR and ActionSets

```bash
$ kubectl delete blueprints.cr.kanister.io mysql-blueprint -n kanister

$ kubectl get repositoryservers -n kanister
NAME                  AGE
kopia-repo-server-1   122m

$ kubectl delete repositoryservers kopia-repo-server-1 -n kanister

$ kubectl --namespace kanister delete actionsets.cr.kanister.io backup-rslmb restore-backup-rslmb-2hdsz delete-backup-rslmb-cq6bw
```
