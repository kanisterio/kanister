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
* Kanister version 0.22.0 installed in the namespace ``<kanister-op-namespace>``

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

ElasticSearch
=============
ElasticSearch is a distributed, JSON-based search engine. To install ElasticSearch
we can follow below instructions and use their official helm chart.

Installing ElasticSearch
------------------------

Below commands can be followed to install the ElasticSearch cluster in your cluster

.. code-block:: bash

  # add ElasticSearch helm repo
  $ helm repo add elastic https://helm.elastic.co

  # install the ElasticSearch database (helm V2)
  $ helm install --namespace es-test --name elasticsearch elastic/elasticsearch --set antiAffinity=soft

  # install the ElasticSearch database (helm V3)
  $ kubectl create namespace es-test
  $ helm install --namespace es-test elasticsearch elastic/elasticsearch --set antiAffinity=soft

Backup and Restore of ElasticSearch
-----------------------------------

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

MongoDB
=======

MongoDB is a general purpose, document-based, distributed database built for
modern application developers and for the cloud era.

Installing MongoDB
------------------

You can use below command to install the MongoDB application.

.. code-block:: bash

  # add the helm repo
  $ helm repo add stable https://kubernetes-charts.storage.googleapis.com/

  # update the repo list
  $ helm repo update

  # install the database (helm V2)
  helm install stable/mongodb --name my-release --namespace mongo-test  \
      --set replicaSet.enabled=true                                     \
      --set image.repository=kanisterio/mongodb                         \
      --set image.tag=0.22.0

  # install the database (helm V3)
  $ kubectl create namespace mongo-test
  helm install my-release stable/mongodb --namespace mongo-test         \
      --set replicaSet.enabled=true                                     \
      --set image.repository=kanisterio/mongodb                         \
      --set image.tag=0.22.0

You can notice that we are using a customized image of MongoDB to get it
installed and the only reason we are doing that is because we have to use some
Kanister tools on top of the standard MongoDB image that will help us in
taking backup and restore of the database.

So, in the customized image we are using standard MongoDB as base image and
then just installing some Kanister tools for ex ``kando`` and an other
tool ``restic``.

Backup and Restore of MongoDB
-----------------------------

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

MySQL
=====
MySQL is an open-source relational database management system. In this example we are
going to install it using helm chart and the will follow the same steps to create
``backup`` and then eventually ``restore`` that backup.

Installing MySQL
----------------

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

Backup and Restore of MySQL
---------------------------

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

PostgreSQL
==========


PostgreSQL-Wale
---------------

PostgreSQL is an object-relational database management system (ORDBMS)
with an emphasis on the ability to be extended and on standards-compliance.

Installing PostgreSQL-Wale
^^^^^^^^^^^^^^^^^^^^^^^^^^

You can follow below guide to install PostgreSQL-Wale

.. code-block:: bash

  # add repo
  $ helm repo add stable https://kubernetes-charts.storage.googleapis.com/

  # update repo list
  $ helm repo update

  # install the database (helm V2)
  helm install stable/postgresql --name my-release \
      --namespace postgres-test \
      --set image.repository=kanisterio/postgresql \
      --set image.tag=0.22.0 \
      --set postgresqlPassword=postgres-12345 \
      --set postgresqlExtendedConf.archiveCommand="'envdir /bitnami/postgresql/data/env wal-e wal-push %p'" \
      --set postgresqlExtendedConf.archiveMode=true \
      --set postgresqlExtendedConf.archiveTimeout=60 \
      --set postgresqlExtendedConf.walLevel=archive

  # install the database (helm V3)
  $ kubectl create namespace postgres-test
  helm install stable/postgresql my-release \
      --namespace postgres-test \
      --set image.repository=kanisterio/postgresql \
      --set image.tag=0.22.0 \
      --set postgresqlPassword=postgres-12345 \
      --set postgresqlExtendedConf.archiveCommand="'envdir /bitnami/postgresql/data/env wal-e wal-push %p'" \
      --set postgresqlExtendedConf.archiveMode=true \
      --set postgresqlExtendedConf.archiveTimeout=60 \
      --set postgresqlExtendedConf.walLevel=archive


