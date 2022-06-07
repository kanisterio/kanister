# Blueprint And Phase Progress Tracking

## Problem Statement

When an actionset triggers a long-running task like the `CopyVolumeData`
function, the only way to determine any sort of progress is to gain direct access
to the logs and events of the task pod. This may not always be feasible as the
user may not have the appropriate RBAC permissions to access these subresource
endpoints in the pod's namespace. If the export operation takes a long time,
it's also possible that the pod might be terminated prematurely without leaving
behind any traces of logs to indicate how far along the task was.

Being able to persist progress data will be very helpful for both live reporting
of task progress, as well as future retrospection (e.g., the latency of every
phase of an action).

## Proposed Solution

### Summary

The `ActionSet` CRD's  `status` subresource will be updated with new fields to
communicate the blueprint and phases' progress status to the user. The
`kube.ExecOutput()` and `kube.Task()` interfaces will be updated to accept a new
progress I/O writer.

### Assumptions And Constraints

* The progress computation should not compromise the main data protection task's
latency nor lead to resource contention. Progress computation will be performed
on a best-effort basis, where it may be de-prioritized with no guarantee on the
accuracy of its result, or skipped entirely in the event of resource contention.

### Changes To API

The blueprint's overall progress will be reported under the new
`status.progress` field. The progress of each phase will be included in the
phase's subsection as `status.actions[*].phases[*].progess`.

For example,

```yaml
status:
  progress:
    percentCompleted: 50.00 # 1 out of 2 actions are completed
    lastTransitionTime: 2022-04-06 14:23:34
  actions:
  - blueprint: my-blueprint
    name: action-00
    phases:
    - name: echo
      state: completed
      progress:
        percentCompleted: 100.00
        lastTransitionTime: 2022-04-06 14:13:00
    - name: echo
      state: completed
      progress:
        percentCompleted: 100.00
        lastTransitionTime: 2022-04-06 14:23:34
    name: action-01
    phases:
    - name: echo
      state: pending
      progress:
        percentCompleted: 30.00
        lastTransitionTime: 2022-04-06 14:30:31
  state: pending
```

### Progress Tracking

Since progress tracking may not be meaningful for short-lived tasks, we will
limit the initial implementation effort to the following Kanister Functions
which normally are used to invoke long-running operations:

* `BackupData`
* `BackupDataAll`
* `RestoreData`
* `RestoreDataAll`
* `CopyVolumeData`
* `CreateRDSSnapshot`
* `ExportRDSSnapshotToLocation`
* `RestoreRDSSnapshot`

#### Blueprint Progress

Initially, the blueprint overall progress is computed by checking the number of
completed actions against the total number of actions:

```
blueprint_progress_percent = num_completed_actions / total_num_actions * 100
```

In subsequent implementation, the computation alogrithm can be updated to assign
more weight to phases with long-running operations. It's also possible to post
periodic progress updates using an exponential backoff mechanism as long as the
underlying phases are still alive.

When the first action of the blueprint is started, the blueprint progress will be
updated to 10%, instead of keeping it at 0%. This will help to distinguish
between in-progress first action from an inactive first action.

The blueprint progress will only be set to 100% after all the actions completed
without failures. The blueprint progress should never exceed 100%.

#### Phase Progress

As each phase within a blueprint may involve executing different commands
producing different outputs, this design proposes a phase progress tracking
interface that can use different "trackers" to map command outputs to numeric
progress status.

Some example trackers include ones that can track progress by:

* checking the number of uploaded bytes against estimated total bytes
* checking the duration elapsed against the estimated duration to complete the
operation
* parsing the log outputs for milestone events to indicate the 25%, 50%, 75% and
100% markers

✍️ Currently, Kanister Functions do not utilize Kopia to perform their
underlying work. Once the work to integrate Kopia into Kanister is completed,
we can extract the progress status directly from the log outputs.

Here's a sample log output of the Kopia create snapshot function:

```sh
$ kopia snapshot create kanister
Snapshotting isim@pop-os:/home/isim/workspace/kanisterio/kanister ...
- 5 hashing, 4186 hashed (329.1 MB), 0 cached (0 B), uploaded 309.8 MB, estimated 2 GB (16.3%) 3m38s left
```

Kanister Functions that are currently using Restic already have a set of library
functions that can be used to extract progress status from the Restic logs. See
e.g., the `restic.SnapshotStatsFromBackupLog()` function in
[`pkg/restic/restic.go`](https://github.com/kanisterio/kanister/blob/c5acaac88a60c22faeadd59c49b20942f662331d/pkg/restic/restic.go#L362)

Since all the long-running functions rely on the `KubeExec` and `KubeTask`
functions, most implementation changes will be done on these two functions.

✍️ Defer phase should also included in the phase-level progress tracking.

Here's an example code snippet of the proposed interface written in Go:

```go
// ./pkg/progress/phase
package phase

type ProgressTracker struct {
  t Tracker
  R Result
}

type Result struct {
  StatusInPercent chan string
  Err             chan error
}

func (pt *ProgressTracker) Write(p []byte) (n int, err error) {
  if err := pt.t.Compute(string(p), pt); err != nil {
    return len(p), err
  }

  return len(p), nil
}

func (pt *ProgressTracker) Result() <-chan string {
  return pt.R.StatusInPercent
}

func (pt *ProgressTracker) Err() <-chan error {
  return pt.R.Err
}

type Tracker interface {
  Compute(cmdOutput string, p *ProgressTracker) error
}
```

This is an example of what the client code would look like:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

bytesTracker := BytesTracker { totalNumBytes: 268435456 }
progressTracker := phase.NewProgressTracker(bytesTracker)
go func() {
  for {
    select {
    case <-context.Done():
       // handle context.Err()
       return
    case err := <-progressTracker.Err():
       // handle err
       return
    case r := <-progressTracker.Result():
       // update the actionset's status with r.
       // might need some more refactoring in order to return
       // this to ./pkg/controller/controller.go
    }
  }
}()

out := io.MultiWriter(os.Stdout, progressTracker)
kube.ExecOutput(cli, namespace, pod, container, command, in, out, errw)
```

Here's an sample tracker that computes progress status based on the amount of
data uploaded vs. the total amount of data:

```go
var _ Tracker = (*BytesTracker)(nil)

type BytesTracker struct {
  totalNumBytes int64
}

func (b BytesTracker) Compute(cmdOutput string, t *ProgressTracker) error {
  totalNumBytesUploaded, err := parse(cmdOutput)
  if err != nil {
    return err
  }

  pt.R.StatusInPercent = totalNumBytesUploaded/totalNumBytes * 100
  return nil
}
```

#### Error Handling

If a phase failed, the progress tracking will cease immediately. The last
reported progress will be retained in the actionset's `status` subresource.

## Test Cases

New unit tests to be added to the new `progress` package to cover blueprint
progress, and phase progress calculated with a dummy tracker:

* Blueprint with single phase:
  * Completed successfully - assert that blueprint and phase progress are at 100%
  * Failed to finish - assert that blueprint progress and phase progress are at 10%
* Blueprint with multiple phases:
  * Completed successfully - assert that blueprint and phase progress are at 100%
  * Failed to finish at different phases - assert that progress calculation is correct
