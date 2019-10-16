.. _functions:

Kanister Functions
******************

Kanister Functions are written in go and are compiled when building the
controller. They are referenced by Blueprints phases. A Kanister Function
implements the following go interface:

.. code-block:: go

  // Func allows custom actions to be executed.
  type Func interface {
      Name() string
      Exec(ctx context.Context, args ...string) (map[string]interface{}, error)
      RequiredArgs() []string
  }

Kanister Functions are registered by the return value of ``Name()``, which must be
static.

Each phase in a Blueprint executes a Kanister Function.  The ``Func`` field in
a ``BlueprintPhase`` is used to lookup a Kanister Function.  After
``BlueprintPhase.Args`` are rendered, they are passed into the Kanister Function's
``Exec()`` method.

The ``RequiredArgs`` method returns the list of argument names that are required.

Existing Functions
==================

The Kanister controller ships with the following Kanister Functions out-of-the-box
that provide integration with Kubernetes:

KubeExec
--------

KubeExec is similar to running

.. code-block:: bash

  kubectl exec -it --namespace <NAMESPACE> <POD> -c <CONTAINER> [CMD LIST...]


.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `pod`, Yes, `string`, name of the pod in which to execute
   `container`, Yes, `string`, name of the container in which to execute
   `command`, Yes, `[]string`,  command list to execute

Example:

.. code-block:: yaml
  :linenos:

  - func: KubeExec
    name: examplePhase
    args:
      namespace: "{{ .Deployment.Namespace }}"
      pod: "{{ index .Deployment.Pods 0 }}"
      container: kanister-sidecar
      command:
        - sh
        - -c
        - |
          echo "Example"


KubeExecAll
-----------

KubeExecAll is similar to running KubeExec on multiple containers on
multiple pods (all specified containers on all pods) in parallel.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `pods`, Yes, `[]string`, list of names of pods in which to execute
   `containers`, Yes, `[]string`, list of names of the containers in which to execute
   `command`, Yes, `[]string`,  command list to execute

Example:

.. code-block:: yaml
  :linenos:

  - func: KubeExec
    name: examplePhase
    args:
      namespace: "{{ .Deployment.Namespace }}"
      pods:
        - "{{ index .Deployment.Pods 0 }}"
        - "{{ index .Deployment.Pods 1 }}"
      containers:
        - kanister-sidecar1
        - kanister-sidecar2
      command:
        - sh
        - -c
        - |
          echo "Example"

KubeTask
--------

KubeTask spins up a new container and executes a command via a Pod.
This allows you to run a new Pod from a Blueprint.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `image`, Yes, `string`, image to be used for executing the task
   `command`, Yes, `[]string`,  command list to execute
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

Example:

.. code-block:: yaml
  :linenos:

  - func: KubeTask
    name: examplePhase
    args:
      namespace: "{{ .Deployment.Namespace }}"
      image: busybox
      podOverride:
        containers:
        - name: container
          imagePullPolicy: IfNotPresent
      command:
        - sh
        - -c
        - |
          echo "Example"

ScaleWorkload
-------------

ScaleWorkload is used to scale up or scale down a Kubernetes workload.
The function only returns after the desired replica state is achieved:

* When reducing the replica count, wait until all terminating pods
  complete.

* When increasing the replica count, wait until all pods are ready.

Currently the function supports Deployments and StatefulSets.

It is similar to running

.. code-block:: bash

  kubectl scale deployment <DEPLOYMENT-NAME> --replicas=<NUMBER OF REPLICAS> --namespace <NAMESPACE>

This can be useful if the workload needs to be shutdown before processing
certain data operations. For example, it may be useful to use ``ScaleWorkload``
to stop a database process before restoring files.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, No, `string`, namespace in which to execute
   `name`, No, `string`, name of the workload to scale
   `kind`, No, `string`, `deployment` or `statefulset`
   `replicas`, Yes, `int`,  The desired number of replicas

Example of scaling down:

.. code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: examplePhase
    args:
      namespace: "{{ .Deployment.Namespace }}"
      kind: deployment
      replicas: 0

Example of scaling up:

.. code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: examplePhase
    args:
      namespace: "{{ .Deployment.Namespace }}"
      kind: deployment
      replicas: 1

PrepareData
-----------

This function allows running a new Pod that will mount one or more PVCs
and execute a command or script that manipulates the data on the PVCs.