You can notice that we are using a customized image of ``postgresql`` to get it
installed and the only reason we are doing that is because we have to use some
Kanister tools on top of the standard ``postgresql`` image that will help us in
taking backup and restore of the database.

So, in the customized image we are using standard ``postgresql`` as base image and
then just installing some Kanister tools for ex ``kando`` and an other
tool ``restic``.

Backup and Restore of PostgreSQL-Wale
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Once we have PostgreSQL installed we can create the Kanister resources
that will be used while creating ``Backup`` and ``Restore`` Actionset

Since we already have created Profile resource we will now create Blueprint,
please follow below command to create the Blueprint

.. code-block:: bash

  # replace kanister-op-namespace with the namespace where your kanister operator is installed.
  kubectl create -f https://raw.githubusercontent.com/kanisterio/kanister/master/examples/stable/postgresql-wale/postgresql-blueprint.yaml -n <kanister-op-namespace>

Once we have Profile and Blueprint created, we will have to create
the base backup of the database. Please follow below command to
create the base backup

.. code-block:: bash

  # Find profile name
  $ kubectl get profile -n postgres-test
  NAME               AGE
  s3-profile-zvrg9   109m

  # Create Actionset
  # Create a base backup by creating an ActionSet
  cat << EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
      name: pg-base-backup
      namespace: kasten-io
  spec:
      actions:
      - name: backup
        blueprint: postgresql-blueprint
        object:
          kind: StatefulSet
          name: my-release-postgresql
          namespace: postgres-test
        profile:
          apiVersion: v1alpha1
          kind: Profile
          name: s3-profile-k8s9l
          namespace: postgres-test
        secrets:
          postgresql:
            name: my-release-postgresql
            namespace: postgres-test
  EOF

  # View the status of the actionset
  $ kubectl --namespace kasten-io describe actionset pg-base-backup

Now let's go ahead with creating some data into the database
that we just created, this is the data that we will try to restore
after deleting it manually to imitate disaster.

