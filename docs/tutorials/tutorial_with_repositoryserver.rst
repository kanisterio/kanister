Using Kopia Repository Server as Data Mover in Blueprint
********************************************************

This tutorial will demonstrate the use of Kopia to copy/restore backups
to a kopia repository. We will be using kanister functions
that use Kopia repository Server as data mover in the blueprint.
For additional documentation on kanister functions and blueprints
refer to the :ref:`architecture` and :ref:`kanister functions<functions>`
sections respectively

.. contents:: Tutorial Overview
  :local:

Prerequisites
=============

* Kubernetes ``1.16`` or higher. For cluster version lower than ``1.16``,
  we recommend installing Kanister version ``0.62.0`` or lower.

* `kubectl <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed
  and setup

* `helm <https://helm.sh>`_ installed and initialized using the command `helm init`

* docker

* Kopia repository server controller should be deployed along with Kanister controller
  Refer
  :ref:`Deploying Kopia Repository server controller <deploying_repo_server_controller>`

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
  apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: time-log-pvc
    labels:
      app: time-logger
  spec:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
  ---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: time-logger
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: time-logger
    template:
      metadata:
        labels:
          app: time-logger
      spec:
        containers:
        - name: test-container
          image: ghcr.io/kanisterio/kanister-tools:0.92.0
          command: ["sh", "-c"]
          args: ["while true; do for x in $(seq 1200); do date >> /var/log/time.log; sleep 1; done; truncate /var/log/time.log --size 0; done"]
          volumeMounts:
          - name: data
            mountPath: /var/log
        volumes:
        - name: data
          persistentVolumeClaim:
            claimName: time-log-pvc
  EOF

Starting Kopia Repository Server
================================

To copy or restore backups to the location storage using the Kopia data mover,
it is necessary to start the Kopia repository server. To learn more about kopia
repository server, refer to :ref:`architecture <architecture>`.

The repository server controller requires the creation of a Repository Server
custom resource to start the server. To understand more about this custom resource,
see :ref:`architecture`.

.. _creating_kopia_repository:

Creating a Kopia Repository
---------------------------

The Kopia repository needs to be created before starting repository server.

You can create it as shown below:

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

To learn more about how to create repository and gain further insight into the Kopia
repository refer to `kopia documentation <https://kopia.io/docs/reference/command-line/>`_


Creating Secrets
----------------

To learn about the secrets that need to be created for the repository server,
Please refer to :ref:`architecture` section

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

   The secret should contain identical values for the ``bucket``, ``endpoint``, ``region``
   fields that were used during the creation of the Kopia repository.

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
     # optional: used as a sub path in the bucket for all backups
     path: <base-64-encoded-value>
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

.. _creating_repo_server_CR:

Creating Repository Server Custom Resource
------------------------------------------

After creating the secrets, it is necessary to generate a repository server CR that
references the previously created secrets. For more detailed information about the
repository server CR, refer to the :ref:`architecture` section.

It is important to ensure consistency by using the same values for the fields
``spec.repository.username`` , ``spec.repository.hostname`` in the CR(Custom Resource) as those
used during the repository creation process described in section
:ref:`Creating a Kopia Repository <creating_kopia_repository>`.

The ``--prefix`` field's value is a combination of prefix specified in `spec.data.path`
field of the location secret and the sub-path provided in the ``spec.repository.RootPath``
field of Repository server CR.

The ``spec.data.path`` field of the location storage secret ``s3-location`` appended
with the ``spec.repository.RootPath`` in the repository Server CR should be combined
together to match the ``--prefix`` field of the command used to create repository,as
specified in section :ref:`Creating a Kopia Repository <creating_kopia_repository>`.


.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
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
        username: kanisteruser
  EOF


After creating the Repository Server, a repository server pod and
a service will be visible in the ``kanister`` namespace,which exposes the
created Kopia repository server.

.. code-block:: bash

   $ kubectl get pods,svc -n kanister
   NAME                                              READY   STATUS    RESTARTS   AGE
   pod/kanister-kanister-operator-5b7dfbf97b-5j5p5   2/2     Running   0          33m
   pod/repo-server-pod-4tjcw                         1/1     Running   0          2m13s

   NAME                                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)     AGE
   service/kanister-kanister-operator   ClusterIP   10.96.197.93    <none>        443/TCP     33m
   service/repo-server-service-rq2pq    ClusterIP   10.96.127.153   <none>        51515/TCP   2m13s

To verify the successful start of the server, you can use the following command to
check the server's status.