The function can be useful when it is necessary to perform operations on the
data volumes that are used by one or more application containers. The typical
sequence is to stop the application using ScaleWorkload, perform the data
manipulation using PrepareData, and then restart the application using
ScaleWorkload.

.. note::
   It is extremely important that, if PrepareData modifies the underlying
   data, the PVCs must not be currently in use by an active application
   container (ensure by using ScaleWorkload with replicas=0 first).
   For advanced use cases, it is possible to have concurrent access but
   the PV needs to have RWX mode enabled and the volume needs to use a
   clustered file system that supports concurrent access.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `image`, Yes, `string`, image to be used the command
   `volumes`, No, `map[string]string`, Mapping of ``pvcName`` to ``mountPath`` under which the volume will be available.
   `command`, Yes, `[]string`,  command list to execute
   `serviceaccount`, No, `string`,  service account info
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

.. note::
   The ``volumes`` argument does not support ``subPath`` mounts so the
   data manipulation logic needs to be aware of any ``subPath`` mounts
   that may have been used when mounting a PVC in the primary
   application container.
   If ``volumes`` argument is not specified, all volumes belonging to the protected object
   will be mounted at the predefined path ``/mnt/prepare_data/<pvcName>``

Example:

.. code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: ShutdownApplication
    args:
      namespace: "{{ .Deployment.Namespace }}"
      kind: deployment
      replicas: 0
  - func: PrepareData
    name: ManipulateData
    args:
      namespace: "{{ .Deployment.Namespace }}"
      image: busybox
      volumes:
        application-pvc-1: "/data"
        application-pvc-2: "/restore-data"
      command:
        - sh
        - -c
        - |
          cp /restore-data/file_to_replace.data /data/file.data

.. _backupdata:

BackupData
----------

This function backs up data from a container into any object store
supported by Kanister.

.. note::
   It is important that the application includes a ``kanister-tools``
   sidecar container. This sidecar is necessary to run the
   tools that capture path on a volume and store it on the object store.

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `pod`, Yes, `string`, pod in which to execute
   `container`, Yes, `string`, container in which to execute
   `includePath`, Yes, `string`, path of the data to be backed up
   `backupArtifactPrefix`, Yes, `string`, path to store the backup on the object store
   `encryptionKey`, No, `string`, encryption key to be used for backups

Outputs:

.. csv-table::
   :header: "Output", "Type", "Description"
   :align: left
   :widths: 5,5,15

   `backupTag`,`string`, unique tag added to the backup
   `backupID`,`string`, unique snapshot id generated during backup

Example:

.. code-block:: yaml
  :linenos:

  actions:
    backup:
      type: Deployment
      outputArtifacts:
        backupInfo:
          keyValue:
            backupIdentifier: "{{ .Phases.BackupToObjectStore.Output.backupTag }}"
      phases:
        - func: BackupData
          name: BackupToObjectStore
          args:
            namespace: "{{ .Deployment.Namespace }}"
            pod: "{{ index .Deployment.Pods 0 }}"
            container: kanister-tools
            includePath: /mnt/data
            backupArtifactPrefix: s3-bucket/path/artifactPrefix

.. _backupdataall:

BackupDataAll
-------------

This function concurrently backs up data from one or more pods into an any
object store supported by Kanister.

.. note::
   It is important that the application includes a ``kanister-tools``
   sidecar container. This sidecar is necessary to run the
   tools that capture path on a volume and store it on the object store.

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `pods`, No, `string`, pods in which to execute (by default runs on all the pods)
   `container`, Yes, `string`, container in which to execute
   `includePath`, Yes, `string`, path of the data to be backed up
   `backupArtifactPrefix`, Yes, `string`, path to store the backup on the object store appended by pod name later
   `encryptionKey`, No, `string`, encryption key to be used for backups

Outputs:

.. csv-table::
   :header: "Output", "Type", "Description"
   :align: left
   :widths: 5,5,15

   `BackupAllInfo`,`string`, info about backup tag and identifier required for restore

Example:

.. code-block:: yaml
  :linenos:

  actions:
    backup:
      type: Deployment
      outputArtifacts:
        params:
          keyValue:
            backupInfo: "{{ .Phases.backupToObjectStore.Output.BackupAllInfo }}"
      phases:
        - func: BackupDataAll
          name: BackupToObjectStore
          args:
            namespace: "{{ .Deployment.Namespace }}"
            container: kanister-tools
            includePath: /mnt/data
            backupArtifactPrefix: s3-bucket/path/artifactPrefix

.. _restoredata:

RestoreData
-----------

