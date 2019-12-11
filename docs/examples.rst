.. _examples:

Community Applications Examples
*******************************
This page has examples on how you can go about backing up your application and then,
unfortunately if something bad happens, restoring that backed up application.
Before going through the given application examples you will have to have Kanister
setup.

Prerequisites Details:

* Kubernetes 1.9+ with Beta APIs enabled.
* PV support on the underlying infrastructure.
* Kanister version 0.22.0 installed in the namespace ``kanister-op-namespace``

You can follow :ref:`this <install>` guide to install Kanister, if you don't have it
installed already.

For every database that we are going to discuss here, we are first going to look
into how we can install that database in our cluster and then insert some records into
that database. Next step would be to take backup of the database to recover from any
unfortunate scenarios.

Once we have the data inserted into our database and back up has been taken we will go
ahead and imitate disaster by deleting the data from the database manually. After deleting
the data we will try to recover the lost data by restoring the backup that we have already
created.

To actually backup and restore data using Kanister
:ref:`Actionset <architecture>` resource we will have to create
Profile and Blueprint Kanister resources, these resource support the backup
and restore mechanisms that we are going to achieve using Actionset Kanister
resource.

Creating Profile resource is common to all the applications that we are going
to discuss here so let's start by creating a Profile Kanister resource.

.. code-block:: bash

  $ kanctl --namespace <database-namespace> create profile --bucket <bucket-name> --region <region-name> s3compliant --access-key <aws-access-key> --secret-key <aws-secret-key>

Creating a Kanister Profile actually configures a location where artifacts
resulting from Kanister data operations such as backup should be stored.

Please make a note of the Profile name that we just created, we will need
this Profile name while creating ``backup`` and ``restore`` Actionset.

.. contents:: Application Examples
  :local:

ElasticSearch Example
=====================
ElasticSearch is a distributed, JSON-based search engine. To install ElasticSearch
we can follow below instructions and use their official helm chart. Below commands
can be followed to install the ElasticSearch cluster in your cluster

.. code-block:: bash

  # add ElasticSearch helm repo
  $ helm repo add elastic https://helm.elastic.co

  # install the ElasticSearch database (helm V2)
  $ helm install --namespace es-test --name elasticsearch elastic/elasticsearch --set antiAffinity=soft

  # install the ElasticSearch database (helm V3)
  $ kubectl create namespace es-test
  $ helm install --namespace es-test elasticsearch elastic/elasticsearch --set antiAffinity=soft

Once we have the database installed. Let's go ahead and insert some records into
this ElasticSearch instance. To insert the records into ElasticSearch cluster we
will first have create and index and insert some documents into that index.

.. code-block:: bash

  # create an index called customer
  $ curl -X PUT "localhost:9200/customer?pretty"

  # add a document into the customer index
  $ curl -X PUT "localhost:9200/customer/_doc/1?pretty" -H 'Content-Type: application/json' -d'
  {
    "name": "John Smith"
  }
  '

Once we have created the database and inserted some records into that database.
We will have to create the Kanister resources before we go ahead and take backup
of the database using another Kanister resource.
Since we have created Profile resource already, we will have to create Blueprint
resource. You can create the Blueprint resource using below command

.. code-block:: bash

  $ kubectl create -f https://raw.githubusercontent.com/kanisterio/kanister/master/examples/stable/elasticsearch/elasticsearch-blueprint.yaml -n <kanister-op-namespace>

After creating the Blueprint, we will have to create the Backup of the database,
to create Backup we will have to create Actionset Kanister resource with ``backup``
as action. Please follow below command to create the Actionset.

.. code-block:: bash

  # replace kanister-op-namespace with the namespace, you have installed Kanister in
  # replace blueprint_name with the name of the blueprint that we created in previous step.
  # replace profile_name with the name of the profile that we created earlier
  $ kanctl create actionset --action backup --namespace <kanister-op-namespace> --blueprint <blueprint-name> --statefulset es-test/elasticsearch-master --options --profile es-test/<profile_name>
  actionset <backup-actionset-name> created.
  # you can check the status of the Actionset by describing it to make sure that the Backup is complete
  $ kubectl describe actionset <actionset-name> -n <kanister-op-namespace>

Once the ``backup`` Actionset is complete, we will have to imitate the disaster by
deleting the data from the database. Use below commands to delete the data from the
database

