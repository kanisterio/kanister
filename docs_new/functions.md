# Functions

Kanister Functions are written in go and are compiled when building the
controller. They are referenced by Blueprints phases. A Kanister
Function implements the following go interface:

``` go
// Func allows custom actions to be executed.
type Func interface {
    Name() string
    Exec(ctx context.Context, args ...string) (map*string*interface{}, error)
    RequiredArgs() []string
    Arguments() []string
}
```

Kanister Functions are registered by the return value of `Name()`, which
must be static.

Each phase in a Blueprint executes a Kanister Function. The `Func` field
in a `BlueprintPhase` is used to lookup a Kanister Function. After
`BlueprintPhase.Args` are rendered, they are passed into the Kanister
Function\'s `Exec()` method.

The `RequiredArgs` method returns the list of argument names that are
required. And `Arguments` method returns the list of all the argument
names that are supported by the function.

## Existing Functions

The Kanister controller ships with the following Kanister Functions
out-of-the-box that provide integration with Kubernetes:

### KubeExec

KubeExec is similar to running

``` bash
kubectl exec -it --namespace <NAMESPACE> <POD> -c <CONTAINER> [CMD LIST...]
```

| Argument   | Required | Type        | Description |
| ---------- | :------: | ----------- | ----------- |
| namespace  | Yes      | string      | namespace in which to execute |
| pod        | Yes      | string      | name of the pod in which to execute |
| container  | No       | string      | (required if pod contains more than 1 container) name of the container in which to execute |
| command    | Yes      | []string    | command list to execute |

Example:

``` yaml
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
```

### KubeExecAll

KubeExecAll is similar to running KubeExec on specified containers of
given pods (all specified containers on given pods) in parallel. In the
below example, the command is going to be executed in both the
containers of the given pods.

  | Argument   | Required | Type        | Description |
  | ---------- | :------: | ----------- | ----------- |
  | namespace  | Yes      | string      | namespace in which to execute |
  | pods       | Yes      | string      | space separated list of names of pods in which to execute |
  | containers | Yes      | string      | space separated list of names of the containers in which to execute |
  | command    | Yes      | []string    | command list to execute |

Example:

``` yaml
- func: KubeExecAll
  name: examplePhase
  args:
    namespace: "{{ .Deployment.Namespace }}"
    pods: "{{ index .Deployment.Pods 0 }} {{ index .Deployment.Pods 1 }}"
    containers: "container1 container2"
    command:
      - sh
      - -c
      - |
        echo "Example"
```

### KubeTask