This function restores data backed up by the BackupData function.
It creates a new Pod that mounts the PVCs referenced by the specified Pod
and restores data to the specified path.

.. note::
   It is extremely important that, the PVCs are not be currently
   in use by an active application container, as they are required
   to be mounted to the new Pod (ensure by using
   ScaleWorkload with replicas=0 first).
   For advanced use cases, it is possible to have concurrent access but
   the PV needs to have RWX mode enabled and the volume needs to use a
   clustered file system that supports concurrent access.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `image`, Yes, `string`, image to be used for running restore
   `backupArtifactPrefix`, Yes, `string`, path to the backup on the object store
   `backupIdentifier`, No, `string`, (required if backupTag not provided) unique snapshot id generated during backup
   `backupTag`, No, `string`, (required if backupIdentifier not provided) unique tag added during the backup
   `restorePath`, No, `string`, path where data is restored
   `pod`, No, `string`, pod to which the volumes are attached
   `volumes`, No, `map[string]string`, Mapping of `pvcName` to `mountPath` under which the volume will be available
   `encryptionKey`, No, `string`, encryption key to be used during backups
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

.. note::
   The ``image`` argument requires the use of ``kanisterio/kanister-tools``
   image since it includes the required tools to restore data from
   the object store.
   Between the ``pod`` and ``volumes`` arguments, exactly one argument
   must be specified.

Example:

Consider a scenario where you wish to restore the data backed up by the
:ref:`backupdata` function. We will first scale down the application,
restore the data and then scale it back up.
For this phase, we will use the ``backupInfo`` Artifact provided by
backup function.

.. substitution-code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: ShutdownApplication
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: Deployment
      replicas: 0
  - func: RestoreData
    name: RestoreFromObjectStore
    args:
      namespace: "{{ .Deployment.Namespace }}"
      pod: "{{ index .Deployment.Pods 0 }}"
      image: kanisterio/kanister-tools:|version|
      backupArtifactPrefix: s3-bucket/path/artifactPrefix
      backupTag: "{{ .ArtifactsIn.backupInfo.KeyValue.backupIdentifier }}"
  - func: ScaleWorkload
    name: StartupApplication
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: Deployment
      replicas: 1


.. _restoredataall:

RestoreDataAll
--------------

This function concurrently restores data backed up by the :ref:`backupdataall`
function, on one or more pods.
It concurrently runs a job Pod for each workload Pod, that mounts the
respective PVCs and restores data to the specified path.

.. note::
   It is extremely important that, the PVCs are not be currently
   in use by an active application container, as they are required
   to be mounted to the new Pod (ensure by using
   ScaleWorkload with replicas=0 first).
   For advanced use cases, it is possible to have concurrent access but
   the PV needs to have RWX mode enabled and the volume needs to use a
   clustered file system that supports concurrent access.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `image`, Yes, `string`, image to be used for running restore
   `backupArtifactPrefix`, Yes, `string`, path to the backup on the object store
   `restorePath`, No, `string`, path where data is restored
   `pods`, No, `string`, pods to which the volumes are attached
   `encryptionKey`, No, `string`, encryption key to be used during backups
   `backupInfo`, Yes, `string`, snapshot info generated as output in BackupDataAll function
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

.. note::
   The `image` argument requires the use of `kanisterio/kanister-tools`
   image since it includes the required tools to restore data from
   the object store.
   Between the `pod` and `volumes` arguments, exactly one argument
   must be specified.

Example:

Consider a scenario where you wish to restore the data backed up by the
:ref:`backupdataall` function. We will first scale down the application,
restore the data and then scale it back up. We will not specify ``pods`` in
args, so this function will restore data on all pods concurrently.
For this phase, we will use the ``params`` Artifact provided by
BackupDataAll function.

.. substitution-code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: ShutdownApplication
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: Deployment
      replicas: 0
  - func: RestoreDataAll
    name: RestoreFromObjectStore
    args:
      namespace: "{{ .Deployment.Namespace }}"
      image: kanisterio/kanister-tools:|version|
      backupArtifactPrefix: s3-bucket/path/artifactPrefix
      backupInfo: "{{ .ArtifactsIn.params.KeyValue.backupInfo }}"
  - func: ScaleWorkload
    name: StartupApplication
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: Deployment
      replicas: 2


CopyVolumeData
--------------

This function copies data from the specified volume (referenced by a
Kubernetes PersistentVolumeClaim) into an object store.
This data can be restored into a volume using the :ref:`restoredata`
function