.. code-block:: bash

   $ kubectl get repositoryservers.cr.kanister.io kopia-repo-server -n kanister -oyaml
   apiVersion: cr.kanister.io/v1alpha1
   kind: RepositoryServer
   metadata:
     annotations:
       kubectl.kubernetes.io/last-applied-configuration: |
         {"apiVersion":"cr.kanister.io/v1alpha1","kind":"RepositoryServer","metadata":{"annotations":{},"name":"kopia-repo-server","namespace":"kanister"},"spec":{"repository":{"hostname":"timelog.app","passwordSecretRef":{"name":"repository-pass","namespace":"kanister"},"rootPath":"/test/repo-controller","username":"kansiterAdmin"},"server":{"adminSecretRef":{"name":"repository-server-admin","namespace":"kanister"},"tlsSecretRef":{"name":"repository-server-tls-cert","namespace":"kanister"},"userAccess":{"userAccessSecretRef":{"name":"repository-server-user-access","namespace":"kanister"},"username":"kanisteruser"}},"storage":{"credentialSecretRef":{"name":"s3-loc-creds","namespace":"kanister"},"secretRef":{"name":"s3-location","namespace":"kanister"}}}}
     creationTimestamp: "2023-06-05T05:45:49Z"
     generation: 1
     name: kopia-repo-server
     namespace: kanister
     resourceVersion: "41529"
     uid: b4458c4f-b2d5-4dcd-99de-a0a4d32ed216
   spec:
     repository:
       hostname: timelog.app
       passwordSecretRef:
         name: repository-pass
         namespace: kanister
       rootPath: /test/repo-controller
       username: kansiterAdmin
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
         username: kanisteruser
     storage:
       credentialSecretRef:
         name: s3-loc-creds
         namespace: kanister
       secretRef:
         name: s3-location
         namespace: kanister
   status:
     progress: ServerReady
     serverInfo:
       podName: repo-server-pod-4tjcw
       serviceName: repo-server-service-rq2pq

``pod/repo-server-pod-4tjcw`` and ``service/repo-server-service-rq2pq`` populated in
``status.serverInfo`` field  should be used by the client to connect to the server

Invoking Kanister Actions
=========================

Kanister CustomResources are created in the same namespace as
the Kanister controller.

The initial Kanister CustomResource to be deployed is referred to as Blueprint.
Blueprints are a set of instructions that direct the controller on executing
actions on an application. An action consists of one or more phases. Each phase
invokes a :doc:`Kanister Function </functions>`. Every Kanister function accepts a
string list as input. The ``args`` field in a Blueprint's phase is rendered and passed
into the specified function.

To learn more about Kanister's CustomResources, see :ref:`architecture`.

The Blueprint to be created includes two actions called ``backup``
and ``restore``. The ``backup`` action comprises of a single phase named as
``backupToS3``.

``backupToS3`` invokes the Kanister function ``BackupDataUsingKopiaServer``
that uses kopia repository server to copy backup data to s3 storage. The action
``restore`` uses two kanister functions ``ScaleWorkload`` and ``RestoreDataUsingKopiaServer``.
``ScaleWorkload`` function scales down the ``timelog`` application before restoring the data.
``RestoreDataUsingKopiaServer`` restores data using kopia repository server form
s3 storage.

To learn more about the Kanister function, refer to the documentation on
:doc:`Kanister's parameter templating </functions>`.

Output artifacts are used in this scenario to store the data path in s3 and
the corresponding snapshot ID that which will serve as the ``backupIdentifier``
during restoration process.

To know more about artifacts, refer to the :ref:`tutorials` section.

Blueprint
---------

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: Blueprint
  metadata:
    name: time-log-bp
    namespace: kanister
  actions:
    backup:
      outputArtifacts:
        timeLog:
          keyValue:
            path: '/repo-controller/time-logger/'
        backupIdentifier:
          keyValue:
            id: "{{ .Phases.backupToS3.Output.backupID }}"
      phases:
      - func: BackupDataUsingKopiaServer
        name: backupToS3
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod: "{{ index .Deployment.Pods 0 }}"
          container: test-container
          includePath: /var/log
    restore:
      inputArtifactNames:
      - timeLog
      - backupIdentifier
      phases:
      - func: ScaleWorkload
        name: shutdownPod
        args:
          namespace: "{{ .Deployment.Namespace }}"
          name: "{{ .Deployment.Name }}"
          kind: Deployment
          replicas: 0
      - func: RestoreDataUsingKopiaServer
        name: restoreFromS3
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod: "{{ index .Deployment.Pods 0 }}"
          image: ghcr.io/kanisterio/kanister-tools:0.92.0
          backupIdentifier: "{{ .ArtifactsIn.backupIdentifier.KeyValue.id }}"
          restorePath: /var/log
      - func: ScaleWorkload
        name: bringupPod
        args:
          namespace: "{{ .Deployment.Namespace }}"
          name: "{{ .Deployment.Name }}"
          kind: Deployment
          replicas: 1
  EOF

