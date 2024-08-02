# Redis

[Redis](https://redis.io/) is the open source, in-memory data store used by millions of developers
as a database, cache, streaming engine and message broker.

## Introduction

We will be using [Redis](https://github.com/bitnami/charts/tree/main/bitnami/redis) chart that bootstraps a Redis deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.20+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)
- Docker CLI installed
- A docker image containing the required tools to back up Redis. The Dockerfile for the image can be found [here](https://raw.githubusercontent.com/kanisterio/kanister/master/docker/redis-tools/Dockerfile). To build and push the docker image to your docker registry, execute [these](#build-docker-image) steps.

### Build docker image
- Execute below commands to build and push `redis-tools` docker image to a registry.
```bash
# On your local kanister git repo
$ cd ~/kanister/docker/redis-tools
$ docker build -t <registry>/<account_name>/redis-tools:<tag_name> .
$ docker push <registry>/<account_name>/redis-tools:<tag_name>
```

## Installing the Chart

Execute the below commands to install the Redis database using the `bitnami` chart with the release name `redis`:

```bash
# Add bitnami in your local chart repository
$ helm repo add bitnami https://charts.bitnami.com/bitnami

# Update your local chart repository
$ helm repo update

# Install the Redis database
$ helm install redis bitnami/redis --namespace redis-test --create-namespace \
    --set auth.password='<redis-password>' --set volumePermissions.enabled=true
```

The command deploys a Redis instance in the `redis-test` namespace.

By default a random password will be generated for the user. For setting your own password, use the `auth.password` param as shown above.

You can retrieve your root password by running the following command. Make sure to replace [YOUR_RELEASE_NAME] and [YOUR_NAMESPACE]:

    `kubectl get secret [YOUR_RELEASE_NAME] --namespace [YOUR_NAMESPACE] -o jsonpath="{.data.redis-password}" | base64 -d`

> **Tip**: List all releases using `helm list --all-namespaces`, using Helm Version 3.

## Integrating with Kanister

If you have deployed Redis application with name other than `redis` and namespace other than `redis-test`, you need to modify the commands (backup, restore and delete) used below to use the correct release name and namespace.

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace redis-test
```

You can read more about the Profile custom Kanister resource [here](https://docs.kanister.io/architecture.html?highlight=profile#profiles).

**NOTE:**

The above command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

### Create Blueprint

Create Blueprint in the same namespace as the Kanister controller

**NOTE:**

Replace `<registry>`, `<account_name>` and `<tag_name>` for the image value in `./redis-blueprint.yaml` before running following command.

```bash
$ kubectl create -f ./redis-blueprint.yaml -n kanister
```

Once Redis is running, you can populate it with some data. Let's add a key called "name":

```bash
# Connect to Redis by running a shell inside Redis' pod
$ kubectl -n redis-test exec -it redis-master-0 -- bash

# From inside the shell, use the redis-cli to insert some data
# Replace redis-password with the password that you have set while installing Redis
$ redis-cli -a <redis-password>

# Set value for "name" key
127.0.0.1:6379> set name test-redis
OK

# Verify value is properly set
127.0.0.1:6379> get name
"test-redis"
```

## Protect the Application

You can now take a backup of the Redis data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
# Find profile name
$ kubectl get profile -n redis-test
NAME               AGE
s3-profile-75ql6   2m

# Create Actionset
# Make sure the value of profile and blueprint matches the names of profile and blueprint created above
$ kanctl create actionset --action backup --namespace kanister --blueprint redis-blueprint --statefulset redis-test/redis-master --profile redis-test/s3-profile-75ql6 --secrets redis=redis-test/redis
actionset backup-ms8wg created

# View the status of the actionset
$ kubectl --namespace kanister get actionsets.cr.kanister.io backup-ms8wg
NAME           PROGRESS   LAST TRANSITION TIME   STATE
backup-ms8wg   100.00     2022-12-30T08:26:36Z   complete
```

### Disaster strikes!

Let's say someone accidentally deleted the key using the following command:
```bash
# Connect to Redis by running a shell inside Redis' pod
$ kubectl -n redis-test exec -it redis-master-0 -- bash

# From inside the shell, use the redis-cli to insert some data
# Replace redis-password with the password that you have set while installing Redis
$ redis-cli -a <redis-password>

# Delete key from Redis
127.0.0.1:6379> get name
"test-redis"

127.0.0.1:6379> del name
(integer) 1

127.0.0.1:6379> get name
(nil)
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action restore --from backup-ms8wg
actionset restore-backup-ms8wg-2c4c7 created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io restore-backup-ms8wg-2c4c7
NAME                         PROGRESS   LAST TRANSITION TIME   STATE
restore-backup-ms8wg-2c4c7   100.00     2022-12-30T08:42:21Z   complete
```

Once the ActionSet status is set to "complete", you can verify that the data has been successfully restored to Redis.

```bash
# Connect to Redis by running a shell inside Redis' pod
$ kubectl -n redis-test exec -it redis-master-0 -- bash

# From inside the shell, use the redis-cli to insert some data
# Replace redis-password with the password that you have set while installing Redis
$ redis-cli -a <redis-password>

127.0.0.1:6379> get name
"test-redis"
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-ms8wg --namespacetargets kanister
actionset delete-backup-ms8wg-b6lz4 created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-ms8wg-b6lz4
NAME                        PROGRESS   LAST TRANSITION TIME   STATE
delete-backup-ms8wg-b6lz4   100.00     2022-12-30T08:44:40Z   complete
```


## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-ms8wg-2c4c7 -n kanister
```


## Cleanup

### Uninstalling the Chart

To uninstall/delete the `redis` deployment:

```bash
# Helm Version 3
$ helm delete redis -n redis-test
release "redis" uninstalled
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint, Profile CR and ActionSets

```bash
$ kubectl delete blueprints.cr.kanister.io redis-blueprint -n kanister
blueprint.cr.kanister.io "redis-blueprint" deleted

$ kubectl get profiles.cr.kanister.io -n redis-test
NAME               AGE
s3-profile-75ql6   23m

$ kubectl delete profiles.cr.kanister.io s3-profile-75ql6 -n redis-test
profile.cr.kanister.io "s3-profile-75ql6" deleted

$ kubectl --namespace kanister delete actionsets.cr.kanister.io backup-ms8wg delete-backup-ms8wg-b6lz4 restore-backup-ms8wg-2c4c7
actionset.cr.kanister.io "backup-ms8wg" deleted
actionset.cr.kanister.io "delete-backup-ms8wg-b6lz4" deleted
actionset.cr.kanister.io "restore-backup-ms8wg-2c4c7" deleted
```