.. code-block:: bash

  # delete the ElasticSearch index
  $ curl -X DELETE "localhost:9200/customer?pretty"
  {
    "acknowledged" : true
  }

Deleting the index from the ElasticSearch cluster will result in all the data getting
deleted and we will now restore that data using restore Actionset. Create another
Actionset with action ``restore`` using following below command

.. code-block:: bash

  # replace backup-actionset-name with the name of the backup that we have already created
  $ kanctl --namespace <kanister-op-namespace> create actionset --action restore --from <backup-actionset-name>
  actionset <restore-actionset-name> created

  # you can check the status of the actionset using describe command
  $ kubectl describe actionset -n <kanister-op-name> <restore-actionset-name>

Once we have verified that the status of the actionset is complete we can go ahead
and check if the document that we stored in our ElasticSearch cluster has been
restored or not.

.. code-block:: bash

  $ curl -X GET "localhost:9200/_cat/indices?v"
  # and you should be able to see the restored index after this command.

So this is how we can use Kanister to backup and eventually restore out database
application.

MongoDB Example
===============

MongoDB is a general purpose, document-based, distributed database built for
modern application developers and for the cloud era.
You can use below command to install the MongoDB application.

.. code-block:: bash

  # add the helm repo
  $ helm repo add stable https://kubernetes-charts.storage.googleapis.com/

  # update the repo list
  $ helm repo update

  # install the application (helm V2)
  helm install stable/mongodb --name my-release --namespace mongo-test  \
      --set replicaSet.enabled=true                                     \
      --set image.repository=kanisterio/mongodb                         \
      --set image.tag=0.22.0

  # install the application (helm V3)
  $ kubectl create namespace mongo-test
  helm install my-release stable/mongodb --namespace mongo-test         \
      --set replicaSet.enabled=true                                     \
      --set image.repository=kanisterio/mongodb                         \
      --set image.tag=0.22.0

You can notice that we are using a customized image to get MongoDB installed and
the only reason we are doing is because we have to install some Kanister tools on
top of the standard MongoDB image.

Once we have the database up and running we will have to insert some records into
the database, to do that we will have to ``EXEC`` into the MongoDB pod and use
MongoDB CLI to create the records.

.. code-block:: bash

  # exec into the mongodb pod
  $ kubectl exec -ti my-release-mongodb-primary-0 -n mongo-test -- bash

  # from  insice the sheel use mongo CLI to insert some data into the mongo database
  $ mongo admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"

  # you can view the inserted data using below command
  $ mongo admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.find()"

Once you have the data inserted into the MongoDB database, let's create the a Blueprint
resource that will be used to create ``backup`` Actionset resource.
To create the Blueprint resource you can follow below command

.. code-block:: bash

  # kanister-op-namespace is namespace where your kanister operator is installed.
  $ kubectl create -f https://raw.githubusercontent.com/kanisterio/kanister/master/examples/stable/mongodb/mongodb-blueprint.yaml -n <kanister-op-namespace>

Now that we have blueprint created, lets create the Actionset with action ``backup``
that will be used to create the backup of the MongoDB database.

.. code-block:: bash

  # replace kanister-op-namespace with namespace you kanister operator is installed in
  $ kanctl create actionset --action backup --namespace <kanister-op-namespace> --blueprint mongodb-blueprint --statefulset mongo-test/my-release-mongodb-primary --profile mongo-test/<profile-name>

  # you can check the status of the actionset by following below command
  $ kubectl describe actionset -n <kanister-op-namespace> <backup-actionset-name>

Please make sure that backup actionset is completed so that we can delete the data
manually in order to restore that. Once you have verified that the Actionset is completed
delete the data from the MongoDB database, using below commands

.. code-block:: bash

  # exec into the mongodb pod
  kubectl exec -ti my-release-mongodb-primary-0 -n mongo-test -- bash

  # drop the database
  $ mongo admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.drop()"

  # if you try to get all the records once again, you should not see them
  $ mongo admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.find()"

Once we have dropped the database, let's go ahead and try to restore the data using
the backup that we already have created. You can follow below commands to create a
restore Actionset.

.. code-block:: bash

  # replace backup-actionset-name with the name of the backup actionset that we created
  $ kanctl --namespace kasten-io create actionset --action restore --from <backup-actionset-name>

  # you can check the status of the this actionset by describing it
  $ kubectl describe actionset <restore-actionset-name> -n <kanister-op-namespace>

