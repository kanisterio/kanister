# Walkthrough of PostgreSQL

This is an example of using Kanister to backup and restore PostgreSQL. In this example, we will deploy PostgreSQL using the Patroni operator.

### 1. Deploy the Application

Deploy a Postgres instance using the instructions [here](https://github.com/kubernetes/charts/tree/master/incubator/patroni):
```bash
$ helm repo add incubator https://charts.helm.sh/incubator/
$ helm dependency update
$ helm install my-release --namespace kanister --create-namespace incubator/patroni
```

### 2. Protect the Application

Next create a blueprint which describes how backup and restore actions can be executed on this application. The blueprint for this application can be found in `blueprint.yaml`.

Notice that the actions in the blueprint reference a S3 location specified in the config map in `s3-location-configmap.yaml`. In order for this example to work, you should update the path field of s3-location-configmap.yaml to point to an S3 bucket to which you have access.

```bash
# Edit the ConfigMap spec to specify an existing S3 bucket
$ vim s3-location-configmap.yaml

# Create the ConfigMap
$ kubectl apply -f s3-location-configmap.yaml --namespace kanister
configmap "postgres-s3-location" created
```

You should also update secrets.yaml to contain the necessary AWS credentials to access this bucket. Provide your AWS credentials by setting the corresponding data values for `aws_access_key_id` and `aws_secret_access_key` in secrets.yaml. These are encoded using base64.

```bash
# Get base64 encoded aws keys
$ echo -n ${AWS_ACCESS_KEY_ID} | base64
$ echo -n ${AWS_SECRET_ACCESS_KEY} | base64

# Edit the secret spec and add the base64 encoded AWS keys
$ vim secrets.yaml

# Create secret
$ kubectl apply -f secrets.yaml --namespace kanister
secret "aws-creds" created
```
Finally, create the blueprint

```bash
# Create the blueprint for MongoDB
$ kubectl apply -f blueprint.yaml --namespace kanister
blueprint "postgres-task" created
```

You can now take a backup of the Postgres instance data using an action set defining backup for this application:
```bash
$ kubectl create -f backup-actionset.yaml --namespace kanister
actionset "pg-backup-bvwpr" created

$ kubectl get actionsets.cr.kanister.io --namespace kanister
NAME                KIND
pg-backup-bvwpr   ActionSet.v1alpha1.cr.kanister.io
```

To see the status of the backup, describe the actionset as follows:
```bash
$ kubectl describe actionset pg-backup-bvwpr --namespace kanister
```

### 3. Restore the Application

To restore the missing data, we want to use the backup created in step 2. An easy way to do this is to leverage kanctl, a command-line tool that helps create action sets that depend on other action sets:

```bash
$ kanctl create actionset --action restore --from "pg-backup-bvwpr" --namespace kanister
actionset restore-pg-backup-bvwpr-shzq1 created

# View the status of the actionset
$ kubectl get actionset restore-pg-backup-bvwpr-shzq1 -oyaml --namespace kanister

apiVersion: cr.kanister.io/v1alpha1
kind: ActionSet
metadata:
  name: restore-pg-backup-bvwpr-shzq1
...
status:
  actions:
  - artifacts: {}
    blueprint: postgres-task
    name: restore
    object:
      apiVersion: ""
      kind: StatefulSet
      name: my-release-patroni
      namespace: kanister
    phases:
    - name: restoreBackup
      state: complete
  state: complete
```