After creating a blueprint, the events associated with it can be viewed by using
the following command:

.. code-block:: yaml

  $ kubectl --namespace kanister describe Blueprint time-log-bp
  Events:
    Type     Reason    Age   From                 Message
    ----     ------    ----  ----                 -------
    Normal   Added      4m   Kanister Controller  Added blueprint time-log-bp

The next CustomResource to be deployed is an ActionSet. An ActionSet is created
whenever there is a need to execute Kanister actions. It contains all
the runtime information that is essential for the controller during execution.
It can include multiple actions, each acting on a different Kubernetes object.
In this tutorial, the forthcoming ActionSet specifies the ``time-logger``
deployment that was previously created and selects the ``backup`` action
within our Blueprint.


Add some data in the time logger app.

.. code-block:: bash

   kubectl exec -it time-logger-6d89687cbb-bmdj8 -n default -it sh
   sh-5.1# cd /var/log/
   sh-5.1# ls
   time.log
   sh-5.1# echo "hello world" >> test.log
   sh-5.1# cat test.log
   hello world

ActionSet
---------

.. code-block:: bash

  # Create action set using the blueprint created in above step
  $ kanctl create actionset --action backup --namespace kanister --blueprint time-log-bp --deployment default/time-logger --repository-server kanister/kopia-repo-server
  actionset actionset backup-rlcnp created

The ``--repository-server`` flag is used to provide the reference to the repository server
CR created in step :ref:`Creating Repository Server custom resource <creating_repo_server_CR>`.
As the CR contains the details related to the Kopia repository server and the associated secrets,
the blueprint can access these details using template parameters. This enables the blueprint to
execute backup operation using the Kopia repository server.


.. code-block:: bash

  $ kubectl describe actionsets.cr.kanister.io backup-rlcnp -n kanister

  Events:
  Type    Reason           Age   From                 Message
  ----    ------           ----  ----                 -------
  Normal  Started Action   14s   Kanister Controller  Executing action backup
  Normal  Started Phase    14s   Kanister Controller  Executing phase backupToS3
  Normal  Ended Phase      9s    Kanister Controller  Completed phase backupToS3
  Normal  Update Complete  9s    Kanister Controller  Updated ActionSet 'backup-rlcnp' Status->complete


Lets delete the date from ``timelogger`` app.

.. code-block:: bash

   $ kubectl exec -it time-logger-6d89687cbb-bmdj8 -n default -it sh
   sh-5.1# cd /var/log/
   sh-5.1# ls -lrt
   total 12
   -rw-r--r-- 1 root root   12 Jun  5 06:22 test.log
   -rw-r--r-- 1 root root 7308 Jun  5 06:26 time.log
   sh-5.1# rm -rf test.log
   sh-5.1# ls -lrt
   total 8
   -rw-r--r-- 1 root root 7482 Jun  5 06:26 time.log


Now, let's proceed with the restore process by using the ``restore`` action from the
``time-log-bp`` blueprint:

.. code-block:: bash

   $ kanctl --namespace kanister create actionset --action restore --from "backup-rlcnp" --repository-server kanister/kopia-repo-server
   actionset restore-backup-rlcnp-g5h65 create

The success of the restore operation can be assessed by describing the actionset.

.. code-block:: bash

  $ kubectl describe actionsets.cr.kanister.io restore-backup-rlcnp-g5h65 -n kanister

  Events:
    Type    Reason           Age   From                 Message
    ----    ------           ----  ----                 -------
    Normal  Started Action   20s   Kanister Controller  Executing action restore
    Normal  Started Phase    20s   Kanister Controller  Executing phase shutdownPod
    Normal  Ended Phase      8s    Kanister Controller  Completed phase shutdownPod
    Normal  Started Phase    8s    Kanister Controller  Executing phase restoreFromS3
    Normal  Ended Phase      4s    Kanister Controller  Completed phase restoreFromS3
    Normal  Started Phase    4s    Kanister Controller  Executing phase bringupPod
    Normal  Ended Phase      3s    Kanister Controller  Completed phase bringupPod
    Normal  Update Complete  2s    Kanister Controller  Updated ActionSet 'restore-backup-rlcnp-g5h65' Status->complete

It is necessary to verify if the data has been successfully restored. The presence of
the ``time.log`` file, which was removed prior to the restore process, should confirm
the successful restoration.

.. code-block:: bash

   $ kubectl exec -it time-logger-6d89687cbb-pv5x6 -n default -it sh
   sh-5.1# ls -lrt /var/log
   total 16
   -rw-r--r-- 1 root root   12 Jun  5 06:22 test.log
   -rw-r--r-- 1 root root 9715 Jun  5 06:32 time.log
   sh-5.1# cat /var/log/test.log
   hello world


