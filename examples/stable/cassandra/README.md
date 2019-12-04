# Cassandra 

As the official documentation of [Cassandra](http://cassandra.apache.org/) says, it is the right choice when you need scalability and high availability without compromising performance. [Linear scalability](http://techblog.netflix.com/2011/11/benchmarking-cassandra-scalability-on.html) and proven fault-tolerance on commodity hardware or cloud infrastructure make it the perfect platform for mission-critical data. Cassandra's support for replicating across multiple datacenters is best-in-class, providing lower latency for your users and the peace of mind of knowing that you can survive regional outages.

## Prerequisites

* Kubernetes 1.9+
* Kubernetes beta APIs enabled only if `podDisruptionBudget` is enabled
* PV support on the underlying infrastructure
* Kanister controller version 0.22.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

To install kanister and related tools you can follow [this](https://docs.kanister.io/install.html#install) link.

**NOTE:**
The helm commands that are mentioned in this 

## Chart Details

We will be using [cassandra helm chart](https://github.com/helm/charts/tree/master/incubator/cassandra) from from official helm repo wchich bootstraps a [cassandra](http://cassandra.apache.org/) cluster on Kubernetes. 

You can decide the number of nodes that will be there in your configured cassandra cluster using the flag `--set config.cluster_size=n` where `n` is the number of nodes you want in your cassandra cluster. For this demo example we will be spinning up our cassandra cluster with 2 nodes.

## Installing the Chart

To install the cassandra cluster in your cluster you can run below command 
```bash
$ helm repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com
$ helm repo update
# remove app-namespace with the namespace you want to deploy the cassandra app in 
$ helm install --namespace "<app-namespace>" "cassandra" incubator/cassandra --set image.repo=viveksinghggits/cassandra --set image.tag=0.22.1 --set config.cluster_size=2
```
This command will install a cassandra on your kubernetes cluster with 2 nodes.

## Integrating with Kanister

We actually mean creating a [profile resource](https://docs.kanister.io/architecture.html#profiles) when we say integrating the application with Kanister. If you have deployed cassandra on your kubernetes cluster using the command that is mentioned above. 

### Create Profile

Please go ahead and crate the profile resource using below command

```bash
kanctl create profile s3compliant --access-key <access-key> \
        --secret-key <secret-key> \
        --bucket <bucket-name> --region <region-name> \
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

Once  cassandra is running, we will have to populate some data into the cassandra database, the data that we willd delete so that we can restore that as part of this example.

Let's add a [keyspace](https://docs.datastax.com/en/dse/5.1/cql/cql/cql_using/cqlKeyspacesAbout.html) called `restaurants`
```bash
# EXEC to the cassandra pod 
$ kubectl exec -it -n <app-namespace> cassandra-0 bash
# once you are inside the pod use `cqlsh` to get into the cassandra CLI and run below commands to create the keyspace 
cqlsh> create keyspace restaurants with replication  = {'class':'SimpleStrategy', 'replication_factor': 3};

# once the keyspace is created lets create a table named guests and some data into that table 
cqlsh> create table restaurants.guests (id UUID primary key, firstname text, lastname text, birthday timestamp);
cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e2, 'Vivek', 'Singh', '2015-02-18');
cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e3, 'Tom', 'Singh', '2015-02-18'); 
cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e4, 'Prasad', 'Hemsworth', '2015-02-18');

# once you have the data inserted you can list all the data inside a table using the command
cqlsh> select * from restaurants.guests;
```

### Protect the application 
The next step is to protect the application/data that we just stored, so that if something bad happens we have the backup data to restore. To protest the application we will have to take the backup of the database using [Actionset](https://docs.kanister.io/architecture.html#actionsets) Kanister resource.
Create an Actionset in the same name space as the kanister controller
```bash
# kanister-operator-namespace will be the namespace where you kanister operator is installed
# blueprint-name will be the name of the blueprint that you will get after creating the blueprint
# profile-name will be the profile name you get when you create the profile 
$ kanctl create actionset --action backup --namespace <kanister-operator-namespace> --blueprint <blueprint-name> --statefulset cassandra/cassandra  --profile cassandra/<profile-name>
actionset <bakcup-actionset-name> created
# you can check the status of the actionset either by describing the actionset resource or by checking the kanister operator's pod log
$ kubectl describe actionset -n <kanister-operator-namespace> <backup-actionset-name>
```

### Disaster strikes!
Let's say someone with fat fingers accidentally deleted the `restaurants` keyspace and `guests` table using the following command in the cassandra pod :
```bash
$ kubectl exec -it -n <app-namespace> cassandra-0 bash
# once you are inside the pod use `cqlsh` to get into the cassandra CLI and run below commands to create the keyspace 
# drop the guests table
cqlsh> drop table if exists restaurants.guests;
# drop restaurants keyspace 
cqlsh> drop  keyspace  restaurants;    
```
If you run the same that you ran earlier to get all the data you should not see any records from the guests table
```bash
$ select * from restaurants.guests;
```

### Restore the Application

Now that we have removed the data from the cassandra table, lets restore the data using the backup that we already have created in earlier step. To do that we will again create an Actionset but for restore instead of backup. You can create the Actionset using below command
```bash
$ kanctl --namespace <kanister-operator-namespace> create actionset --action restore --from "<backup-actionset-name>"
actionset <restore-actionset-name> created
# you can see the status of the actionset by describing the restore actionset
$ kubectl describe actionset -n <kanister-operator-namespace> <restore-actionset-name>
```

Once you have verified that the status of the Actionset <restore-actionset> is completed. You can check if the data is restored or not by `EXEC`ing into the cassandra pod and selecting all the data from the table.
```bash
$ kubectl exec -it -n <app-namespace> cassandra-0 bash
# once you are inside the pod use `cqlsh` to get into the cassandra CLI and run below commands to create the keyspace 
cqlsh> select * from restaurants.guests;
```
and you should be able to see all the records that you have inserted earlier. And that simply means that we were able to restore the data into the cassandra database.

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace <kanister-operator-namespace> create actionset --action delete --from "<backup-actionset-name>"
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
$ kubectl describe actionset <actionset-name> -n <kanister-operator-name>
```

## Uninstalling the Chart

To uninstall/delete the cassandra application run below command:

```bash
$ helm delete cassandra <app-namespace>
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io <blueprint-name> -n <kanister-operator-name>

$ kubectl get profiles.cr.kanister.io -n <app-namespace>
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n <app-namespace>
```