.. note::
   The PVC must not be in-use (attached to a running Pod)

   If data needs to be copied from a running workload without stopping
   it, use the :ref:`backupdata` function

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace the source PVC is in
   `volume`, Yes, `string`, name of the source PVC
   `dataArtifactPrefix`, Yes, `string`, path on the object store to store the data in
   `encryptionKey`, No, `string`, encryption key to be used during backups
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

Outputs:

.. csv-table::
   :header: "Output", "Type", "Description"
   :align: left
   :widths: 5,5,15

   `backupID`,`string`, unique snapshot id generated when data was copied
   `backupRoot`,`string`,  parent directory location of the data copied from
   `backupArtifactLocation`,`string`, location in objectstore where data was copied
   `backupTag`,`string`,  unique string to identify this data copy

Example:

If the ActionSet ``Object`` is a PersistentVolumeClaim:

.. code-block:: yaml
  :linenos:

  - func: CopyVolumeData
    args:
      namespace: "{{ .PVC.Namespace }}"
      volume: "{{ .PVC.Name }}"
      dataArtifactPrefix: s3-bucket-name/path

DeleteData
----------

This function deletes the snapshot data backed up by the BackupData function.


.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `backupArtifactPrefix`, Yes, `string`, path to the backup on the object store
   `backupIdentifier`, No, `string`, (required if backupTag not provided) unique snapshot id generated during backup
   `backupTag`, No, `string`, (required if backupIdentifier not provided) unique tag added during the backup
   `encryptionKey`, No, `string`, encryption key to be used during backups
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

Example:

Consider a scenario where you wish to delete the data backed up by the
:ref:`backupdata` function.
For this phase, we will use the ``backupInfo`` Artifact provided by backup function.

.. code-block:: yaml
  :linenos:

  - func: DeleteData
    name: DeleteFromObjectStore
    args:
      namespace: "{{ .Namespace.Name }}"
      backupArtifactPrefix: s3-bucket/path/artifactPrefix
      backupTag: "{{ .ArtifactsIn.backupInfo.KeyValue.backupIdentifier }}"

DeleteDataAll
-------------

This function concurrently deletes the snapshot data backed up by the
BackupDataAll function.


.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `backupArtifactPrefix`, Yes, `string`, path to the backup on the object store
   `backupInfo`, Yes, `string`, snapshot info generated as output in BackupDataAll function
   `encryptionKey`, No, `string`, encryption key to be used during backups
   `reclaimSpace`, No, `bool`, provides a way to specify if space should be reclaimed
   `podOverride`, No, `map[string]interface{}`, specs to override default pod specs with

Example:

Consider a scenario where you wish to delete all the data backed up by the
:ref:`backupdataall` function.
For this phase, we will use the ``params`` Artifact provided by backup function.

.. code-block:: yaml
  :linenos:

  - func: DeleteDataAll
    name: DeleteFromObjectStore
    args:
      namespace: "{{ .Namespace.Name }}"
      backupArtifactPrefix: s3-bucket/path/artifactPrefix
      backupInfo: "{{ .ArtifactsIn.params.KeyValue.backupInfo }}"
      reclaimSpace: true

LocationDelete
--------------

This function uses a new Pod to delete the specified artifact
from an object store.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `artifact`, Yes, `string`, artifact to be deleted from the object store

.. note::
   The Kubernetes job uses the ``kanisterio/kanister-tools`` image,
   since it includes all the tools required to delete the artifact
   from an object store.

Example:

.. code-block:: yaml
  :linenos:

  - func: LocationDelete
    name: LocationDeleteFromObjectStore
    args:
      artifact: s3://bucket/path/artifact

.. _createvolumesnapshot:

CreateVolumeSnapshot
--------------------

This function is used to create snapshots of one or more PVCs
associated with an application. It takes individual snapshot
of each PVC which can be then restored later. It generates an
output that contains the Snapshot info required for restoring PVCs.

.. note::
   Currently we only support PVC snapshots on AWS EBS. Support for more storage
   providers is coming soon!

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `pvcs`, No, `[]string`, list of names of PVCs to be backed up
   `skipWait`, No, `bool`, initiate but do not wait for the snapshot operation to complete

When no PVCs are specified in the ``pvcs`` argument above, all PVCs in use by a
Deployment or StatefulSet will be backed up.

Outputs:

.. csv-table::
   :header: "Output", "Type", "Description"
   :align: left
   :widths: 5,5,15

   `volumeSnapshotInfo`,`string`, Snapshot info required while restoring the PVCs

