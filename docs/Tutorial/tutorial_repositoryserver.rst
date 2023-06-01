.. _tutorial:

Blueprint using kopia repository server as Data Mover
=====================================================

This tutorial will demonstrate the use of Kopia to copy/restore backups 
to a kopia repository. We will be using kanister functions 
that use Kopia repository Server as datamover in the blueprint. For more documentation
on kanister functions and blueprints see :ref:`architecture` ,
:ref:`kanister functions<functions>` respectively

Prerequisites
=============

* Kubernetes ``1.16`` or higher. For cluster version lower than ``1.16``,
  we recommend installing Kanister version ``0.62.0`` or lower.

* `kubectl <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed
  and setup

* `helm <https://helm.sh>`_ installed and initialized using the command `helm init`

* docker

* Kopia repository server controller should be deployed along with Kanister controller
See :ref:`Deploying Kopia Repository server controller`

* Access to s3 bucket and credentials

Example Application
===================

This tutorial begins by deploying a sample application. The application is
contrived, but useful for demonstrating Kanister's features. The application
appends the current time to a log file every second. The application's container
includes the aws command-line client which we'll use later in the tutorial. The
application is installed in the ``default`` namespace.

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: time-logger
  spec:
    replicas: 1
    template:
      metadata:
        labels:
          app: time-logger
      spec:
        containers:
        - name: test-container
          image: containerlabs/aws-sdk
          command: ["sh", "-c"]
          args: ["while true; do for x in $(seq 1200); do date >> /var/log/time.log; sleep 1; done; truncate /var/log/time.log --size 0; done"]
  EOF

Starting Kopia Repository Server
================================

Since we will be using kopia data mover to copy/restore the backups to the location storage,
we need to start the Kopia repository Server. To know more about kopia repository server,
see :ref:`Kopia Repository Server Controller<Kopia Repository Server Controller>`

The repository server controller requires Repository Server custom resource to be created to
start the server. To understand more about this custom resource, see :ref:`architecture`.


Creating a Kopia Repository
---------------------------

The kopia repository needs to be created before we start the repository server.

You can create it as shown below

.. code-block:: bash
  $ kopia --log-level=error --config-file=/tmp/kopia-repository.config 
    --log-dir=/tmp/kopia-cache repository create --no-check-for-updates 
    --cache-directory=/tmp/cache.dir --content-cache-size-mb=0 --metadata-cache-size-mb=500 
    --override-hostname=timelog.app --override-username=kanisterAdmin s3 
    --bucket=test-bucket 
    --prefix=/test/repo-controller
    --region=us-east-1 
    --access-key=<ACCESS_KEY> 
    --secret-access-key=<SECRET_ACCESS_KEY>

You can check `kopia documentation
<https://kopia.io/docs/reference/command-line/>`_ to understand more about kopia repository.


Creating Secrets
----------------

Please see :ref:`architecture` to know the secrets that needs to be created for repository server

- ``Creating TLS secret``
.. code-block:: bash
  $ kubectl create secret tls repository-server-tls-cert --cert=/path/to/certificate.pem --key=/path/to/key.pem -n kanister
  
- ``Creating Repository Server User Access Secret``

.. code-block:: bash
  $ kubectl create secret generic repository-server-user-access --type='secrets.kanister.io/kopia-repository/serveruser' -n kanister

- ``Creating Repository Server Admin Secret``
.. code-block:: bash
  $ kubectl create secret generic repository-server-admin --type='secrets.kanister.io/kopia-repository/serveradmin' -n kanister --from-literal=username=admin@testpod1 --from-literal=password=test1234

- ``Creating Repository Password Secret``
.. code-block:: bash
  $ kubectl create secret generic repository-pass --type='secrets.kanister.io/kopia-repository/password' -n kanister --from-literal=repo-password=test1234

- ``Creating Storage Location Secret``
   The secret should have same values for ``bucket``, ``endpoint``, ``region`` fields that
   we have used while creating kopia repository

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: v1
  kind: Secret
  metadata:
     name: s3-location
     namespace: kanister
  type: secrets.kanister.io/storage-location
  data:
     # required: specify the type of the store
     # supported values are s3, gcs, azure, and file-store
     type: Z2Nz
     # required
     bucket: <base-64-encoded-value>
     # optional: specified in case of S3-compatible stores
     endpoint: <base-64-encoded-value>
     # required, if supported by the provider
     region: <base-64-encoded-value> 
  EOF

- ``Creating Storage Location Credentials Secret``
.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: v1
  kind: Secret
  metadata:
     name: s3-loc-creds
     namespace: kanister
  type: secrets.kanister.io/aws
  data:
     # required: base64 encoded value for key with proper permissions for the bucket
     access-key: <redacted>
     # required: base64 encoded value for the secret corresponding to the key above
     secret-acccess-key: <redacted>
  EOF



Creating Repository Server custom resource
------------------------------------------

Once the secrets are created, we need to create a repository Server CR having references
to above created secrets. More details of the repository server CR 
can be found at :ref:`architecture`

We have to make sure that we use the same values for field ``spec.repository.RootPath``, 
``spec.repository.username`` , ``spec.repository.hostname`` in the CR that we used while
creating the repository in section :ref:`Creating a Kopia Repository<Creating a Kopia Repository>`

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: RepositoryServer
  metadata:
    name: kopia-repo-server
    namespace: kanister
  spec:
    storage:
      secretRef:
        name: s3-location
        namespace: kanister
      credentialSecretRef:
        name: s3-loc-creds
        namespace: kanister
    repository:
      rootPath: /test/repo-controller
      passwordSecretRef:
        name: repository-pass
        namespace: kanister
      username: kansiterAdmin
      hostname: timelog.app
    server:
      adminSecretRef:
        name: repository-server-admin
        namespace: kanister
      tlsSecretRef:
        name: repository-server-tls-cert
        namespace: kanister
      userAccess:
        userAccessSecretRef:
          name: repository-server-user-access
          namespace: kanister
        username: kanisterUser


Invoking Kanister Actions
=========================

Kanister CustomResources are created in the same namespace as
the Kanister controller.

The first Kanister CustomResource we're going to deploy is a Blueprint.
Blueprints are a set of instructions that tell the controller how to perform
actions on an application. An action consists of one or more phases. Each phase
invokes a :doc:`Kanister Function </functions>`. All Kanister functions accept a
list of strings. The ``args`` field in a Blueprint's phase is rendered and passed
into the specified Function.

For more on CustomResources in Kanister, see :ref:`architecture`.


The Blueprint we'll create has a single action called ``backup``.  The action
``backup`` has a single phase named ``backupToS3``. ``backupToS3`` invokes the
Kanister function ``KubeExec``, which is similar to invoking ``kubectl exec ...``.
At this stage, we'll use ``KubeExec`` to echo our time log's name and
:doc:`Kanister's parameter templating </functions>` to specify the container
with our log.