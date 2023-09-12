# Problem Statement

Kanister actions are triggered by creating an ActionSet resource. The status
field in the ActionSet CR represents the status of a Kanister operation. The
action completion percentage is set by calculating the weights given on phases
([https://github.com/kanisterio/kanister/blob/master/design/progress-tracking.md](https://github.com/kanisterio/kanister/blob/master/design/progress-tracking.md)).

The goal of this proposal is to

- Have progress tracking per phase
- Improve action progress tracking mechanism by removing hard-coded weights

## High Level Design

### Changes in ActionSet `status` Field

In order to show the progress of each phase execution while performing an
action, a progress field can be added to the `status.phase`.

```
// ActionStatus is updated as we execute phases.
type ActionStatus struct {
        // Name is the action we'll perform. For example: `backup` or `restore`.
        Name string `json:"name"`
        // Object refers to the thing we'll perform this action on.
        Object ObjectReference `json:"object"`
        // Blueprint with instructions on how to execute this action.
        Blueprint string `json:"blueprint"`
        // Phases are sub-actions an are executed sequentially.
        Phases []Phase `json:"phases,omitempty"`
        // Artifacts created by this phase.
        Artifacts map[string]Artifact `json:"artifacts,omitempty"`
        // DeferPhase is the phase that is executed at the end of an action
        // irrespective of the status of other phases in the action
        DeferPhase Phase `json:"deferPhase,omitempty"`
}

 // Phase is subcomponent of an action.
 type Phase struct {
        Name     string                 `json:"name"`
        State    State                  `json:"state"`
        Output   map[string]interface{} `json:"output,omitempty"`
+       Progress PhaseProgress          `json:"progress,omitempty"` <-----
 }

+type PhaseProgress struct {
+       ProgressPercent       int64
+       LastTransitionTime    *metav1.Time
+       SizeUploadedB         int64
+       EstimatedTimeSeconds  int64
+       EstimatedUploadSizeB  int64
+}
```

### Action Progress Tracking

An Action consists of multiple phases. The phases consume different duration
for completion based on the operations they perform. All the phases in an
action can be given equal weightage, and to calculate the action completion
progress %, the average of all phase completion % can be calculated.

```
% completion of action = sum(% completion of all the phases)/ no. of phases
```

### Phase Progress Tracking

Each Kanister function performs its operations in different ways. Since the
operation execution is specific to a function, each function has a specific way
to calculate and provide info about the progress. For example, datamover
functions can leverage Kopia stats to track progress. So, the progress tracking
can be delegated to the function implementation.

Each Kanister function calculates and returns the progress stats by
implementing the `Progress` interface.

**Progress interface**

```
type Progress interface {
        Stats() (*FuncProgress, error)
}

type FuncProgress struct {
        ProgressPercent      int64
        SizeUploadedB        int64
        EstimatedTimeSeconds int64
        EstimatedUploadSizeB int64
}
```

**Kanister function interface**

```
type Func interface {
+       Progress
        RequiredArgs() []string
        Arguments() []string
        Exec(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)
 }
```

The `PhaseProgress.ProgressPercent` is the progress of its Kanister function.

The following ActionSet example shows the status of an action with 3 phases.
The first phase is complete and the second one is running.

```
status:
    actions:
    - artifacts:
        mysqlCloudDump:
          keyValue:
            s3path: '{{ .Phases.dumpToObjectStore.Output.s3path }}'
      blueprint: mysql-blueprint
      name: backup
      object:
        kind: statefulset
        name: mysql
        namespace: mysql
      phases:
      - name: updatePermissions
        state: completed
        progress:
          progressPercent: "100.00"
          lastTransitionTime:  "2023-02-20T12:448:55Z"
      - name: dumpToObjectStore
        state: running
        progress:
          progressPercent: "30.00"
          lastTransitionTime:  "2023-02-20T12:49:55Z"
          sizeUploaded: "50000"
          estimatedTimeSeconds: "120"
          estimatedUploadSize: "100000000"
      - name: cleanup
        state: pending
    error:
      message: ""
    progress:
      lastTransitionTime: "2023-02-20T12:49:55Z"
      percentCompleted: "43.33"
      runningPhase: dumpToObjectStore
    state: running
```

### Kanister Functions Progress Tracking

At a high level, we can divide the Kanister functions into two groups.

#### Datamover Kanister Functions

Kanister data mover functions (like `BackupData`, `CopyVolumeData`,
`RestoreData`, etc.) use Kopia to snapshot the filesystem and move data to/from
external storage.

While snapshotting, Kopia provides the info about progress on stdout.

```
$ kopia --log-level=error --config-file=/tmp/kopia-repository.config --log-dir=/tmp/kopia-log snapshot --progress-update-interval=5s /data
 / 1 hashing, 118 hashed (546.9 MB), 0 cached (0 B), uploaded 0 B, estimated 4.5 GB (12.1%) 35s left
 * 0 hashing, 136 hashed (1.4 GB), 0 cached (0 B), uploaded 244 B, estimated 4.5 GB (30.1%) 30s left
```

The command output can be parsed to get the progress metadata.

#### Non-datamover Kanister Functions

Non-datamover Kanister functions like `KubeTask`, `KubeExec`, `KubeOps`,
`ScaleWorkload`, etc., allow users to perform operations like executing scripts
on a Pod or managing K8s resources. The duration it takes to execute these
functions depends on different factors like the type of operations, the
commands defined, and the function arguments. We can roughly divide the
function execution into 3 steps.

- Prerequisites - includes steps to perform before running actual operations,
  e.g., setting up env, preparing Pod specs, creating the K8s resources, and
  waiting for them to be ready.
- Execution - this is the step where the function performs its operations.
- Cleanup - operations like deleting pods or cleaning up resources.

Since there is no standard way to check the progress of these functions, we can
divide the progress equally into 3 giving each step equal weightage, i.e.,
33.33%. For example, once the prerequisite step is completed, the progress can
be set to 33.33%, and 66.66% once the specified operations are completed.

A few functions like `ScaleWorkload` may not have any prerequisite step. In
that case, the progress can be set to 0 till the operation completes.

### Updating ActionSet `status`

The following changes are required to update the ActionSet `status`.

- [TrackActionProgress](https://github.com/kanisterio/kanister/blob/master/pkg/progress/action.go#L47)
  function is currently responsible for periodically updating the progress of an
  action in the ActionSet resource `status`.
  
  Refactor `TrackActionsProgress` to remove the weight-based progress calculation
  and use the `Progress` interface implemented by functions.
  
  Pass an additional `Phase` parameter to the `TrackActionsProgress`` that can be
  used to get the progress of the Kanister function being executed.

```
+                               go func() {
+                                       // progress update is computed on a best-effort basis.
+                                       // if it exits with error, we will just log it.
+                                       if err := progress.TrackActionsProgress(ctx, c.crClient, as.GetName(), as.GetNamespace(), p); err != nil {
+                                               log.Error().WithError(err)
+                                       }
+                               }()
                                output, err = p.Exec(ctx, *bp, action.Name, *tp)

```

- Implement `Progress()` function on the `Phase`` type to return the progress
  of the Kanister function.

```
+func (p *Phase) Progress() (crv1alpha1.PhaseProgress, error) {
+       return p.f.Stats()
+}
```

- Use the progress of the current function to compute the average values and
  update the `Phase.ProgressPercent` and `Action.ProgressPercent` in the
  ActionSet `status`.
