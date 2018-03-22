![Kanister Logo](./graphic/graphic.png)

# Kanister

[![Run Status](https://api.shippable.com/projects/5a18e8649f19c90600633402/badge?branch=master)](https://app.shippable.com/github/kanisterio/kanister)

## Overview
Kanister is a framework that enables application-level data management on Kubernetes. It allows domain experts to capture application specific data management tasks via blueprints, which can be easily shared and extended. The framework takes care of the tedious details surrounding execution on Kubernetes and presents a homogeneous operational experience across applications at scale.

### Design Goals

The design of Kanister was driven by the following main goals:

1. **Application-Centric:** Given the increasingly complex and distributed nature of cloud-native data services, there is a growing need for data management tasks to be at the *application* level. Experts who possess domain knowledge of a specific application's needs should be able to capture these needs when performing data operations on that application.

2. **API Driven:** Data management tasks for each specific application may vary widely, and these tasks should be encapsulated by a well-defined API so as to provide a uniform data management experience. Each application expert can provide an application-specific pluggable implementation that satisfies this API, thus enabling a homogeneous data management experience of diverse and evolving data services.

3. **Extensible:** Any data management solution capable of managing a diverse set of applications must be flexible enough to capture the needs of custom data services running in a variety of environments. Such flexibility can only be provided if the solution itself can easily be extended.

### Documentation

This README provides the basic set of information to get up and running with Kanister. For further information, please refer to the [Kanister Documentation](https://docs.kanister.io).

## Quick Start

The following commands will install Kanister, Kanister-enabled MySQL and
backup to an S3 bucket.

```bash
# Install the Kanister Controller
helm install --name myrelease --namespace kanister stable/kanister-operator --set image.tag=v0.3.0

# Add Kanister charts
helm repo add kanister http://charts.kanister.io

# Install MySQL and configure its Kanister blueprint.
helm install kanister/kanister-mysql                        \
    --name mysql-release --namespace mysql-ns               \
    --set kanister.s3_bucket="mysql-backup-bucket"          \
    --set kanister.s3_api_key="${AWS_ACCESS_KEY_ID}"        \
    --set kanister.s3_api_secret="${AWS_SECRET_ACCESS_KEY}" \
    --set kanister.controller_namespace=kanister

# Perform a backup by creating an ActionSet
cat << EOF | kubectl create -f -
apiVersion: cr.kanister.io/v1alpha1
kind: ActionSet
metadata:
  generateName: mysql-backup-
  namespace: kanister
spec:
  actions:
  - name: backup
    blueprint: mysql-release-kanister-mysql-blueprint
    object:
      kind: Deployment
      name: mysql-release-kanister-mysql
      namespace: mysql-ns
EOF
```

## Getting Started

### Prerequisites

In order to use Kanister, you will first need to have the following setup:
- Kubernetes version 1.8 or higher
- kubectl
- Docker
- Helm

### Kanister Installation

Kanister is based on the operator pattern. The first step to using Kanister is to deploy the Kanister controller. The Kanister controller can be configured and installed using Helm.  See [this](https://hub.kubeapps.com/charts/stable/kanister-operator) for more information on the controller's Helm chart. Once Helm is initialized, install the controller with:

```bash
helm install --name myrelease --namespace kanister stable/kanister-operator --set image.tag=v0.3.0
```

If you wish to build and deploy the controller from source, instructions to do so can be found [here](https://docs.kanister.io/install.html#building-and-deploying-from-source).

Helm can give us the status of the objects we've installed:
```bash
$ helm status myrelease
LAST DEPLOYED: Wed Mar 21 16:40:43 2018
NAMESPACE: kanister
STATUS: DEPLOYED

RESOURCES:
==> v1/ServiceAccount
NAME                         SECRETS  AGE
myrelease-kanister-operator  1        9s

==> v1beta1/ClusterRole
NAME                                      AGE
myrelease-kanister-operator-cluster-role  9s

==> v1beta1/ClusterRoleBinding
NAME                                   AGE
myrelease-kanister-operator-edit-role  9s
myrelease-kanister-operator-cr-role    9s

==> v1beta1/Deployment
NAME                         DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
myrelease-kanister-operator  1        1        1           1          9s

==> v1/Pod(related)
NAME                                          READY  STATUS   RESTARTS  AGE
myrelease-kanister-operator-1484730505-2s279  1/1    Running  0         9s

...
```

To check the status of the controller's pod:
```bash
# Check the pod's status.
$ kubectl --namespace kanister get pod -l app=kanister-operator
NAME                                 READY     STATUS    RESTARTS   AGE
kanister-operator-2733194401-l79mg   1/1       Running   1          12m
```

The Kanister will create CRDs on startup if they don't already exist. We can verify that they exist:
```bash
$ kubectl get crd
NAME                        AGE
actionsets.cr.kanister.io   30m
blueprints.cr.kanister.io   30m
```

As shown above, two custom resources are defined - blueprints and actionsets. A blueprint specifies a set of actions that can be executed on an application. An actionset provides the necessary runtime information to trigger taking an action on the application.

Since Kanister follows the operator pattern, other useful kubectl commands work with the Kanister controller as well, such as fetching the logs:
```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

In addition to installing the Kanister controller, please also install the appropriate kanctl binary from [releases](https://github.com/kanisterio/kanister/releases).
Alternatively, you can also install kanctl by using the following command. Make sure your GOPATH is set.
```bash
$ go install -v github.com/kanisterio/kanister/cmd/kanctl
```

## Example Application: Helm-Deployed MySQL

[This](https://github.com/kanisterio/blueprint-helm-charts) git repo contains Helm charts of stateful applications from the stable chart repo, modified to include Kanister blueprints. These applications can be easily backed-up and restored.

The following commands will install MySQL and configure Kanister to backup to an S3 bucket named `mysql-backup-bucket`.

```bash
# Add Kanister charts
helm repo add kanister http://charts.kanister.io

# Install MySQL and configure its Kanister blueprint.
helm install kanister/kanister-mysql                        \
    --name mysql-release --namespace mysql-ns               \
    --set kanister.s3_bucket="mysql-backup-bucket"          \
    --set kanister.s3_api_key="${AWS_ACCESS_KEY_ID}"        \
    --set kanister.s3_api_secret="${AWS_SECRET_ACCESS_KEY}" \
    --set kanister.controller_namespace=kanister
```
To backup this application's data, we create a Kanister actionset. The command to create an actionset is included in the Helm notes, which can be displayed with `helm status mysql-release`.

```bash
$ cat << EOF | kubectl create -f -
apiVersion: cr.kanister.io/v1alpha1
kind: ActionSet
metadata:
  generateName: mysql-backup-
  namespace: kanister
spec:
  actions:
  - name: backup
    blueprint: mysql-release-kanister-mysql-blueprint
    object:
      kind: Deployment
      name: mysql-release-kanister-mysql
      namespace: mysql-ns
EOF
actionset "mysql-backup-qgx06" created
```

We can now restore this backup by chaining a restore off the actionset we just created using `kanctl`.

```bash
$ kanctl --namespace kanister perform restore --from mysql-backup-qgx06
actionset restore-mysql-backup-qgx06-bd4mq created
```

## Example Application: MongoDB

To get a more detailed overview of Kanister's components, let's walk through a non-Helm example of using Kanister to backup and restore MongoDB. In this example, we will deploy MongoDB with a sidecar container. This sidecar container will include the necessary tools to store protected data from MongoDB into an S3 bucket in AWS. Note that a sidecar container is not required to use Kanister, but is just one of several ways to access tools needed to protect the application.

### 1. Deploy the Application

Deploy the example MongoDB application using the following command:
```bash
$ kubectl apply -f ./examples/mongo-sidecar/mongo-cluster.yaml
configmap "mongo-cluster" created
service "mongo-cluster" created
statefulset "mongo-cluster" created
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "restaurants" to a test database:
```bash
# Connect to MongoDB by running a shell inside Mongo's pod
$ kubectl exec -i -t mongo-cluster-0 -- bash -l

# From inside the shell, use the mongo CLI to insert some data into the test database
$ mongo test --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
WriteResult({ "nInserted" : 1 })

# View the restaurants data in the test database
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### 2. Protect the Application

Next create a blueprint which describes how backup and restore actions can be executed on this application. The blueprint for this application can be found at `./examples/mongo-sidecar/blueprint.yaml`. Notice that the backup action of the blueprint references the S3 location specified in the config map in `./examples/mongo-sidecar/s3-location-configmap.yaml`. In order for this example to work, you should update the path field of s3-location-configmap.yaml to point to an S3 bucket to which you have access. You should also update secrets.yaml to include AWS credentials that have read/write access to the S3 bucket. Provide your AWS credentials by setting the corresponding data values for `aws_access_key_id` and `aws_secret_access_key` in secrets.yaml. These are encoded using base64.

```bash
# Get base64 encoded aws keys
$ echo "YOUR_KEY" | base64

# Create the ConfigMap with an S3 path
$ kubectl apply -f ./examples/mongo-sidecar/s3-location-configmap.yaml
configmap "mongo-s3-location" created

# Create the secrets with the AWS credentials
$ kubectl apply -f ./examples/mongo-sidecar/secrets.yaml
secrets "aws-creds" created

# Create the blueprint for MongoDB
$ kubectl apply -f ./examples/mongo-sidecar/blueprint.yaml
blueprint "mongo-sidecar" created
```

You can now take a backup of MongoDB's data using an actionset defining backup for this application. Create an actionset in the same namespace as the controller:
```bash
$ kubectl --namepsace kanister apply -f ./examples/mongo-sidecar/backup-actionset.yaml
actionset "mongo-backup-12046" created

$ kubectl --namespace kanister get actionsets.cr.kanister.io
NAME                KIND
mongo-backup-12046   ActionSet.v1alpha1.cr.kanister.io
```

### 3. Disaster strikes!

Let's say someone with fat fingers accidentally deleted the restaurants collection using the following command:
```bash
# Drop the restaurants collection
$ mongo test --quiet --eval "db.restaurants.drop()"
true
```

If you try to access this data in the database, you should see that it is no longer there:
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
# No entries should be found in the restaurants collection
```

### 4. Restore the Application

To restore the missing data, we want to use the backup created in step 2. An easy way to do this is to leverage kanctl, a command-line tool that helps create actionsets that depend on other actionsets:

```bash
$ kanctl --namespace kanister perform restore --from "mongo-backup-12046"
actionset restore-mongo-backup-12046-s1wb7 created

# View the status of the actionset
kubectl get actionset restore-mongo-backup-12046-s1wb7 -oyaml
```

You should now see that the data has been successfully restored to MongoDB!
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### 5. Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister perform delete --from "mongo-backup-12046"
actionset "delete-mongo-backup-12046-kf8mt" created

# View the status of the actionset
$ kubectl get actionset delete-mongo-backup-12046-kf8mt -oyaml
```

## Cleanup

The Kanister components can be cleaned up with the follwoing commands

```bash
$ helm delete --purge myrelease
$ kubectl delete crd {actionsets,blueprints}.cr.kanister.io
$ kubectl --namespace kanister delete actionset --all
```

## More Applications

Example applications deployed through yaml can be found in the [examples directory](https://github.com/kanisterio/kanister/tree/master/examples).

## Support
For troubleshooting help, you can email the [Kanister Google Group](https://groups.google.com/forum/#!forum/kanisterio), reach out to us on [Slack](https://kasten.typeform.com/to/QBcw8T), or file an [issue](https://github.com/kanisterio/kanister/issues).


## License
Apache License 2.0, see [LICENSE](https://github.com/kanisterio/kanister/blob/master/LICENSE).
