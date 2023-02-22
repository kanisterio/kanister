# Problem Statement

Kanister actions are triggered by creating ActionSet resource. The status field in ActionSet CR represents the status of Kanister operation. The action completion percentage is set by calculating the weights given on phases ([https://github.com/kanisterio/kanister/blob/master/design/progress-tracking.md](https://github.com/kanisterio/kanister/blob/master/design/progress-tracking.md) ).

The goal of this proposal is

- Have progress tracking per phase
- Improve action progress tracking mechanism by removing hard coded weights

## High Level Design

### Changes in ActionSet status field

In order to inform progress of each phase execution while performing action, the progress field can be added to status.phase

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

### Action progress tracking

Action can consists of multiple phases. These all phases can take consume different duration for completion based on the operations it performs. All phases for the action can be given equal weight-age and to calculate action progress completion %, average of all phase completion %ge can be calculated

```
% completion of action = sum(% Phase completion of all the phases)/ no. of phases
```

### Phase progress tracking

Each function is designed to perform different operations in different ways. Since the operation execution is specific to function, each function can be different ways to calculate and provide info about the progress. E.g datamover functions can leverage kopia stats to track progress. So it makes sense to delegate progress tracking to function implementation.

Each Kanister function implements Progress interface. The Kanister function implementation calculates and returns the progress stats.

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

The PhaseProgress.ProgressPercent would be the of progress the functions it consists of.

The following is the example ActionSet status of BP action consists of 3 phases. Out of which first phase is completed and second one is running

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

### Kanister functions progress tracking

On high level we can divide the Kanister functions into two groups

### Datamover Kanister functions

Kanister data mover functions (like BackupData, CopyVolumeData, RestoreData, etc) use Kopia to snapshot filesystem and move data to/from objects.

While snapshotting, kopia provides the info about progress on the terminal

```
$ kopia --log-level=error --config-file=/tmp/kopia-repository.config --log-dir=/tmp/kopia-log snapshot --progress-update-interval=5s /data
 / 1 hashing, 118 hashed (546.9 MB), 0 cached (0 B), uploaded 0 B, estimated 4.5 GB (12.1%) 35s left
 * 0 hashing, 136 hashed (1.4 GB), 0 cached (0 B), uploaded 244 B, estimated 4.5 GB (30.1%) 30s left
```

The command output can be parsed to get the progress metadata.

### Non-datamover Kanister functions

Non-datamover Kanister functions like `KubeTask`, `KubeExec`, `KubeOps`, `ScaleWorkload`, etc allow users to perform operations like executing scripts on a Pod or managing K8s resources. The duration it takes to execute these functions depends on different factors like the type of operations, function arguments, and types of commands listed in BP in the case of KubeExec or KubeTask functions. We can roughly divide these function execution into 3 steps.

- Prerequisites - which include steps to perform before running actual operations like setting up env, preparing Pod specs, creating the K8s resources and waiting for them to be ready.
- Execution - This is the step where the function performs operations.
- Cleanup - Operations like deleting pods or cleaning up resources

Since there is no standard way to check the progress of these functions, we can divide the progress equally into 3 giving each step equal weightage i.e 33.33%. E.g once the prerequisite step is completed, the progress can be set to 33.33%. And 66.66% once the specified operations are completed.

A few functions like `ScaleWorkload` may not have any prerequisite step. In that case, the progress can be set to 0 till the operation completes.

### Updating actionset status

[TrackActionProgress](https://github.com/kanisterio/kanister/blob/master/pkg/progress/action.go#L47) function is responsible for periodically updating progress for the action in ActionSet resource status.

Refactor TrackActionsProgress to remove weight based progress calculation to use Progress interface implemented by functions.

Pass additional parameter Phase to the TrackActionsProgress which can be used to get progress of the function in execution.

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

Implement Progress() function on Phase type to return Progress of the Kanister function

```
+func (p *Phase) Progress() (crv1alpha1.PhaseProgress, error) {
+       return p.f.Stats()
+}
```

Once we have information about Progress of current function, the Phase ProgressPercent and Action ProgressPercent in the ActionSet status can be updated by computing the average.