KubeTask spins up a new container and executes a command via a Pod. This
allows you to run a new Pod from a Blueprint.

  | Argument    | Required | Type                    | Description |
  | ----------- | :------: | ----------------------- | ----------- |
  | namespace   | No       | string                  | namespace in which to execute (the pod will be created in controller's namespace if not specified) |
  | image       | Yes      | string                  | image to be used for executing the task |
  | command     | Yes      | []string                | command list to execute |
  | podOverride | No       | map[string]interface{} | specs to override default pod specs with |
  | podAnnotations | No       | map[string]string | custom annotations for the temporary pod that gets created |
  | podLabels | No       | map[string]string | custom labels for the temporary pod that gets created |

Example:

``` yaml
- func: KubeTask
  name: examplePhase
  args:
    namespace: "{{ .Deployment.Namespace }}"
    image: busybox
    podOverride:
      containers:
      - name: container
        imagePullPolicy: IfNotPresent
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
    command:
      - sh
      - -c
      - |
        echo "Example"
```

### ScaleWorkload

ScaleWorkload is used to scale up or scale down a Kubernetes workload.
It also sets the original replica count of the workload as output
artifact with the key `originalReplicaCount`. The function only returns
after the desired replica state is achieved:

- When reducing the replica count, wait until all terminating pods
    complete.
- When increasing the replica count, wait until all pods are ready.

Currently the function supports Deployments, StatefulSets and
DeploymentConfigs.

It is similar to running

``` bash
kubectl scale deployment <DEPLOYMENT-NAME> --replicas=<NUMBER OF REPLICAS> --namespace <NAMESPACE>
```

This can be useful if the workload needs to be shutdown before
processing certain data operations. For example, it may be useful to use
`ScaleWorkload` to stop a database process before restoring files. See
[Using ScaleWorkload function with output artifact](tasks/scaleworkload.md) for an example with
new `ScaleWorkload` function.

  | Argument     | Required | Type    | Description |
  | ------------ | :------: | ------- | ----------- |
  | namespace    | No       | string  | namespace in which to execute |
  | name         | No       | string  | name of the workload to scale |
  | kind         | No       | string  | [deployment] or [statefulset] |
  | replicas     | Yes      | int     | The desired number of replicas |
  | waitForReady | No       | bool    | Whether to wait for the workload to be ready before executing next steps. Default Value is `true` |

Example of scaling down:

``` yaml
- func: ScaleWorkload
  name: examplePhase
  args:
    namespace: "{{ .Deployment.Namespace }}"
    name: "{{ .Deployment.Name }}"
    kind: deployment
    replicas: 0
```

Example of scaling up:

``` yaml
- func: ScaleWorkload
  name: examplePhase
  args:
    namespace: "{{ .Deployment.Namespace }}"
    name: "{{ .Deployment.Name }}"
    kind: deployment
    replicas: 1
    waitForReady: false
```

### PrepareData

This function allows running a new Pod that will mount one or more PVCs
and execute a command or script that manipulates the data on the PVCs.

The function can be useful when it is necessary to perform operations on
the data volumes that are used by one or more application containers.
The typical sequence is to stop the application using ScaleWorkload,
perform the data manipulation using PrepareData, and then restart the
application using ScaleWorkload.

::: tip NOTE

It is extremely important that, if PrepareData modifies the underlying
data, the PVCs must not be currently in use by an active application
container (ensure by using ScaleWorkload with replicas=0 first). For
advanced use cases, it is possible to have concurrent access but the PV
needs to have RWX mode enabled and the volume needs to use a clustered
file system that supports concurrent access.
:::

  | Argument       | Required | Type                    | Description |
  | -------------- | :------: | ----------------------- | ----------- |
  | namespace      | Yes      | string                  | namespace in which to execute |
  | image          | Yes      | string                  | image to be used the command |
  | volumes        | No       | map[string]string       | Mapping of `pvcName` to `mountPath` under which the volume will be available |
  | command        | Yes      | []string                | command list to execute |
  | serviceaccount | No       | string                  | service account info |
  | podOverride    | No       | map[string]interface{} | specs to override default pod specs with |
  | podAnnotations | No       | map[string]string | custom annotations for the temporary pod that gets created |
  | podLabels | No       | map[string]string | custom labels for the temporary pod that gets created |

::: tip NOTE

The `volumes` argument does not support `subPath` mounts so the data
manipulation logic needs to be aware of any `subPath` mounts that may
have been used when mounting a PVC in the primary application container.
If `volumes` argument is not specified, all volumes belonging to the
protected object will be mounted at the predefined path
`/mnt/prepare_data/<pvcName>`
:::

Example:

``` yaml
- func: ScaleWorkload
  name: ShutdownApplication
  args:
    namespace: "{{ .Deployment.Namespace }}"
    name: "{{ .Deployment.Name }}"
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
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
    command:
      - sh
      - -c
      - |
        cp /restore-data/file_to_replace.data /data/file.data
```

### BackupData

This function backs up data from a container into any object store
supported by Kanister.

::: tip WARNING

The *BackupData* will be deprecated soon. We recommend using
[CopyVolumeData](#copyvolumedata) instead. However, [RestoreData](#restoredata) and
[DeleteData](#deletedata) will continue to be available, ensuring you retain control
over your existing backups.
:::

::: tip NOTE

It is important that the application includes a `kanister-tools` sidecar
container. This sidecar is necessary to run the tools that capture path
on a volume and store it on the object store.
:::

Arguments:

  | Argument             | Required | Type    | Description |
  | -------------------- | :------: | ------- | ----------- |
  | namespace            | Yes      | string  | namespace in which to execute |
  | pod                  | Yes      | string  | pod in which to execute |
  | container            | Yes      | string  | container in which to execute |
  | includePath          | Yes      | string  | path of the data to be backed up |
  | backupArtifactPrefix | Yes      | string  | path to store the backup on the object store |
  | encryptionKey        | No       | string  | encryption key to be used for backups |
  | insecureTLS          | No       | bool    | enables insecure connection for data mover |

Outputs:

  | Output    | Type   | Description |
  | --------- | ------ | ----------- |
  | backupTag | string | unique tag added to the backup |
  | backupID  | string | unique snapshot id generated during backup |

Example:

``` yaml
actions:
  backup:
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
```

### BackupDataAll

This function concurrently backs up data from one or more pods into an
any object store supported by Kanister.

::: tip WARNING

The *BackupDataAll* will be deprecated soon. However, [RestoreDataAll](#restoredataall) and
[DeleteDataAll](#deletedataall) will continue to be available, ensuring you retain control
over your existing backups.
:::

::: tip NOTE

It is important that the application includes a `kanister-tools` sidecar
container. This sidecar is necessary to run the tools that capture path
on a volume and store it on the object store.
:::

Arguments:

  | Argument             | Required | Type    | Description |
  | -------------------- | :------: | ------- | ----------- |
  | namespace            | Yes      | string  | namespace in which to execute |
  | pods                 | No       | string  | pods in which to execute (by default runs on all the pods) |
  | container            | Yes      | string  | container in which to execute |
  | includePath          | Yes      | string  | path of the data to be backed up |
  | backupArtifactPrefix | Yes      | string  | path to store the backup on the object store appended by pod name later |
  | encryptionKey        | No       | string  | encryption key to be used for backups |
  | insecureTLS          | No       | bool    | enables insecure connection for data mover |

Outputs:

  | Output        | Type   | Description |
  | ------------- | ------ | ----------- |
  | BackupAllInfo | string | info about backup tag and identifier required for restore |

Example:

``` yaml
actions:
  backup:
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
```

### RestoreData

This function restores data backed up by the
[BackupData](#backupdata) function. It creates a new
Pod that mounts the PVCs referenced by the specified Pod and restores
data to the specified path.

::: tip NOTE

It is extremely important that, the PVCs are not be currently in use by
an active application container, as they are required to be mounted to
the new Pod (ensure by using ScaleWorkload with replicas=0 first). For
advanced use cases, it is possible to have concurrent access but the PV
needs to have RWX mode enabled and the volume needs to use a clustered
file system that supports concurrent access.
:::

  | Argument             | Required | Type                    | Description |
  | -------------------- | :------: | ----------------------- | ----------- |
  | namespace            | Yes      | string                  | namespace in which to execute |
  | image                | Yes      | string                  | image to be used for running restore |
  | backupArtifactPrefix | Yes      | string                  | path to the backup on the object store |
  | backupIdentifier     | No       | string                  | (required if backupTag not provided) unique snapshot id generated during backup |
  | backupTag            | No       | string                  | (required if backupIdentifier not provided) unique tag added during the backup |
  | restorePath          | No       | string                  | path where data is restored |
  | pod                  | No       | string                  | pod to which the volumes are attached |
  | volumes              | No       | map[string]string       | Mapping of [pvcName] to [mountPath] under which the volume will be available |
  | encryptionKey        | No       | string                  | encryption key to be used during backups |
  | insecureTLS          | No       | bool                    | enables insecure connection for data mover |
  | podOverride          | No       | map[string]interface{}  | specs to override default pod specs with |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

::: tip NOTE

The `image` argument requires the use of
`ghcr.io/kanisterio/kanister-tools` image since it includes the required
tools to restore data from the object store. Between the `pod` and
`volumes` arguments, exactly one argument must be specified.
:::

Example:

Consider a scenario where you wish to restore the data backed up by the
[BackupData](#backupdata) function. We will first scale
down the application, restore the data and then scale it back up. For
this phase, we will use the `backupInfo` Artifact provided by backup
function.

``` yaml
- func: ScaleWorkload name: ShutdownApplication args: namespace: \"{{
    .Deployment.Namespace }}\" name: \"{{ .Deployment.Name }}\" kind:
    Deployment replicas: 0
- func: RestoreData name: RestoreFromObjectStore args: namespace: \"{{
    .Deployment.Namespace }}\" pod: \"{{ index .Deployment.Pods 0 }}\"
    image: ghcr.io/kanisterio/kanister-tools: backupArtifactPrefix:
    s3-bucket/path/artifactPrefix backupTag: \"{{
    .ArtifactsIn.backupInfo.KeyValue.backupIdentifier }}\"
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
- func: ScaleWorkload name: StartupApplication args: namespace: \"{{
    .Deployment.Namespace }}\" name: \"{{ .Deployment.Name }}\" kind:
    Deployment replicas: 1
```

### RestoreDataAll

This function concurrently restores data backed up by the
[BackupDataAll](#backupdataall) function, on one or more
pods. It concurrently runs a job Pod for each workload Pod, that mounts
the respective PVCs and restores data to the specified path.

::: tip NOTE

It is extremely important that, the PVCs are not be currently in use by
an active application container, as they are required to be mounted to
the new Pod (ensure by using ScaleWorkload with replicas=0 first). For
advanced use cases, it is possible to have concurrent access but the PV
needs to have RWX mode enabled and the volume needs to use a clustered
file system that supports concurrent access.
:::

  | Argument             | Required | Type                    | Description |
  | -------------------- | :------: | ----------------------- | ----------- |
  | namespace            | Yes      | string                  | namespace in which to execute |
  | image                | Yes      | string                  | image to be used for running restore |
  | backupArtifactPrefix | Yes      | string                  | path to the backup on the object store |
  | restorePath          | No       | string                  | path where data is restored |
  | pods                 | No       | string                  | pods to which the volumes are attached |
  | encryptionKey        | No       | string                  | encryption key to be used during backups |
  | backupInfo           | Yes      | string                  | snapshot info generated as output in BackupDataAll function |
  | insecureTLS          | No       | bool                    | enables insecure connection for data mover |
  | podOverride          | No       | map[string]interface{} | specs to override default pod specs with |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

::: tip NOTE

The *image* argument requires the use of
[ghcr.io/kanisterio/kanister-tools] image since it includes
the required tools to restore data from the object store. Between the
*pod* and *volumes* arguments, exactly one
argument must be specified.
:::

Example:

Consider a scenario where you wish to restore the data backed up by the
[BackupDataAll](#backupdataall) function. We will first
scale down the application, restore the data and then scale it back up.
We will not specify `pods` in args, so this function will restore data
on all pods concurrently. For this phase, we will use the `params`
Artifact provided by BackupDataAll function.

``` yaml

- func: ScaleWorkload name: ShutdownApplication args: namespace: \"{{
    .Deployment.Namespace }}\" name: \"{{ .Deployment.Name }}\" kind:
    Deployment replicas: 0
- func: RestoreDataAll name: RestoreFromObjectStore args: namespace:
    \"{{ .Deployment.Namespace }}\" image:
    ghcr.io/kanisterio/kanister-tools: backupArtifactPrefix:
    s3-bucket/path/artifactPrefix backupInfo: \"{{
    .ArtifactsIn.params.KeyValue.backupInfo }}\"
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
- func: ScaleWorkload name: StartupApplication args: namespace: \"{{
    .Deployment.Namespace }}\" name: \"{{ .Deployment.Name }}\" kind:
    Deployment replicas: 2
```

### CopyVolumeData

This function copies data from the specified volume (referenced by a
Kubernetes PersistentVolumeClaim) into an object store. This data can be
restored into a volume using the `restoredata`{.interpreted-text
role="ref"} function

::: tip NOTE

The PVC must not be in-use (attached to a running Pod)

If data needs to be copied from a running workload without stopping it,
use the [BackupData](#backupdata) function
:::

Arguments:

  | Argument           | Required | Type                    | Description |
  | ------------------ | :------: | ----------------------- | ----------- |
  | namespace          | Yes      | string                  | namespace the source PVC is in |
  | volume             | Yes      | string                  | name of the source PVC |
  | dataArtifactPrefix | Yes      | string                  | path on the object store to store the data in |
  | encryptionKey      | No       | string                  | encryption key to be used during backups |
  | insecureTLS        | No       | bool                    | enables insecure connection for data mover |
  | podOverride        | No       | map[string]interface{} | specs to override default pod specs with |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

Outputs:

  | Output                 | Type   | Description |
  | ---------------------- | ------ | ----------- |
  | backupID               | string | unique snapshot id generated when data was copied |
  | backupRoot             | string | parent directory location of the data copied from |
  | backupArtifactLocation | string | location in objectstore where data was copied |
  | backupTag              | string | unique string to identify this data copy |

Example:

If the ActionSet `Object` is a PersistentVolumeClaim:

``` yaml
- func: CopyVolumeData
  args:
    namespace: "{{ .PVC.Namespace }}"
    volume: "{{ .PVC.Name }}"
    dataArtifactPrefix: s3-bucket-name/path
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
```

### DeleteData

This function deletes the snapshot data backed up by the
[BackupData](#backupdata) function.

  | Argument             | Required | Type                    | Description |
  | -------------------- | :------: | ----------------------- | ----------- |
  | namespace            | Yes      | string                  | namespace in which to execute |
  | backupArtifactPrefix | Yes      | string                  | path to the backup on the object store |
  | backupID             | No       | string                  | (required if backupTag not provided) unique snapshot id generated during backup |
  | backupTag            | No       | string                  | (required if backupID not provided) unique tag added during the backup |
  | encryptionKey        | No       | string                  | encryption key to be used during backups |
  | insecureTLS          | No       | bool                    | enables insecure connection for data mover |
  | podOverride          | No       | map[string]interface{} | specs to override default pod specs with |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

Example:

Consider a scenario where you wish to delete the data backed up by the
[BackupData](#backupdata) function. For this phase, we
will use the `backupInfo` Artifact provided by backup function.

``` yaml
- func: DeleteData
  name: DeleteFromObjectStore
  args:
    namespace: "{{ .Namespace.Name }}"
    backupArtifactPrefix: s3-bucket/path/artifactPrefix
    backupTag: "{{ .ArtifactsIn.backupInfo.KeyValue.backupIdentifier }}"
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
```

### DeleteDataAll

This function concurrently deletes the snapshot data backed up by the
BackupDataAll function.

  | Argument             | Required | Type                    | Description |
  | -------------------- | :------: | ----------------------- | ----------- |
  | namespace            | Yes      | string                  | namespace in which to execute |
  | backupArtifactPrefix | Yes      | string                  | path to the backup on the object store |
  | backupInfo           | Yes      | string                  | snapshot info generated as output in BackupDataAll function |
  | encryptionKey        | No       | string                  | encryption key to be used during backups |
  | reclaimSpace         | No       | bool                    | provides a way to specify if space should be reclaimed |
  | insecureTLS          | No       | bool                    | enables insecure connection for data mover |
  | podOverride          | No       | map[string]interface{} | specs to override default pod specs with |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

Example:

Consider a scenario where you wish to delete all the data backed up by
the [BackupDataAll](#backupdataall) function. For this
phase, we will use the `params` Artifact provided by backup function.

``` yaml
- func: DeleteDataAll
  name: DeleteFromObjectStore
  args:
    namespace: "{{ .Namespace.Name }}"
    backupArtifactPrefix: s3-bucket/path/artifactPrefix
    backupInfo: "{{ .ArtifactsIn.params.KeyValue.backupInfo }}"
    reclaimSpace: true
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
```

### LocationDelete

This function uses a new Pod to delete the specified artifact from an
object store.

  | Argument | Required | Type   | Description |
  | -------- | :------: | ------ | ----------- |
  | artifact | Yes      | string | artifact to be deleted from the object store |

::: tip NOTE

The Kubernetes job uses the `ghcr.io/kanisterio/kanister-tools` image,
since it includes all the tools required to delete the artifact from an
object store.
:::

Example:

``` yaml
- func: LocationDelete
  name: LocationDeleteFromObjectStore
  args:
    artifact: s3://bucket/path/artifact
```

### CreateVolumeSnapshot

This function is used to create snapshots of one or more PVCs associated
with an application. It takes individual snapshot of each PVC which can
be then restored later. It generates an output that contains the
Snapshot info required for restoring PVCs.

::: tip NOTE

Currently we only support PVC snapshots on AWS EBS. Support for more
storage providers is coming soon!
:::

Arguments:

  | Argument  | Required | Type       | Description |
  | --------- | :------: | ---------- | ----------- |
  | namespace | Yes      | string     | namespace in which to execute |
  | pvcs      | No       | []string   | list of names of PVCs to be backed up |
  | skipWait  | No       | bool       | initiate but do not wait for the snapshot operation to complete |

When no PVCs are specified in the `pvcs` argument above, all PVCs in use
by a Deployment or StatefulSet will be backed up.

Outputs:

  | Output              | Type   | Description |
  | ------------------- | ------ | ----------- |
  | volumeSnapshotInfo  | string | Snapshot info required while restoring the PVCs |

Example:

Consider a scenario where you wish to backup all PVCs of a deployment.
The output of this phase is saved to an Artifact named `backupInfo`,
shown below:

``` yaml
actions:
  backup:
    outputArtifacts:
      backupInfo:
        keyValue:
          manifest: "{{ .Phases.backupVolume.Output.volumeSnapshotInfo }}"
    phases:
    - func: CreateVolumeSnapshot
      name: backupVolume
      args:
        namespace: "{{ .Deployment.Namespace }}"
```

### WaitForSnapshotCompletion

This function is used to wait for completion of snapshot operations
initiated using the [CreateVolumeSnapshot](#createvolumesnapshot) function.
function.

Arguments:

  | Argument  | Required | Type   | Description |
  | --------- | :------: | ------ | ----------- |
  | snapshots | Yes      | string | snapshot info generated as output in CreateVolumeSnapshot function |

### CreateVolumeFromSnapshot

This function is used to restore one or more PVCs of an application from
the snapshots taken using the `createvolumesnapshot`{.interpreted-text
role="ref"} function. It deletes old PVCs, if present and creates new
PVCs from the snapshots taken earlier.

Arguments:

  | Argument  | Required | Type   | Description |
  | --------- | :------: | ------ | ----------- |
  | namespace | Yes      | string | namespace in which to execute |
  | snapshots | Yes      | string | snapshot info generated as output in CreateVolumeSnapshot function |

Example:

Consider a scenario where you wish to restore all PVCs of a deployment.
We will first scale down the application, restore PVCs and then scale
up. For this phase, we will make use of the backupInfo Artifact provided
by the [CreateVolumeSnapshot](#createvolumesnapshot) function.

``` yaml
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
```

### DeleteVolumeSnapshot

This function is used to delete snapshots of PVCs taken using the
[CreateVolumeSnapshot](#createvolumesnapshot) function.

Arguments:

  | Argument  | Required | Type   | Description |
  | --------- | :------: | ------ | ----------- |
  | namespace | Yes      | string | namespace in which to execute |
  | snapshots | Yes      | string | snapshot info generated as output in CreateVolumeSnapshot function |

Example:

``` yaml
- func: DeleteVolumeSnapshot
  name: deleteVolumeSnapshot
  args:
    namespace: "{{ .Deployment.Namespace }}"
    snapshots: "{{ .ArtifactsIn.backupInfo.KeyValue.manifest }}"
```

### BackupDataStats

This function get stats for the backed up data from the object store
location

::: tip NOTE

It is important that the application includes a `kanister-tools` sidecar
container. This sidecar is necessary to run the tools that get the
information from the object store.
:::

Arguments:

  | Argument             | Required | Type   | Description |
  | -------------------- | :------: | ------ | ----------- |
  | namespace            | Yes      | string | namespace in which to execute |
  | backupArtifactPrefix | Yes      | string | path to the object store location |
  | backupID             | Yes      | string | unique snapshot id generated during backup |
  | mode                 | No       | string | mode in which stats are expected |
  | encryptionKey        | No       | string | encryption key to be used for backups |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

Outputs:

  | Output   | Type   | Description |
  | -------- | ------ | ----------- |
  | mode     | string | mode of the output stats |
  | fileCount| string | number of files in backup |
  | size     | string | size of the number of files in backup |

Example:

``` yaml
actions:
  backupStats:
    outputArtifacts:
      backupStats:
        keyValue:
          mode: "{{ .Phases.BackupDataStatsFromObjectStore.Output.mode }}"
          fileCount: "{{ .Phases.BackupDataStatsFromObjectStore.Output.fileCount }}"
          size: "{{ .Phases.BackupDataStatsFromObjectStore.Output.size }}"
    phases:
      - func: BackupDataStats
        name: BackupDataStatsFromObjectStore
        args:
          namespace: "{{ .Deployment.Namespace }}"
          backupArtifactPrefix: s3-bucket/path/artifactPrefix
          mode: restore-size
          backupID: "{{ .ArtifactsIn.snapshot.KeyValue.backupIdentifier }}"
          podAnnotations:
            annKey: annValue
          podLabels:
            labelKey: labelValue
```

### CreateRDSSnapshot

This function creates RDS snapshot of running RDS instance.

Arguments:

  | Argument   | Required | Type   | Description |
  | ---------- | :------: | ------ | ----------- |
  | instanceID | Yes      | string | ID of RDS instance you want to create snapshot of |
  | dbEngine   | No       | string | Required in case of RDS Aurora instance. Supported DB Engines: `aurora` `aurora-mysql` and `aurora-postgresql` |

Outputs:

  | Output           | Type       | Description |
  | ---------------- | ---------- | ----------- |
  | snapshotID       | string     | ID of the RDS snapshot that has been created |
  | instanceID       | string     | ID of the RDS instance |
  | securityGroupID  | []string   | AWS Security Group IDs associated with the RDS instance |
  | allocatedStorage | string     | Specifies the allocated storage size in gibibytes (GiB) |
  | dbSubnetGroup    | string     | Specifies the DB Subnet group associated with the RDS instance |

Example:

``` yaml
actions:
  backup:
    outputArtifacts:
      backupInfo:
        keyValue:
          snapshotID: "{{ .Phases.createSnapshot.Output.snapshotID }}"
          instanceID: "{{ .Phases.createSnapshot.Output.instanceID }}"
          securityGroupID: "{{ .Phases.createSnapshot.Output.securityGroupID }}"
          allocatedStorage: "{{ .Phases.createSnapshot.Output.allocatedStorage }}"
          dbSubnetGroup: "{{ .Phases.createSnapshot.Output.dbSubnetGroup }}"
    configMapNames:
    - dbconfig
    phases:
    - func: CreateRDSSnapshot
      name: createSnapshot
      args:
        instanceID: '{{ index .ConfigMaps.dbconfig.Data "postgres.instanceid" }}'
```

### ExportRDSSnapshotToLocation

This function spins up a temporary RDS instance from the given snapshot,
extracts database dump and uploads that dump to the configured object
storage.

Arguments:

  | Argument             | Required | Type       | Description |
  | -------------------- | :------: | ---------- | ----------- |
  | instanceID           | Yes      | string     | RDS db instance ID |
  | namespace            | Yes      | string     | namespace in which to execute the Kanister tools pod for this function |
  | snapshotID           | Yes      | string     | ID of the RDS snapshot |
  | dbEngine             | Yes      | string     | one of the RDS db engines. Supported engine(s): `PostgreSQL` |
  | username             | No       | string     | username of the RDS database instance |
  | password             | No       | string     | password of the RDS database instance |
  | backupArtifactPrefix | No       | string     | path to store the backup on the object store |
  | databases            | No       | []string   | list of databases to take backup of |
  | securityGroupID      | No       | []string   | list of `securityGroupID` to be passed to temporary RDS instance |
  | dbSubnetGroup        | No       | string     | DB Subnet Group to be passed to temporary RDS instance |
  | image                | No       | string     | kanister-tools image to be used for running export job |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

::: tip NOTE

\- If `databases` argument is not set, backup of all the databases will
be taken. - If `securityGroupID` argument is not set,
`ExportRDSSnapshotToLocation` will find out Security Group IDs
associated with instance with `instanceID` and will pass the same. - If
`backupArtifactPrefix` argument is not set, `instanceID` will be used as
*backupArtifactPrefix*. - If `dbSubnetGroup` argument is not
set, `default` DB Subnet group will be used.
:::

Outputs:

  | Output          | Type       | Description |
  | --------------- | ---------- | ----------- |
  | snapshotID      | string     | ID of the RDS snapshot that has been created |
  | instanceID      | string     | ID of the RDS instance |
  | backupID        | string     | unique backup id generated during storing data into object storage |
  | securityGroupID | []string   | AWS Security Group IDs associated with the RDS instance |

Example:

``` yaml
actions:
  backup:
    outputArtifacts:
      backupInfo:
        keyValue:
          snapshotID: "{{ .Phases.createSnapshot.Output.snapshotID }}"
          instanceID: "{{ .Phases.createSnapshot.Output.instanceID }}"
          securityGroupID: "{{ .Phases.createSnapshot.Output.securityGroupID }}"
          backupID: "{{ .Phases.exportSnapshot.Output.backupID }}"
          dbSubnetGroup: "{{ .Phases.createSnapshot.Output.dbSubnetGroup }}"
    configMapNames:
    - dbconfig
    phases:

    - func: CreateRDSSnapshot
      name: createSnapshot
      args:
        instanceID: '{{ index .ConfigMaps.dbconfig.Data "postgres.instanceid" }}'

    - func: ExportRDSSnapshotToLocation
      name: exportSnapshot
      objects:
        dbsecret:
          kind: Secret
          name: '{{ index .ConfigMaps.dbconfig.Data "postgres.secret" }}'
          namespace: "{{ .Namespace.Name }}"
      args:
        namespace: "{{ .Namespace.Name }}"
        instanceID: "{{ .Phases.createSnapshot.Output.instanceID }}"
        securityGroupID: "{{ .Phases.createSnapshot.Output.securityGroupID }}"
        username: '{{ index .Phases.exportSnapshot.Secrets.dbsecret.Data "username" | toString }}'
        password: '{{ index .Phases.exportSnapshot.Secrets.dbsecret.Data "password" | toString }}'
        dbEngine: "PostgreSQL"
        databases: '{{ index .ConfigMaps.dbconfig.Data "postgres.databases" }}'
        snapshotID: "{{ .Phases.createSnapshot.Output.snapshotID }}"
        backupArtifactPrefix: test-postgresql-instance/postgres
        dbSubnetGroup: "{{ .Phases.createSnapshot.Output.dbSubnetGroup }}"
        podAnnotations:
          annKey: annValue
        podLabels:
          labelKey: labelValue
```

### RestoreRDSSnapshot

This function restores the RDS DB instance either from an RDS snapshot
or from the data dump (if [snapshotID] is not set) that is
stored in an object storage.

::: tip NOTE

\- If [snapshotID] is set, the function will restore RDS
instance from the RDS snapshot. Otherwise *backupID* needs
to be set to restore the RDS instance from data dump. - While restoring
the data from RDS snapshot if RDS instance (where we have to restore the
data) doesn\'t exist, the RDS instance will be created. But if the data
is being restored from the Object Storage (data dump) and the RDS
instance doesn\'t exist new RDS instance will not be created and will
result in an error.
:::

Arguments:

  | Argument             | Required | Type       | Description |
  | -------------------- | :------: | ---------- | ----------- |
  | instanceID           | Yes      | string     | RDS db instance ID |
  | snapshotID           | No       | string     | ID of the RDS snapshot |
  | username             | No       | string     | username of the RDS database instance |
  | password             | No       | string     | password of the RDS database instance |
  | backupArtifactPrefix | No       | string     | path to store the backup on the object store |
  | backupID             | No       | string     | unique backup id generated during storing data into object storage |
  | securityGroupID      | No       | []string   | list of `securityGroupID` to be passed to restored RDS instance |
  | namespace            | No       | string     | namespace in which to execute. Required if `snapshotID` is nil |
  | dbEngine             | No       | string     | one of the RDS db engines. Supported engines: `PostgreSQL`, `aurora`, `aurora-mysql` and `aurora-postgresql`. Required if `snapshotID` is nil or Aurora is run in RDS instance |
  | dbSubnetGroup        | No       | string     | DB Subnet Group to be passed to restored RDS instance |
  | image                | No       | string     |  kanister-tools image to be used for running restore, only relevant when restoring from data dump (if `snapshotID` is empty) |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

::: tip NOTE

\- If `snapshotID` is not set, restore will be done from data dump. In
that case `backupID` [arg] is required. - If
`securityGroupID` argument is not set, `RestoreRDSSnapshot` will find
out Security Group IDs associated with instance with `instanceID` and
will pass the same. - If `dbSubnetGroup` argument is not set, `default`
DB Subnet group will be used.
:::

Outputs:

  | Output  | Type   | Description |
  | ------- | ------ | ----------- |
  | endpoint| string | endpoint of the RDS instance |

Example:

``` yaml
restore:
  inputArtifactNames:
  - backupInfo
  kind: Namespace
  phases:
  - func: RestoreRDSSnapshot
    name: restoreSnapshots
    objects:
      dbsecret:
        kind: Secret
        name: '{{ index .ConfigMaps.dbconfig.Data "postgres.secret" }}'
        namespace: "{{ .Namespace.Name }}"
    args:
      namespace: "{{ .Namespace.Name }}"
      backupArtifactPrefix: test-postgresql-instance/postgres
      instanceID:  "{{ .ArtifactsIn.backupInfo.KeyValue.instanceID }}"
      backupID:  "{{ .ArtifactsIn.backupInfo.KeyValue.backupID }}"
      securityGroupID:  "{{ .ArtifactsIn.backupInfo.KeyValue.securityGroupID }}"
      username: '{{ index .Phases.restoreSnapshots.Secrets.dbsecret.Data "username" | toString }}'
      password: '{{ index .Phases.restoreSnapshots.Secrets.dbsecret.Data "password" | toString }}'
      dbEngine: "PostgreSQL"
      dbSubnetGroup: "{{ .ArtifactsIn.backupInfo.KeyValue.dbSubnetGroup }}"
      podAnnotations:
        annKey: annValue
      podLabels:
        labelKey: labelValue
```

### DeleteRDSSnapshot

This function deletes the RDS snapshot by the [snapshotID].

Arguments:

  | Argument   | Required | Type   | Description |
  | ---------- | :------: | ------ | ----------- |
  | snapshotID | No       | string | ID of the RDS snapshot |

Example:

``` yaml
actions:
  delete:
  kind: Namespace
  inputArtifactNames:
  - backupInfo
  phases:
  - func: DeleteRDSSnapshot
    name: deleteSnapshot
    args:
      snapshotID: "{{ .ArtifactsIn.backupInfo.KeyValue.snapshotID }}"
```

### KubeOps

This function is used to create or delete Kubernetes resources.

Arguments:

  | Argument        | Required | Type                     | Description |
  | --------------- | :------: | ------------------------ | ----------- |
  | operation       | Yes      | string                   | `create` or `delete` Kubernetes resource |
  | namespace       | No       | string                   | namespace in which the operation is executed |
  | spec            | No       | string                   | resource spec that needs to be created |
  | objectReference | No       | map[string]interface{}   | object reference for delete operation |

Example:

``` yaml
- func: KubeOps
  name: createDeploy
  args:
    operation: create
    namespace: "{{ .Deployment.Namespace }}"
    spec: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: "{{ .Deployment.Name }}"
      spec:
        replicas: 1
        selector:
          matchLabels:
            app: example
        template:
          metadata:
            labels:
              app: example
          spec:
            containers:
            - image: busybox
              imagePullPolicy: IfNotPresent
              name: container
              ports:
              - containerPort: 80
                name: http
                protocol: TCP
- func: KubeOps
  name: deleteDeploy
  args:
    operation: delete
    objectReference:
      apiVersion: "{{ .Phases.createDeploy.Output.apiVersion }}"
      group: "{{ .Phases.createDeploy.Output.group }}"
      resource: "{{ .Phases.createDeploy.Output.resource }}"
      name: "{{ .Phases.createDeploy.Output.name }}"
      namespace: "{{ .Phases.createDeploy.Output.namespace }}"
```

### WaitV2

This function is used to wait on a Kubernetes resource until a desired
state is reached. The wait condition is defined in a Go template syntax.

Arguments:

  | Argument   | Required | Type                     | Description |
  | ---------- | :------: | ------------------------ | ----------- |
  | timeout    | Yes      | string                   | wait timeout |
  | conditions | Yes      | map[string]interface{}   | keys should be `allOf` and/or `anyOf` with value as `[]Condition` |

`Condition` struct:

``` yaml
condition: "Go template condition that returns true or false"
objectReference:
  apiVersion: "Kubernetes resource API version"
  resource: "Type of resource to wait for"
  name: "Name of the resource"
```

The Go template conditions can be validated using kubectl commands with
`-o go-template` flag. E.g. To check if the Deployment is ready, the
following Go template syntax can be used with kubectl command

``` bash
kubectl get deploy -n $NAMESPACE $DEPLOY_NAME \
  -o go-template='{{ $available := false }}{{ range $condition := $.status.conditions }}{{ if and (eq .type "Available") (eq .status "True")  }}{{ $available = true }}{{ end }}{{ end }}{{ $available }}'
```

The same Go template can be used as a condition in the WaitV2 function.

Example:

``` yaml
- func: WaitV2
  name: waitForDeploymentReady
  args:
    timeout: 5m
    conditions:
      anyOf:
      - condition: '{{ $available := false }}{{ range $condition := $.status.conditions }}{{ if and (eq .type "Available") (eq .status "True") }}{{ $available = true }}{{ end }}{{ end }}{{ $available }}'
        objectReference:
          apiVersion: "v1"
          group: "apps"
          name: "{{ .Object.metadata.name }}"
          namespace: "{{ .Object.metadata.namespace }}"
          resource: "deployments"
```

### Wait (deprecated)

This function is used to wait on a Kubernetes resource until a desired
state is reached.

Arguments:

  | Argument   | Required | Type                     | Description |
  | ---------- | :------: | ------------------------ | ----------- |
  | timeout    | Yes      | string                   | wait timeout |
  | conditions | Yes      | map[string]interface{}   | keys should be `allOf` and/or `anyOf` with value as `[]Condition` |

`Condition` struct:

``` yaml
condition: "Go template condition that returns true or false"
objectReference:
  apiVersion: "Kubernetes resource API version"
  resource: "Type of resource to wait for"
  name: "Name of the resource"
```

::: tip NOTE

We can refer to the object key-value in Go template condition with the
help of a `$` prefix JSON-path syntax.
:::

Example:

``` yaml
- func: Wait
  name: waitNsReady
  args:
    timeout: 60s
    conditions:
      allOf:
        - condition: '{{ if (eq "{ $.status.phase }" "Invalid")}}true{{ else }}false{{ end }}'
          objectReference:
            apiVersion: v1
            resource: namespaces
            name: "{{ .Namespace.Name }}"
        - condition: '{{ if (eq "{ $.status.phase }" "Active")}}true{{ else }}false{{ end }}'
          objectReference:
            apiVersion: v1
            resource: namespaces
            name: "{{ .Namespace.Name }}"
```

### CreateCSISnapshot

This function is used to create CSI VolumeSnapshot for a
PersistentVolumeClaim. By default, it waits for the VolumeSnapshot to be
`ReadyToUse`.

Arguments:

  | Argument       | Required | Type                 | Description |
  | -------------- | :------: | -------------------- | ----------- |
  | name           | No       | string               | name of the VolumeSnapshot. Default value is `<pvc>-snapshot-<random-alphanumeric-suffix>` |
  | pvc            | Yes      | string               | name of the PersistentVolumeClaim to be captured |
  | namespace      | Yes      | string               | namespace of the PersistentVolumeClaim and resultant VolumeSnapshot |
  | snapshotClass  | Yes      | string               | name of the VolumeSnapshotClass |
  | labels         | No       | map[string]string    | labels for the VolumeSnapshot |

Outputs:

  | Output          | Type   | Description |
  | --------------- | ------ | ----------- |
  | name            | string | name of the CSI VolumeSnapshot |
  | pvc             | string | name of the captured PVC |
  | namespace       | string | namespace of the captured PVC and VolumeSnapshot |
  | restoreSize     | string | required memory size to restore PVC |
  | snapshotContent | string | name of the VolumeSnapshotContent |

Example:

``` yaml
actions:
  backup:
    outputArtifacts:
      snapshotInfo:
        keyValue:
          name: "{{ .Phases.createCSISnapshot.Output.name }}"
          pvc: "{{ .Phases.createCSISnapshot.Output.pvc }}"
          namespace: "{{ .Phases.createCSISnapshot.Output.namespace }}"
          restoreSize: "{{ .Phases.createCSISnapshot.Output.restoreSize }}"
          snapshotContent: "{{ .Phases.createCSISnapshot.Output.snapshotContent }}"
    phases:
    - func: CreateCSISnapshot
      name: createCSISnapshot
      args:
        pvc: "{{ .PVC.Name }}"
        namespace: "{{ .PVC.Namespace }}"
        snapshotClass: do-block-storage
```

### CreateCSISnapshotStatic

This function creates a pair of CSI `VolumeSnapshot` and
`VolumeSnapshotContent` resources, assuming that the underlying *real*
storage volume snapshot already exists. The deletion behavior is defined
by the `deletionPolicy` property (`Retain`, `Delete`) of the snapshot
class.

For more information on pre-provisioned volume snapshots and snapshot
deletion policy, see the Kubernetes
[documentation](https://kubernetes.io/docs/concepts/storage/volume-snapshots/).

Arguments:

  | Argument       | Required | Type   | Description |
  | -------------- | :------: | ------ | ----------- |
  | name           | Yes      | string | name of the new CSI `VolumeSnapshot` |
  | namespace      | Yes      | string | namespace of the new CSI `VolumeSnapshot` |
  | driver         | Yes      | string | name of the CSI driver for the new CSI `VolumeSnapshotContent` |
  | handle         | Yes      | string | unique identifier of the volume snapshot created on the storage backend used as the source of the new `VolumeSnapshotContent` |
  | snapshotClass  | Yes      | string | name of the `VolumeSnapshotClass` to use |

Outputs:

  | Output          | Type   | Description |
  | --------------- | ------ | ----------- |
  | name            | string | name of the new CSI `VolumeSnapshot` |
  | namespace       | string | namespace of the new CSI `VolumeSnapshot` |
  | restoreSize     | string | required memory size to restore the volume |
  | snapshotContent | string | name of the new CSI `VolumeSnapshotContent` |

Example:

``` yaml
actions:
  createStaticSnapshot:
    phases:
    - func: CreateCSISnapshotStatic
      name: createCSISnapshotStatic
      args:
        name: volume-snapshot
        namespace: default
        snapshotClass: csi-hostpath-snapclass
        driver: hostpath.csi.k8s.io
        handle: 7bdd0de3-aaeb-11e8-9aae-0242ac110002
```

### RestoreCSISnapshot

This function restores a new PersistentVolumeClaim using CSI
VolumeSnapshot.

Arguments:

  | Argument      | Required | Type                 | Description |
  | ------------- | :------: | -------------------- | ----------- |
  | name          | Yes      | string               | name of the VolumeSnapshot |
  | pvc           | Yes      | string               | name of the new PVC |
  | namespace     | Yes      | string               | namespace of the VolumeSnapshot and resultant PersistentVolumeClaim |
  | storageClass  | Yes      | string               | name of the StorageClass |
  | restoreSize   | Yes      | string               | required memory size to restore PVC. Must be greater than zero |
  | accessModes   | No       | []string             | access modes for the underlying PV (Default is `["ReadWriteOnce"]`) |
  | volumeMode    | No       | string               | mode of volume (Default is `"Filesystem"`) |
  | labels        | No       | map[string]string    | optional labels for the PersistentVolumeClaim |

::: tip NOTE

Output artifact `snapshotInfo` from `CreateCSISnapshot` function can be
used as an input artifact in this function.
:::

Example:

``` yaml
actions:
  restore:
    inputArtifactNames:
    - snapshotInfo
    phases:
    - func: RestoreCSISnapshot
      name: restoreCSISnapshot
      args:
        name: "{{ .ArtifactsIn.snapshotInfo.KeyValue.name }}"
        pvc: "{{ .ArtifactsIn.snapshotInfo.KeyValue.pvc }}-restored"
        namespace: "{{ .ArtifactsIn.snapshotInfo.KeyValue.namespace }}"
        storageClass: do-block-storage
        restoreSize: "{{ .ArtifactsIn.snapshotInfo.KeyValue.restoreSize }}"
        accessModes: ["ReadWriteOnce"]
        volumeMode: "Filesystem"
```

### DeleteCSISnapshot

This function deletes a VolumeSnapshot from given namespace.

Arguments:

  | Argument  | Required | Type   | Description |
  | --------- | :------: | ------ | ----------- |
  | name      | Yes      | string | name of the VolumeSnapshot |
  | namespace | Yes      | string | namespace of the VolumeSnapshot |

::: tip NOTE

Output artifact `snapshotInfo` from `CreateCSISnapshot` function can be
used as an input artifact in this function.
:::

Example:

``` yaml
actions:
  delete:
    inputArtifactNames:
    - snapshotInfo
    phases:
    - func: DeleteCSISnapshot
      name: deleteCSISnapshot
      args:
        name: "{{ .ArtifactsIn.snapshotInfo.KeyValue.name }}"
        namespace: "{{ .ArtifactsIn.snapshotInfo.KeyValue.namespace }}"
```

### DeleteCSISnapshotContent

This function deletes an unbounded `VolumeSnapshotContent` resource. It
has no effect on bounded `VolumeSnapshotContent` resources, as they
would be protected by the CSI controller.

Arguments:

  | Argument | Required | Type   | Description |
  | -------- | :------: | ------ | ----------- |
  | name     | Yes      | string | name of the `VolumeSnapshotContent` |

Example:

``` yaml
actions:
  deleteVSC:
    phases:
    - func: DeleteCSISnapshotContent
      name: deleteCSISnapshotContent
      args:
        name: "test-snapshot-content-content-dfc8fa67-8b11-4fdf-bf94-928589c2eed8"
```

### BackupDataUsingKopiaServer

This function backs up data from a container into any object store
supported by Kanister using Kopia Repository Server as data mover.

::: tip NOTE

It is important that the application includes a `kanister-tools` sidecar
container. This sidecar is necessary to run the tools that back up the
volume and store it on the object store.

Additionally, in order to use this function, a RepositoryServer CR is
needed while creating the [ActionSets](./architecture.md#actionsets).
:::

Arguments:

  | Argument                    | Required | Type   | Description |
  | --------------------------- | :------: | ------ | ----------- |
  | namespace                   | Yes      | string | namespace of the container that you want to backup the data of |
  | pod                         | Yes      | string | pod name of the container that you want to backup the data of |
  | container                   | Yes      | string | name of the kanister sidecar container |
  | includePath                 | Yes      | string | path of the data to be backed up |
  | snapshotTags                | No       | string | custom tags to be provided to the kopia snapshots |
  | repositoryServerUserHostname| No       | string | user's hostname to access the kopia repository server. Hostname would be available in the user access credential secret |

Outputs:

  | Output   | Type   | Description |
  | -------- | ------ | ----------- |
  | backupID | string | unique snapshot id generated during backup |
  | size     | string | size of the backup |
  | phySize  | string | physical size of the backup |

Example:

``` yaml
actions:
backup:
  outputArtifacts:
    backupIdentifier:
      keyValue:
        id: "{{ .Phases.backupToS3.Output.backupID }}"
  phases:
  - func: BackupDataUsingKopiaServer
    name: backupToS3
    args:
      namespace: "{{ .Deployment.Namespace }}"
      pod: "{{ index .Deployment.Pods 0 }}"
      container: kanister-tools
      includePath: /mnt/data
```

### RestoreDataUsingKopiaServer

This function restores data backed up by the
`BackupDataUsingKopiaServer` function. It creates a new Pod that mounts
the PVCs referenced by the Pod specified in the function argument and
restores data to the specified path.

::: tip NOTE

It is extremely important that, the PVCs are not currently in use by an
active application container, as they are required to be mounted to the
new Pod (ensure by using `ScaleWorkload` with replicas=0 first). For
advanced use cases, it is possible to have concurrent access but the PV
needs to have `RWX` access mode and the volume needs to use a clustered
file system that supports concurrent access.
:::

  | Argument                    | Required | Type                   | Description |
  | --------------------------- | :------: | ---------------------- | ----------- |
  | namespace                   | Yes      | string                 | namespace of the application that you want to restore the data in |
  | image                       | Yes      | string                 | image to be used for running restore job (should contain kopia binary) |
  | backupIdentifier            | Yes      | string                 | unique snapshot id generated during backup |
  | restorePath                 | Yes      | string                 | path where data to be restored |
  | pod                         | No       | string                 | pod to which the volumes are attached |
  | volumes                     | No       | map[string]string      | mapping of [pvcName] to [mountPath] under which the volume will be available |
  | podOverride                 | No       | map[string]interface{} | specs to override default pod specs with |
  | repositoryServerUserHostname| No       | string                 | user's hostname to access the kopia repository server. Hostname would be available in the user access credential secret |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

::: tip NOTE

The `image` argument requires the use of
`ghcr.io/kanisterio/kanister-tools` image since it includes the required
tools to restore data from the object store.

Either `pod` or the `volumes` arguments must be specified to this
function based on the function that was used to backup the data. If
[BackupDataUsingKopiaServer] is used to backup the data we
should specify *pod* and for
[CopyVolumeDataUsingKopiaServer], *volumes*
should be specified.

Additionally, in order to use this function, a RepositoryServer CR is
required.
:::

Example:

Consider a scenario where you wish to restore the data backed up by the
[BackupDataUsingKopiaServer](#backupdatausingkopiaserver) function. We
will first scale down the application, restore the data and then scale
it back up. For this phase, we will use the `backupIdentifier` Artifact
provided by backup function.

``` yaml

- func: ScaleWorkload name: shutdownPod args: namespace: \"{{
    .Deployment.Namespace }}\" name: \"{{ .Deployment.Name }}\" kind:
    Deployment replicas: 0
- func: RestoreDataUsingKopiaServer name: restoreFromS3 args:
    namespace: \"{{ .Deployment.Namespace }}\" pod: \"{{ index
    .Deployment.Pods 0 }}\" backupIdentifier: \"{{
    .ArtifactsIn.backupIdentifier.KeyValue.id }}\" restorePath:
    /mnt/data
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
- func: ScaleWorkload name: bringupPod args: namespace: \"{{
    .Deployment.Namespace }}\" name: \"{{ .Deployment.Name }}\" kind:
    Deployment replicas: 1
```

### DeleteDataUsingKopiaServer

This function deletes the snapshot data backed up by the
`BackupDataUsingKopiaServer` function. It creates a new Pod that runs
`delete snapshot` command.

::: tip NOTE

The `image` argument requires the use of
`ghcr.io/kanisterio/kanister-tools` image since it includes the required
tools to delete snapshot from the object store.

Additionally, in order to use this function, a RepositoryServer CR is
required.
:::

  | Argument                    | Required | Type   | Description |
  | --------------------------- | :------: | ------ | ----------- |
  | namespace                   | Yes      | string | namespace in which to execute the delete job |
  | backupID                    | Yes      | string | unique snapshot id generated during backup |
  | image                       | Yes      | string | image to be used for running delete job (should contain kopia binary) |
  | repositoryServerUserHostname| No       | string | user's hostname to access the kopia repository server. Hostname would be available in the user access credential secret |
  | podAnnotations       | No       | map[string]string       | custom annotations for the temporary pod that gets created |
  | podLabels            | No       | map[string]string       | custom labels for the temporary pod that gets created |

Example:

Consider a scenario where you wish to delete the data backed up by the
[BackupDataUsingKopiaServer](#backupdatausingkopiaserver) function. For
this phase, we will use the `backupIdentifier` Artifact provided by
backup function.

``` yaml
- func: DeleteDataUsingKopiaServer
  name: DeleteFromObjectStore
  args:
    namespace: "{{ .Deployment.Namespace }}"
    backupID: "{{ .ArtifactsIn.backupIdentifier.KeyValue.id }}"
    image: ghcr.io/kanisterio/kanister-tools:0.89.0
    podAnnotations:
      annKey: annValue
    podLabels:
      labelKey: labelValue
```

### Registering Functions

Kanister can be extended by registering new Kanister Functions.

Kanister Functions are registered using a similar mechanism to
[database/sql](https://golang.org/pkg/database/sql/) drivers. To
register new Kanister Functions, import a package with those new
functions into the controller and recompile it. -->