.. code-block:: bash

  ## Log in into postgresql container and get shell access
  $ kubectl exec -ti my-release-postgresql-0 -n postgres-test -- bash

  ## use psql cli to add entries in postgresql database
  $ PGPASSWORD=${POSTGRES_PASSWORD} psql
  psql (11.5)
  Type "help" for help.

  ## Create DATABASE
  postgres=# CREATE DATABASE test;
  CREATE DATABASE
  postgres=# \l
                                    List of databases
    Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
  -----------+----------+----------+-------------+-------------+-----------------------
  postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  (4 rows)

  ## Create table COMPANY in test database
  postgres=# \c test
  You are now connected to database "test" as user "postgres".
  test=# CREATE TABLE COMPANY(
  test(#     ID INT PRIMARY KEY     NOT NULL,
  test(#     NAME           TEXT    NOT NULL,
  test(#     AGE            INT     NOT NULL,
  test(#     ADDRESS        CHAR(50),
  test(#     SALARY         REAL,
  test(#     CREATED_AT    TIMESTAMP
  test(# );
  CREATE TABLE

  ## Insert data into the table
  test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (10, 'Paul', 32, 'California', 20000.00, now());
  INSERT 0 1
  test=# select * from company;
  id | name | age |                      address                       | salary |         created_at
  ----+------+-----+----------------------------------------------------+--------+----------------------------
  10 | Paul |  32 | California                                         |  20000 | 2019-09-16 14:39:36.316065
  (1 row)

  ## Add few more entries
  test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (20, 'Omkar', 32, 'California', 20000.00, now());
  INSERT 0 1
  test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (30, 'Prasad', 32, 'California', 20000.00, now());
  INSERT 0 1

  test=# select * from company;
  id | name  | age |                      address                       | salary |         created_at
  ----+-------+-----+----------------------------------------------------+--------+----------------------------
  10 | Paul  |  32 | California                                         |  20000 | 2019-09-16 14:39:36.316065
  20 | Omkar |  32 | California                                         |  20000 | 2019-09-16 14:40:52.952459
  30 | Omkar |  32 | California                                         |  20000 | 2019-09-16 14:41:06.433487


After inserting the data into the database, let's assume something bad
happens with the database, and the test database go deleted. To imitate
let's delete the database manually

.. code-block:: bash

  ## Log in into postgresql container and get shell access
  $ kubectl exec -ti my-release-postgresql-0 -n postgres-test -- bash

  ## use psql cli to add entries in postgresql database
  $ PGPASSWORD=${POSTGRES_PASSWORD} psql
  psql (11.5)
  Type "help" for help.

  ## Drop database
  postgres=# \l
                                    List of databases
    Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
  -----------+----------+----------+-------------+-------------+-----------------------
  postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  (4 rows)

  postgres=# DROP DATABASE test;
  DROP DATABASE
  postgres=# \l
                                    List of databases
    Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
  -----------+----------+----------+-------------+-------------+-----------------------
  postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  (3 rows)


To restore the missing data, you should use the backup that you created before.
An easy way to do this is to leverage kanctl, a command-line tool that helps
create ActionSets that depend on other ActionSets:

Let's use PostgreSQL Point-In-Time Recovery to recover data till particular time

.. code-block:: bash

  $ kanctl --namespace kasten-io create actionset --action restore --from pg-base-backup --options pitr=2019-09-16T14:41:00Z
  actionset restore-pg-base-backup-d7g7w created

  ## NOTE: pitr argument to the command is optional. If you want to restore data till the latest consistent state, you can skip '--options pitr' option
  # e.g $ kanctl --namespace kasten-io create actionset --action restore --from pg-base-backup

  ## Check status
  $ kubectl --namespace kasten-io describe actionset restore-pg-base-backup-d7g7w

Once you have verified that the status of the Actionset is complete, you
can login to the database again to make sure the data has been restored
successfully.

.. code-block:: bash

  postgres=# \l
                                    List of databases
    Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
  -----------+----------+----------+-------------+-------------+-----------------------
  postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
            |          |          |             |             | postgres=CTc/postgres
  test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
  (4 rows)

  postgres=# \c test;
  You are now connected to database "test" as user "postgres".
  test=# select * from company;
  id | name  | age |                      address                       | salary |         created_at
  ----+-------+-----+----------------------------------------------------+--------+----------------------------
  10 | Paul  |  32 | California                                         |  20000 | 2019-09-16 14:39:36.316065
  20 | Omkar |  32 | California                                         |  20000 | 2019-09-16 14:40:52.952459

  (2 rows)


PostgreSQL
----------

Installing PostgreSQL
^^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

  # add repo
  $ helm repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com/

  # update repo list
  $ helm dependency update

  # install the database (helm V2)
  $ helm install --namespace kanister --name my-release incubator/patroni

  # install the database (helm V3)
  $ kubectl create namespace kanister
  $ helm install my-release --namespace kanister incubator/patroni


Backup and Restore or PostgreSQL
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Once we have the application up and running we will have to create the Kanister
resources Profile and Blueprint that will be used to create the ``backup``
and ``restore`` Actionset.

// TODO

PostgreSQL on AWS-RDS
---------------------
// TODO

Installing PostgreSQL on AWS-RDS
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Backup and Restore or PostgreSQL
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^


Cassandra
=========

The Apache Cassandra database is the right choice when you need scale ability
and high availability without compromising performance. Linear scale ability
and proven fault-tolerance on commodity hardware or cloud infrastructure make
it the perfect platform for mission-critical data. Cassandra's support for
replicating across multiple data centers is best-in-class, providing lower
latency for your users and the peace of mind of knowing that you can survive
regional outages.

Installing Cassandra
--------------------

To install the Cassandra database we are going to use the standard Cassandra
chart but customized Cassandra image. We had to customize the official Cassandra
just to include some Kanister tooling to helm backup and other things. Please
follow commands to install Cassandra in your machine.

.. code-block:: bash

  # add helm repo
  $ helm repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com

  # Update the helm repo list
  $ helm repo update

  # install the database (helm V2)
  $ helm install --namespace "<app-namespace>"  --name "cassandra" incubator/cassandra --set image.repo=kanisterio/cassandra --set image.tag=0.22.0 --set config.cluster_size=2

  # install the database  (helm V3)
  $ kubectl create namespace <app-namespace>
  $ helm install --namespace "<app-namespace>" "cassandra" incubator/cassandra --set image.repo=kanisterio/cassandra --set image.tag=0.22.0 --set config.cluster_size=2

You can notice that we are using a customized image of Cassandra to get it
installed and the only reason we are doing that is because we have to use some
Kanister tools on top of the standard Cassandra image that will help us in
taking backup and restore of the database.

So, in the customized image we are using standard Cassandra as base image and
then just installing some Kanister tools for ex ``kando`` and an other
tool ``restic``.


Backup and Restore of Cassandra
-------------------------------

Once you have Cassandra database' pods up and running we will have to insert some
records into that database so that we can take of that data to demonstrate the
backup and restore activity.

We will have to Exec into the pod and use Cassandra query language to insert some
data into the Cassandra database.

.. code-block:: bash

  # exec into the cassandra pod
  $ kubectl exec -it -n <app-namespace> cassandra-0 bash

  # once you are inside the pod use `cqlsh` to get into the cassandra CLI and run below commands to create the keyspace
  cqlsh> create keyspace restaurants with replication  = {'class':'SimpleStrategy', 'replication_factor': 3};

  # once the keyspace is created let's create a table named guests and some data into that table
  cqlsh> create table restaurants.guests (id UUID primary key, firstname text, lastname text, birthday timestamp);
  cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e2, 'Robert', 'Downey Jr.', '2015-02-18');

  # once you have the data inserted you can list all the data inside a table using the command
  cqlsh> select * from restaurants.guests;

Once we have inserted data into our Cassandra database, let's go ahead and create Kanister
Blueprint resource so that we can use this in order to create the ``backup`` Actionset. To
create the blueprint please follow below command

.. code-block:: bash

  $ kubectl create -f https://raw.githubusercontent.com/kanisterio/kanister/master/pkg/blueprint/blueprints/cassandra-blueprint.yaml -n <kanister-operator-namespace>

Once you have the blueprint created let's go ahead with creating the Actionset
with ``backup`` action so that we can have ``backup`` of our deployed Cassandra
database.

Please follow below commands to create the Actionset with ``backup`` action

.. code-block:: bash

  # kanister-operator-namespace will be the namespace where you kanister operator is installed
  # blueprint-name will be the name of the blueprint that you will get after creating the blueprint from the Create Blueprint step
  # profile-name will be the profile name you get when you create the profile from Create Profile step

  $ kanctl create actionset --action backup --namespace <kanister-operator-namespace> --blueprint <blueprint-name> --statefulset cassandra/cassandra  --profile cassandra/<profile-name>
  actionset <backup-actionset-name> created

  # you can check the status of the actionset either by describing the actionset resource or by checking the kanister operator's pod log
  $ kubectl describe actionset -n <kanister-operator-namespace> <backup-actionset-name>

If the status of Actionset is complete, it means that the Cassandra database backup
complete. And now that we have taken the backup let's delete the inserted data so
that we can try to restore that by creating another Actionset with ``restore`` action.

Please follow below commands to delete the entire data that we have inserted

.. code-block:: bash

  # Exec into the cassandra pod
  $ kubectl exec -it -n <app-namespace> cassandra-0 bash

  # once you are inside the pod use `cqlsh` to get into the cassandra CLI and run below commands to create the keyspace
  # drop the guests table
  cqlsh> drop table if exists restaurants.guests;

  # drop restaurants keyspace
  cqlsh> drop  keyspace  restaurants;

Now that we have deleted the data, obviously after taking backup, we can create another
Actionset with ``restore`` action to restore the data that we have backed up.

.. code-block:: bash

  $ kanctl --namespace <kanister-operator-namespace> create actionset --action restore --from "<backup-actionset-name>"
  actionset <restore-actionset-name> created
  # you can see the status of the actionset by describing the restore actionset
  $ kubectl describe actionset -n <kanister-operator-namespace> <restore-actionset-name>

Once you have verified that the status of the Actionset is Complete, you can ``exec``
into the Cassandra pods once again and verify that the complete data that we took
backup of has been restored.

.. code-block:: bash

  $ kubectl exec -it -n <app-namespace> cassandra-0 bash
  # once you are inside the pod use `cqlsh` to get into the cassandra CLI and run below commands to create the keyspace
  cqlsh> select * from restaurants.guests;
