![Kanister Logo](./graphic/graphic.png)

# Kanister

[![Run Status](https://api.shippable.com/projects/5a18e8649f19c90600633402/badge?branch=master)](https://app.shippable.com/github/kanisterio/kanister)

## Overview
Kanister is a framework that enables application-level data management on Kubernetes. It allows domain experts to capture application specific data management tasks via blueprints, which can be easily shared and extended. The framework takes care of the tedious details surrounding execution on Kubernetes and presents a homogeneous operational experience across applications at scale.

### Design Goals

The design of Kanister was driven by the following main goals:

1. **Application-Centric:** Given the increasingly complex and distributed nature of cloud-native data services, there is a growing need for data management tasks to be at the *application* level. Experts who possess domain knowledge of a specific application's needs should be able to capture these needs when performing data operations on that application.

2. **API Driven:** Data management tasks for each specific application may vary widely, and these tasks should be encapsulated by a well-defined API so as to provide a uniform data management experience. Each application expert can provide an application-specific pluggable implementation that satisifes this API, thus enabling a homogenous data management experience of diverse and evolving data services.

3. **Extensible:** Any data management solution capable of managing a diverse set of applications must be flexible enough to capture the needs of custom data services running in a variety of environments. Such flexibility can only be provided if the solution itself can easily be extended.

### Documentation

This README provides the basic set of information to get up and running with Kanister. For further information, please refer to the [Kanister Documentation](https://docs.kanister.io).

## Getting Started

### Prerequisites

In order to use Kanister, you will first need to have the following setup:
- Kubernetes version 1.7 or higher
- kubectl
- docker

### Kanister Installation

Kanister is based on the operator pattern. The first step to using Kanister is to deploy the Kanister controller:

```bash
$ git clone git@github.com:kanisterio/kanister.git

# Install Kanister operator controller
$ kubectl apply -f bundle.yaml
```

This will install a controller in the default namespace. If you wish to build and deploy the controller from source, instructions to do so can be found [here](https://docs.kanister.io/install.html#building-and-deploying-from-source).

Let's check to make sure the controller is running with the following commands:
```bash
# Wait for the pod status to be Running
$ kubectl get pod -l app=kanister-operator
NAME                                 READY     STATUS    RESTARTS   AGE
kanister-operator-2733194401-l79mg   1/1       Running   1          12m

# Look at the CRDs
$ kubectl get crd
NAME                        AGE
actionsets.cr.kanister.io   30m
blueprints.cr.kanister.io   30m
```

As shown above, two custom resources are defined - blueprints and action sets. A blueprint specifies a set of actions that can be executed on an application. An action set provides the necessary runtime information to trigger taking an action on the application.

Since Kanister follows the operator pattern, other useful kubectl commands work with the Kanister controller as well, such as fetching the logs:
```bash
$ kubectl logs -l app=kanister-operator
```

In addition to installing the Kanister controller, please also install the appropriate kanctl binary from [releases](https://github.com/kanisterio/kanister/releases).
Alternatively, you can also install kanctl by using the following command. Make sure your GOPATH is set.
```bash
go install -v github.com/kanisterio/kanister/cmd/kanctl
```


## Walkthrough of an Example Application - MongoDB

Let's walk through an example of using Kanister to backup and restore MongoDB. In this example, we will deploy MongoDB with a sidecar container. This sidecar container will include the necessary tools to store protected data from MongoDB into an S3 bucket in AWS. Note that a sidecar container is not required to use Kanister, but rather is just one of several ways to access tools needed to protect the application (see Additional Example Applications below for alternative ways).

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

Next create a blueprint which describes how backup and restore actions can be executed on this application. The blueprint for this application can be found at `./examples/mongo-sidecar/blueprint.yaml`. Notice that the backup action of the blueprint references the S3 location specified in the config map in `./examples/mongo-sidecar/s3-location-configmap.yaml`. In order for this example to work, you should update the path field of s3-location-configmap.yaml to point to an S3 bucket to which you have access. You should also update secrets.yaml to include AWS credentials that have read/write access to the S3 bucket. In secrets.yaml, you have to provide your aws_access_key_id and aws_secret_access_key which must be base64 encoded.

```bash
#Get base64 encoded aws keys
echo "YOUR_KEY" | base64

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

You can now take a backup of MongoDB's data using an action set defining backup for this application:
```bash
$ kubectl create -f ./examples/mongo-sidecar/backup-actionset.yaml
actionset "mongo-backup-12046" created

$ kubectl get actionsets.cr.kanister.io
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

To restore the missing data, we want to use the backup created in step 2. An easy way to do this is to leverage kanctl, a command-line tool that helps create action sets that depend on other action sets:

```bash
$ kanctl perform from restore "mongo-backup-12046"

# View the status of the actionset
kubectl get actionset restore-mongo-backup-12046-s1wb7 -oyaml

```

You should now see that the data has been successfully restored to MongoDB!
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

## Cleanup

The Kanister controller and CRDs can easily be cleaned up with the following commands:

```bash
$ kubectl delete -f bundle.yaml
$ kubectl delete crd {actionsets,blueprints}.cr.kanister.io
```

## Additional Example Applications

Check out additional examples [here](https://github.com/kanisterio/kanister/tree/master/examples).

## Support
For troubleshooting help, you can email the [Kanister Google Group](https://groups.google.com/forum/#!forum/kanisterio), reach out to us on [Slack](https://kasten.typeform.com/to/QBcw8T), or file an [issue](https://github.com/kanisterio/kanister/issues).


## License
Apache License 2.0, see [LICENSE](https://github.com/kanisterio/kanister/blob/master/LICENSE).