Example:

Consider a scenario where you wish to backup all PVCs of a deployment. The output
of this phase is saved to an Artifact named ``backupInfo``, shown below:

.. code-block:: yaml
  :linenos:

  actions:
    backup:
      type: Deployment
      outputArtifacts:
        backupInfo:
          keyValue:
            manifest: "{{ .Phases.backupVolume.Output.volumeSnapshotInfo }}"
      phases:
      - func: CreateVolumeSnapshot
        name: backupVolume
        args:
          namespace: "{{ .Deployment.Namespace }}"

WaitForSnapshotCompletion
-------------------------

This function is used to wait for completion of snapshot operations
initiated using the :ref:`createvolumesnapshot` function.

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `snapshots`, Yes, `string`, snapshot info generated as output in CreateVolumeSnapshot function

CreateVolumeFromSnapshot
------------------------

This function is used to restore one or more PVCs of an application from the
snapshots taken using the :ref:`createvolumesnapshot` function. It deletes old
PVCs, if present and creates new PVCs from the snapshots taken earlier.

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,20

   `namespace`, Yes, `string`, namespace in which to execute
   `snapshots`, Yes, `string`, snapshot info generated as output in CreateVolumeSnapshot function

Example:

Consider a scenario where you wish to restore all PVCs of a deployment.
We will first scale down the application, restore PVCs and then scale up.
For this phase, we will make use of the backupInfo Artifact provided by
the :ref:`createvolumesnapshot` function.

.. code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: shutdownPod
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: Deployment
      replicas: 0
  - func: CreateVolumeFromSnapshot
    name: restoreVolume
    args:
      namespace: "{{ .Deployment.Namespace }}"
      snapshots: "{{ .ArtifactsIn.backupInfo.KeyValue.manifest }}"
  - func: ScaleWorkload
    name: bringupPod
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: Deployment
      replicas: 1

DeleteVolumeSnapshot
--------------------

This function is used to delete snapshots of PVCs taken using the
:ref:`createvolumesnapshot` function.

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,20

   `namespace`, Yes, `string`, namespace in which to execute
   `snapshots`, Yes, `string`, snapshot info generated as output in CreateVolumeSnapshot function

Example:

.. code-block:: yaml
  :linenos:

  - func: DeleteVolumeSnapshot
    name: deleteVolumeSnapshot
    args:
      namespace: "{{ .Deployment.Namespace }}"
      snapshots: "{{ .ArtifactsIn.backupInfo.KeyValue.manifest }}"

BackupDataStats
---------------

This function get stats for the backed up data from the object store location

.. note::
   It is important that the application includes a ``kanister-tools``
   sidecar container. This sidecar is necessary to run the
   tools that get the information from the object store.

Arguments:

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `backupArtifactPrefix`, Yes, `string`, path to the object store location
   `backupID`, Yes, `string`, unique snapshot id generated during backup
   `mode`, No, `string`, mode in which stats are expected
   `encryptionKey`, No, `string`, encryption key to be used for backups

Outputs:

.. csv-table::
   :header: "Output", "Type", "Description"
   :align: left
   :widths: 5,5,15

   `mode`,`string`, mode of the output stats
   `fileCount`,`string`, number of files in backup
   `size`, `string`, size of the number of files in backup

Example:

.. code-block:: yaml
  :linenos:

  actions:
    backupStats:
      type: Deployment
      outputArtifacts:
        backupStats:
          keyValue:
            mode: "{{ .Phases.backupDataStatsFromObjectStore.Output.BackupDataStatsOutputMode }}"
            fileCount "{{ .Phases.backupDataStatsFromObjectStore.Output.BackupDataStatsOutputFileCount }}"
            size: "{{ .Phases.backupDataStatsFromObjectStore.Output.BackupDataStatsOutputSize }}"
      phases:
        - func: BackupData
          name: BackupToObjectStore
          args:
            namespace: "{{ .Deployment.Namespace }}"
            backupArtifactPrefix: s3-bucket/path/artifactPrefix
            mode: restore-size
            backupID: "{{ .ArtifactsIn.snapshot.KeyValue.backupIdentifier }}"


Registering Functions
---------------------

Kanister can be extended by registering new Kanister Functions.

Kanister Functions are registered using a similar mechanism to `database/sql
<https://golang.org/pkg/database/sql/>`_ drivers. To register new Kanister
Functions, import a package with those new functions into the controller and
recompile it.
