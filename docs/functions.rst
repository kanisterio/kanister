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
      Exec(ctx context.Context, args ...string) error
      RequiredArgs() []string
  }

Kanister Functions are registered by the return value of `Name()`, which must be
static.

Each phase in a Blueprint executes a Kanister Function.  The `Func` field in
a `BlueprintPhase` is used to lookup a Kanister Function.  After
`BlueprintPhase.Args` are rendered, they are passed into the Kanister Function's
`Exec()` method.

The `RequiredArgs` method returns the list of argument names that are required.

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

KubeTask spins up a new container and executes a command via a Kubernetes job.
This allows you to run a new Pod from a Blueprint.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `image`, Yes, `string`, image to be used for executing the task
   `command`, Yes, `[]string`,  command list to execute

Example:

.. code-block:: yaml
  :linenos:

  - func: KubeTask
    name: examplePhase
    args:
      namespace: "{{ .Deployment.Namespace }}"
      image: busybox
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

  `kubectl scale deployment <DEPLOYMENT-NAME> --replicas=<NUMBER OF REPLICAS> --namespace <NAMESPACE>`

This can be useful if the workload needs to be shutdown before processing
certain data operations. For example, it may be useful to use `ScaleWorkload`
to stop a database process before restoring files.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `kind`, Yes, `string`, `deployment` or `statefulset`
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

This function allows running a Kubernetes Job that will mount one or more PVCs
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
   `volumes`, No, `map[string]string`, Mapping of `pvcName` to `mountPath` under which the volume will be available.
   `command`, Yes, `[]string`,  command list to execute

.. note::
   The `volumes` argument does not support `subPath` mounts so the
   data manipulation logic needs to be aware of any `subPath` mounts
   that may have been used when mounting a PVC in the primary
   application container.
   If `volumes` argument is not specified, all volumes belonging to the protected object
   will be mounted at the predefined path `/mnt/prepare_data/<pvcName>`

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

BackupData
----------

This function backs up data from a container into an S3 compatible object store.

.. note::
   It is important that the application includes a `kanister-tools`
   sidecar container. This sidecar is necessary to run the
   tools that capture path on a volume and store it on the object store.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, Yes, `string`, namespace in which to execute
   `pod`, Yes, `string`, pod in which to execute
   `container`, Yes, `string`, container in which to execute
   `includePath`, Yes, `string`, path of the data to be backed up
   `backupArtifactPrefix`, Yes, `string`, path to store the backup on the object store
   `backupIdentifier`, Yes, `string`, unique string to identify the backup

Example:

.. code-block:: yaml
  :linenos:

  - func: BackupData
    name: BackupToObjectStore
    args:
      namespace: "{{ .Deployment.Namespace }}"
      pod: "{{ index .Deployment.Pods 0 }}"
      container: kanister-tools
      includePath: /mnt/data
      backupArtifactPrefix: s3-bucket/path/artifactPrefix
      backupIdentifier: "{{ .Time }}"

RestoreData
-----------

This function restores data backed up by the BackupData function.
It creates a Kubernetes Job that mounts the PVCs referenced
by the specified Pod and restores data to the specified path.

.. note::
   It is extremely important that, the PVCs are not be currently
   in use by an active application container, as the Kubernetes Job
   requires to mount the PVCs to a new Pod (ensure by using
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
   `backupIdentifier`, Yes, `string`, unique string to identify the backup
   `restorePath`, No, `string`, path where data is restored
   `pod`, No, `string`, pod to which the volumes are attached
   `volumes`, No, `map[string]string`, Mapping of `pvcName` to `mountPath` under which the volume will be available

.. note::
   The `image` argument requires the use of `kanisterio/kanister-tools`
   image since it includes the required tools to restore data from
   the S3 compatible object store.
   Between the `pod` and `volumes` arguments, exactly one argument
   must be specified.

Example:

.. code-block:: yaml
  :linenos:

  - func: ScaleWorkload
    name: ShutdownApplication
    args:
      namespace: "{{ .Deployment.Namespace }}"
      name: "{{ .Deployment.Name }}"
      kind: deployment
      replicas: 0
  - func: RestoreData
    name: RestoreFromObjectStore
    args:
      namespace: "{{ .Deployment.Namespace }}"
      pod: "{{ index .Deployment.Pods 0 }}"
      image: kanisterio/kanister-tools:0.12.0
      backupArtifactPrefix: s3-bucket/path/artifactPrefix
      backupIdentifier: "{{ .Time }}"

DeleteData
----------

This function uses a Kubernetes Job to delete the specified artifact
from an S3 compatible object store.

.. csv-table::
   :header: "Argument", "Required", "Type", "Description"
   :align: left
   :widths: 5,5,5,15

   `namespace`, No, `string`, namespace in which to execute
   `artifact`, Yes, `string`, artifact to be deleted from the object store

.. note::
   The Kubernetes job uses the `kanisterio/kanister-tools` image,
   since it includes all the tools required to delete the artifact
   from an S3 compatible object store.

Example:

.. code-block:: yaml
  :linenos:

  - func: DeleteData
    name: DeleteFromObjectStore
    args:
      namespace: "{{ .Deployment.Namespace }}"
      artifact: s3://bucket/path/artifact

Registering Functions
---------------------

Kanister can be extended by registering new Kanister Functions.

Kanister Functions are registered using a similar mechanism to `database/sql
<https://golang.org/pkg/database/sql/>`_ drivers. To register new Kanister
Functions, import a package with those new functions into the controller and
recompile it.
