# Cassandra

As the official documentation of [Cassandra](http://cassandra.apache.org/) says, using this database is the right choice when you need scalability and high availability without compromising performance. [Linear scalability](http://techblog.netflix.com/2011/11/benchmarking-cassandra-scalability-on.html) and proven fault-tolerance on commodity hardware or cloud infrastructure, make it the perfect platform for mission-critical data. Cassandra's support for replicating across multiple datacenters is best-in-class, providing lower latency for your users and the peace of mind of knowing that you can survive regional outages.

## Prerequisites

* Kubernetes 1.9+
* Kubernetes beta APIs enabled only if `podDisruptionBudget` is enabled
* PV support on the underlying infrastructure
* Kanister controller version 0.110.0 installed in your cluster, let's say in namespace `<kanister-operator-namespace>`
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

To install kanister and related tools you can follow [this](https://docs.kanister.io/install.html#install) link.

**NOTE:**
The helm commands that are mentioned in this document are run with the helm version 3. If you are using other helm clinet version the commands may differ slightly.

## Chart Details

We will be using [Cassandra helm chart](https://github.com/bitnami/charts/tree/master/bitnami/cassandra) from official helm repo which bootstraps a [Cassandra](http://cassandra.apache.org/) cluster on Kubernetes.

You can decide the number of nodes that will be there in your configured Cassandra cluster using the flag `--set cluster.replicaCount=n` where `n` is the number of nodes you want in your Cassandra cluster. For this demo example we will be spinning up our Cassandra cluster with 2 nodes.

## Installing the Chart

To install the Cassandra in your Kubernetes cluster you can run below command
```bash
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm repo update
# remove app-namespace with the namespace you want to deploy the Cassandra app in
$ kubectl create ns <app-namespace>
$ helm install cassandra bitnami/cassandra --namespace <app-namespace> --set image.repository=kanisterio/cassandra --set image.tag=0.110.0 --set cluster.replicaCount=2 --set image.registry=ghcr.io --set image.pullPolicy=Always


```
This command will install Cassandra on your Kubernetes cluster with 2 nodes. You can notice that we are using custom image of Cassandra in the helm to install the Cassandra cluster. The reason is we have to use some Kanister tools to take backup, so only change that we have done is including that tooling on top of standard `4.1.3-debian-11-r76` image.

## Integrating with Kanister

We actually mean creating a [profile](https://docs.kanister.io/architecture.html#profiles) and other Kanister resources when we say integrating the application with Kanister. If you have deployed Cassandra on your kubernetes cluster using the command that is mentioned above. Follow the steps below to create a Profile-

### Create Profile

Please go ahead and crate the profile resource using below command

```bash
kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <aws-bucket-name> --region <aws-region-name> \
        --namespace <app-namespace>
```
Please make sure to replace the the values inside `<>` with the acutual values.

**NOTE:**

The command will configure a location where artifacts resulting from Kanister data operations such as backup should go. This is stored as a profiles.cr.kanister.io CustomResource (CR) which is then referenced in Kanister ActionSets. Every ActionSet requires a Profile reference to complete the action. This CR (profiles.cr.kanister.io) can be shared between Kanister-enabled application instances.

### Create Blueprint

Create Blueprint in the same namespace as the Kanister controller
```bash
$ kubectl create -f ./cassandra-blueprint.yaml -n <kanister-operator-namespace>
```

Once Cassandra is running, we will have to populate some data into the Cassandra database, the data that we will delete after, to imitate disaster, so that we can restore that as part of this example.

Let's add a [keyspace](https://docs.datastax.com/en/dse/5.1/cql/cql/cql_using/cqlKeyspacesAbout.html) called `restaurants`
```bash
# EXEC to the Cassandra pod
$ kubectl exec -it -n <app-namespace> cassandra-0 bash
# once you are inside the pod use `cqlsh -u cassandra -p $CASSANDRA_PASSWORD` to get into the Cassandra CLI and run below commands to create the keyspace
cqlsh> create keyspace restaurants with replication  = {'class':'SimpleStrategy', 'replication_factor': 3};
# once the keyspace is created let's create a table named guests and some data into that table
cqlsh> create table restaurants.guests (id UUID primary key, firstname text, lastname text, birthday timestamp);
cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e2, 'Vivek', 'Singh', '2015-02-18');
cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e3, 'Tom', 'Singh', '2015-02-18');
cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e4, 'Prasad', 'Hemsworth', '2015-02-18');
# once you have the data inserted you can list all the data inside a table using the command
cqlsh> select * from restaurants.guests;
```

### Protect the application
The next step is to protect the application/data that we just stored, so that if something bad happens we have the backup data to restore. To protect the application we will have to take the backup of the database using [Actionset](https://1docs.kanister.io/architecture.html#actionsets) Kanister resource.
Create an Actionset in the same name space as the kanister controller
```bash
# kanister-operator-namespace will be the namespace where you kanister operator is installed
# blueprint-name will be the name of the blueprint that you will get after creating the blueprint from the Create Blueprint step
# profile-name will be the profile name you get when you create the profile from Create Profile step
$ kanctl create actionset --action backup --namespace <kanister-operator-namespace> --blueprint <blueprint-name> --statefulset <app-namespace>/cassandra  --profile <app-namespace>/<profile-name>
actionset <backup-actionset-name> created
# you can check the status of the actionset either by describing the actionset resource or by checking the kanister operator's pod log
$ kubectl describe actionset -n <kanister-operator-namespace> <backup-actionset-name>
```
Please make sure the status of the Actionset is completed.

### Disaster strikes!
Let's say someone accidentally deleted the `restaurants` keyspace and `guests` table using the following command in the Cassandra pod. To imitate that you will have to follow these steps manually
```bash
$ kubectl exec -it -n <app-namespace> cassandra-0 bash
# once you are inside the pod use `cqlsh` to get into the Cassandra CLI and run below commands to create the keyspace
# drop the guests table
cqlsh> drop table if exists restaurants.guests;
# drop restaurants keyspace
cqlsh> drop  keyspace  restaurants;
```
If you run the same command that you ran earlier to get all the data you should not see any records from the guests table
```bash
$ select * from restaurants.guests;
```

### Restore the Application

Now that we have removed the data from the Cassandra database's table, let's restore the data using the backup that we have already created in earlier step. To do that we will again create an Actionset resource but for restore instead of backup. You can create the Actionset using below command
```bash
$ kanctl create actionset --action restore --namespace <kanister-operator-namespace> --from "<backup-actionset-name>"
actionset <restore-actionset-name> created
# you can see the status of the actionset by describing the restore actionset
$ kubectl describe actionset -n <kanister-operator-namespace> <restore-actionset-name>
```

Once you have verified that the status of the Actionset `<restore-actionset>` is completed. You can check if the data is restored or not by `EXEC`ing into the Cassandra pod and selecting all the data from the table.
```bash
$ kubectl exec -it -n <app-namespace> cassandra-0 bash
# once you are inside the pod use `cqlsh` to get into the Cassandra CLI and run below commands to create the keyspace
cqlsh> select * from restaurants.guests;
```
and you should be able to see all the records that you have inserted earlier. And that simply means that we were able to restore the data into the Cassandra database.

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl create actionset --action delete --namespace <kanister-operator-namespace> --from "<backup-actionset-name>" --namespacetargets <kanister-operator-namespace>
actionset "<delete-actionset-name>" created

# View the status of the ActionSet
$ kubectl --namespace <kanister-operator-namespace> describe actionset <delete-actionset-name>
```

### Troubleshooting
If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace <kanister-operator-namespace> logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset <actionset-name> -n <kanister-operator-namespace>
```

## Uninstalling the Chart

To uninstall/delete the Cassandra application run below command:

```bash
$ helm delete cassandra <app-namespace>
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io <blueprint-name> -n <kanister-operator-namespace>

$ kubectl get profiles.cr.kanister.io -n <app-namespace>
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n <app-namespace>
```