Please make sure that the status of the ``restore`` actionset is completed and
we can login into the MongoDB pod once again to check if the data that we had
created earlier has been restored.

MySQL Example
=============
MySQL is an open-source relational database management system. In this example we are
going to install it using helm chart and the will follow the same steps to create
``backup`` and then eventually ``restore`` that backup.
To install the MySQL database please follow below command

.. code-block:: bash

  # add helm repo
  $ helm repo add stable https://kubernetes-charts.storage.googleapis.com/

  # update the helm repo
  $ helm repo update

  # install the database (helm V2)
  helm install stable/mysql -n my-release --namespace mysql-test  \
      --set mysqlRootPassword='asd#45@mysqlEXAMPLE'               \
      --set persistence.size=10Gi

  # install the database (helm V3)
  kubectl create namespace mysql-test
  helm install my-release stable/mysql --namespace mysql-test     \
      --set mysqlRootPassword='asd#45@mysqlEXAMPLE'               \
      --set persistence.size=10Gi

Once we have the MySQL instance running we will have to ``exec`` into the running
pod and create/insert some data into the MySQL database.

.. code-block:: bash

  # get the pods that is running mysql and exec into that mysql pod
  $ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

  # from inside the shell, let's create database and tables
  $ mysql --user=root --password=$MYSQL_ROOT_PASSWORD
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


Once you have inserted the record into the MySQL database, let's go ahead
and create the Blueprint Kanister resource that will be used while creating
``backup`` Actionset.
Please follow below command to to create the blueprint

.. code-block:: bash

  $ kubectl create -f https://raw.githubusercontent.com/kanisterio/kanister/master/examples/stable/mysql/mysql-blueprint.yaml -n <kanister-op-namespace>

  # you can verify the status of the blueprint by describing the actionset
  # replace backup-actionset-name with the name of the actionset that we have just created.
  $ kubectl describe actionset -n <kanister-op-namespace> <backup-actionset-name>

Once we have the blueprint created let's go ahead and create the ``backup``
actionset using the Blueprint and the Profile that we already have created.

.. code-block:: bash

  $ kanctl create actionset --action backup --namespace <kanister-op-namespace> --blueprint mysql-blueprint --deployment mysql-test/my-release-mysql --profile mysql-test/<profile_name> --secrets mysql=mysql-test/my-release-mysql
  actionset <backup-actionset-name> created.

  # you can check the status of teh actionset to make sure the actionset is completed
  $ kubectl describe actionset <backup-actionset-name> -n <kanister-op-namespace>

Once you have verified that the ``backup`` Actionset is completed, we can go ahead
and delete the data from the database to imitate the disaster. Exec into the pod and
run below command to delete the data from the database

.. code-block:: bash

  # exec into the mysql pod
  $ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

  $ mysql --user=root --password=asd#45@mysqlEXAMPLE

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


Once you have deleted the data from the MySQL database let's go ahead and create another
actionset that will ``restore`` that data back into the database.

.. code-block:: bash

  # replace kanister-op-namespace with the namespace you have deployed your kanister operator in
  # replace backup-actionset-name with the backup actionset name that we earlier created.
  $ kanctl --namespace <kanister-op-namespace> create actionset --action restore --from <backup-actionset-name>
  actionset <restore-actionset-name> created.

  # View the status of the ActionSet
  $ kubectl --namespace <kanister-op-namespace> describe actionset <restore-actionset-name>

Once you have verified that the ``restore`` actionset is complete, you can exec
into the MySQL pod once again and make sure the data, that we inserted earlier,
has been restored successfully.

.. code-block:: bash

  $ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

  $ mysql --user=root --password=asd#45@mysqlEXAMPLE
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

And we can see that the data has been restored successfully.

PostgreSQL-Wale Example
=======================
Details of PostgreSQL example

Cassandra Example
=================

The Apache Cassandra database is the right choice when you need scalability
and high availability without compromising performance. Linear scalability
and proven fault-tolerance on commodity hardware or cloud infrastructure make
it the perfect platform for mission-critical data. Cassandra's support for
replicating across multiple datacenters is best-in-class, providing lower
latency for your users and the peace of mind of knowing that you can survive
regional outages.

