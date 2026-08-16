# CSI Backup Functions — What Changed and Why

**Branch:** `K10-35474-adding-pvc-volume`  
**Author:** Anand Tiwari  
**Date:** May 2026

---

## Background

This branch integrates `backup-csi-driver` with Kanister. The driver provides a FUSE-mounted
volume backed by Kopia + S3. Blueprint phases write a `pg_dumpall` into `/backup`; the CSI driver
handles deduplication, encryption, and upload to S3 automatically.

During development four new functions were written. Before completing the work, each was audited
against the existing Kanister function registry. Two were found to duplicate existing functions
and were deleted. Two were genuinely new. One existing function was extended.

The sections below explain every decision so any reviewer can follow the reasoning.

---

## Where "Volume Populator" fits

The CTO used the term "Volume Populator." Here is exactly what that means in this solution
and where it sits in the flow.

### Current implementation — standard CSI restore-from-snapshot

`backup-csi-driver` today uses the standard CSI mechanism that has existed since Kubernetes
1.17: a PVC with `spec.dataSource: VolumeSnapshot` causes Kubernetes to call the CSI
driver's `CreateVolume` RPC with a `VolumeContentSource` (snapshot source).

When your CTO says "Volume Populator" in the context of today's code, they mean **the CSI
driver itself acting as the thing that populates a new volume with existing snapshot data**
— which it does through the standard CSI snapshot restore path.

### Future direction — Kubernetes Volume Populator API

The CTO has indicated the plan is to migrate to the **Kubernetes Volume Populator API**
(stable since Kubernetes 1.24). This is a different, more explicit mechanism:

- The PVC uses `spec.dataSourceRef` (not `spec.dataSource`) pointing to a **custom resource**
  (e.g. a `KopiaSnapshot` CR), not a `VolumeSnapshot`
- A dedicated **Volume Populator controller pod** watches for PVCs with that custom
  `dataSourceRef` kind and actively pulls the data before the PVC is marked `Bound`
- The PVC only reaches `Bound` after the populator has finished materialising the data —
  which is a stronger guarantee than today

**Key difference from today:**

| | Current (CSI restore-from-snapshot) | Future (Volume Populator API) |
|---|---|---|
| PVC field | `spec.dataSource: VolumeSnapshot` | `spec.dataSourceRef: KopiaSnapshot` (custom) |
| Who populates | CSI driver at `NodePublishVolume` (lazy, on pod start) | Dedicated controller pod (eager, before pod starts) |
| When data is ready | When `NodePublishVolume` completes inside the pod | When PVC reaches `Bound` |
| `waitForBound` semantics | Bound = volume object exists; data fetched later | Bound = data is fully available |
| Extra component needed | No (CSI driver handles it) | Yes — a Volume Populator controller must be deployed |

**What this means for the Kanister functions:**

The `RestoreCSISnapshot` function would need a change to the PVC manifest it creates —
`spec.dataSource` would be replaced with `spec.dataSourceRef` pointing to a custom resource
type. Everything else (args, `waitForBound`, outputs) stays identical. The `waitForBound`
polling becomes even more important in the future model because `Bound` will be the definitive
signal that data is ready, not just that the volume object was created.

### Step-by-step: what happens when `RestoreCSISnapshot` runs

```
Blueprint calls RestoreCSISnapshot
  │
  ▼
Kanister creates a PVC:
  storageClassName: kopia-restore
  spec.dataSource:
    kind: VolumeSnapshot
    name: snapshot-kanister-job-z6qrc-...
  │
  ▼
Kubernetes external-provisioner sees the PVC + storageClass → calls CSI CreateVolume
  ├── req.VolumeContentSource.Snapshot.SnapshotId = "snapshot-kanister-job-z6qrc-..."
  ▼
CSI driver CreateVolume (controller.go:243):
  ├── Reads VolumeContentSource → snapshotID != ""
  ├── Forces volumeContext["mode"] = "restore"        ← KEY: no data moved yet
  ├── Stores volumeContext["snapshotId"] = snapshotID ← just a label on the volume
  └── Returns CreateVolumeResponse immediately
  │
  ▼
PVC status: Pending → Bound  (waitForBound: true waits for this)
  │
  ▼
KubeTask pod starts, Kubernetes calls CSI NodePublishVolume (node.go:166)
  ├── Reads mode = "restore" from volumeContext
  ├── Calls mountRestoreVolume() (node.go:582):
  │     ├── kopiaClient.Connect(ctx)          ← opens Kopia repo → connects to S3
  │     ├── kopiaClient.GetSnapshotDirectory(ctx, snapshotID)
  │     │     └── fetches snapshot root from S3 into memory
  │     └── mounter.MountRestore(ctx, targetPath, dir)
  │           └── FUSE-mounts the snapshot read-only at /restore inside the pod
  └── Returns NodePublishVolumeResponse
  │
  ▼
Pod reads /restore/full.sql → psql applies the dump → restore complete
```

### The critical point: data is NOT pulled during PVC creation

`CreateVolume` does nothing to S3. It only encodes the snapshot ID as a label on the
volume object. The actual Kopia connection and data retrieval happens in `NodePublishVolume`
when the KubeTask pod starts. This is why:

- `waitForBound: true` is the right signal to proceed — once Bound, the volume exists and
  the pod can be scheduled.
- There is no additional "data ready" signal needed — by the time the pod's first command
  runs, `NodePublishVolume` has already completed and `/restore` is fully mounted.

### Why `waitForBound` matters in our extension to `RestoreCSISnapshot`

Before our change, `RestoreCSISnapshot` returned immediately after issuing the PVC create
call — before the PVC was even Bound. If the next blueprint phase (KubeTask) was scheduled
before the PVC reached Bound, the pod would fail to start with `volume not yet available`.
Adding `waitForBound: true` (via the optional arg we added) ensures the PVC is Bound before
Kanister hands off to the next phase.

---

## Existing functions that are used unchanged

| Function | File | What it does for this integration |
|---|---|---|
| `CreateCSISnapshot` | `pkg/function/create_csi_snapshot.go` | Triggers a VolumeSnapshot on a PVC and waits for `readyToUse: true`. Used in the KubeExec backup blueprint after `pg_dumpall` finishes writing to the pre-mounted backup PVC. |
| `KubeExec` | `pkg/function/kube_exec.go` | Exec into the running Postgres pod to run `pg_dumpall`. |
| `KubeTask` | `pkg/function/kube_task.go` | Run a fresh pod (with ephemeral CSI volume or restore PVC mounted) to do backup or restore. |
| `DeleteCSISnapshot` | `pkg/function/delete_csi_snapshot.go` | Delete a VolumeSnapshot object. Available for lifecycle cleanup if needed by a policy. |

---

## What was written, then deleted — and why

### ~~`CreateVolumeSnapshot`~~ → use `CreateCSISnapshot` instead

We initially wrote `CreateVolumeSnapshot` because the existing `CreateCSISnapshot` arg names
looked different in the docs. After reading the source, it turned out the logic was identical:

| | `CreateVolumeSnapshot` (deleted) | `CreateCSISnapshot` (existing, used) |
|---|---|---|
| PVC arg | `pvcName` | `pvc` |
| Snapshot name | auto-generated `<pvc>-snapshot-<rand5>` | same (`defaultSnapshotName()`) |
| Snapshot class arg | `snapshotClass` | `snapshotClass` |
| Waits for `readyToUse` | yes, hardcoded | yes, hardcoded |
| Core call | `snapshotter.Create()` | `snapshotter.Create()` — same function |
| Outputs | `volumeSnapshotName`, `volumeSnapshotNamespace` | `name`, `pvc`, `namespace`, `restoreSize`, `snapshotContent` |

The only differences were the arg name (`pvcName` vs `pvc`) and the output keys.
The underlying Go function called was the same `snapshotter.Create()`.

**Why it was deleted:** Two functions in the registry doing the exact same thing diverge over
time and confuse blueprint authors. `CreateCSISnapshot` is richer (more outputs). Blueprints
using `CreateVolumeSnapshot` were updated to use `CreateCSISnapshot` with `pvc:` instead of
`pvcName:`.

---

### ~~`CreateVolumeFromSnapshot`~~ → use `RestoreCSISnapshot` (extended) instead

We initially wrote `CreateVolumeFromSnapshot` to create a restore PVC from a VolumeSnapshot
and wait until the PVC was `Bound`. The existing `RestoreCSISnapshot` did the same thing
(creates a PVC with `spec.dataSource` pointing to a VolumeSnapshot) but had two gaps:

**Gap 1 — no `waitForBound` option:**  
`RestoreCSISnapshot` created the PVC and immediately returned without waiting. The next
blueprint phase (the KubeTask that mounts the PVC) would fail because the PVC was still
`Pending` while the CSI Volume Populator pulled data from S3.

**Gap 2 — no outputs:**  
`RestoreCSISnapshot` returned `nil, nil`. The downstream KubeTask phase needed
`{{ .Phases.createRestorePVC.Output.pvcName }}` to know which PVC to mount. Without this
output, the restore blueprint could not reference the PVC at all.

**What we did instead of keeping both:**  
Extended `RestoreCSISnapshot` with two additions:

1. Optional arg `waitForBound` (default `false` — existing callers are unaffected).
2. Outputs `pvcName` and `pvcNamespace` — matching what `CreateVolumeFromSnapshot` produced,
   so blueprint template references required no change.

The `waitForPVCBound()` helper that polls until `status.phase == Bound` was moved from
`create_volume_from_snapshot.go` into `restore_csi_snapshot.go`.

**Why `CreateVolumeFromSnapshot` was deleted:** After extending `RestoreCSISnapshot`, the two
functions had identical capability. Keeping both would mean two ways to do the same thing.
`RestoreCSISnapshot` is the established name in the Kanister ecosystem.

---

## What is genuinely new — and why no existing function covers it

### `DeleteVolume`

**File:** `pkg/function/delete_volume.go`

**Team question:** *"Why not use `DeleteCSISnapshot` or another existing delete function?"*

`DeleteCSISnapshot` deletes a **VolumeSnapshot object**. `DeleteVolume` deletes a **PVC**.
These are different Kubernetes resources. The restore workflow creates a temporary restore PVC
that needs to be cleaned up after the SQL dump is applied. There is no existing Kanister
function that deletes a PVC. Without this function, orphaned PVCs would accumulate in the
application namespace after every restore.

`DeleteVolume` calls `kubeCli.CoreV1().PersistentVolumeClaims().Delete()` and treats
`NotFound` as success, so it is safe to re-run if a restore ActionSet is retried.

**Args:** `pvcName` (required), `pvcNamespace` (required)  
**Outputs:** none

---

### `WaitForEphemeralSnapshot`

**File:** `pkg/function/wait_for_ephemeral_snapshot.go`

**Team question:** *"Why not use the existing `Wait` or `WaitV2` functions?"*

`Wait` and `WaitV2` are general condition-check functions. They evaluate a condition expressed
as a Kubernetes object status check — e.g. "wait until this object's `.status.field` equals
this value". They work against a **known, named object**.

`WaitForEphemeralSnapshot` solves a completely different problem: the snapshot does not exist
yet and its name is not known in advance.

Here is why this function is needed and why nothing else covers it:

**The problem — asynchronous snapshot creation:**  
When a `KubeTask` pod with an ephemeral inline CSI volume exits, the CSI driver's
`NodeUnpublishVolume` fires on the node and calls `createEphemeralSnapshot()`. This happens
**after the pod has already terminated** — asynchronously, on the node, outside of Kanister's
control. By the time Kanister advances to the next blueprint phase, the VolumeSnapshot does
not exist yet. `Wait` and `WaitV2` require a named object to watch; they cannot help here.

**What `WaitForEphemeralSnapshot` does:**  
Polls `VolumeSnapshot.List()` on the namespace until a matching snapshot appears, using three
filters to avoid picking up the wrong snapshot:

| Filter | Purpose |
|---|---|
| `creationTimestamp > after` | Ignores snapshots from previous backups. `after` defaults to `{{ .Time }}` (ActionSet start time) — no blueprint config needed. |
| `spec.source.volumeSnapshotContentName != ""` | Identifies pre-provisioned (ephemeral) snapshots. PVC-sourced snapshots have `spec.source.persistentVolumeClaimName` set instead. This is the key structural difference the CSI driver leaves in the object. |
| `name` has prefix `snapshot-<podName>-` (optional) | Narrows to the snapshot created by a specific pod when `podName` is passed. The CSI driver encodes the pod name in the snapshot name. |

Poll interval: 5 seconds. Timeout: 5 minutes. Returns a wrapped error on timeout.

**Args:** `namespace` (required), `after` (optional, RFC3339), `podName` (optional)  
**Outputs:** `volumeSnapshotName`, `volumeSnapshotNamespace`

---

## Net change summary

| File | Status | Reason |
|---|---|---|
| `pkg/function/create_volume_snapshot.go` | **Deleted** | Exact duplicate of `CreateCSISnapshot` (same `snapshotter.Create()` call, only arg name differed) |
| `pkg/function/create_volume_from_snapshot.go` | **Deleted** | Functionality absorbed into extended `RestoreCSISnapshot` |
| `pkg/function/restore_csi_snapshot.go` | **Extended** | Added optional `waitForBound` arg (default `false`), added `pvcName`/`pvcNamespace` outputs, absorbed `waitForPVCBound()` helper |
| `pkg/function/delete_volume.go` | **New** | No existing function deletes a PVC |
| `pkg/function/wait_for_ephemeral_snapshot.go` | **New** | No existing function can discover an asynchronously-created, name-unknown VolumeSnapshot |

---

## Blueprint arg migration reference

For anyone updating blueprints from the old function names:

**Backup phase:**
```yaml
# Before (deleted function)
- func: CreateVolumeSnapshot
  args:
    pvcName: postgres-backup-vol-pvc
    namespace: "{{ .StatefulSet.Namespace }}"
    snapshotClass: kopia-snapshot-class

# After (existing function, just rename pvcName → pvc)
- func: CreateCSISnapshot
  args:
    pvc: postgres-backup-vol-pvc
    namespace: "{{ .StatefulSet.Namespace }}"
    snapshotClass: kopia-snapshot-class
```

**Restore phase:**
```yaml
# Before (deleted function)
- func: CreateVolumeFromSnapshot
  args:
    snapshotName: "{{ .ArtifactsIn.backupVolumeSnapshot.KeyValue.volumeSnapshotName }}"
    snapshotNamespace: "{{ .ArtifactsIn.backupVolumeSnapshot.KeyValue.volumeSnapshotNamespace }}"
    pvcNamespace: "{{ .StatefulSet.Namespace }}"
    storageClass: kopia-restore
    storageSize: 5Gi
    waitForBound: true

# After (existing function, extended — output keys pvcName/pvcNamespace are the same)
- func: RestoreCSISnapshot
  args:
    name: "{{ .ArtifactsIn.backupVolumeSnapshot.KeyValue.volumeSnapshotName }}"
    namespace: "{{ .StatefulSet.Namespace }}"
    pvc: postgres-kubetask-restore-pvc    # must be explicit; no auto-generation
    storageClass: kopia-restore
    restoreSize: 5Gi                       # storageSize → restoreSize
    waitForBound: true
```

Downstream phase references (`{{ .Phases.X.Output.pvcName }}`) are unchanged because
`RestoreCSISnapshot` now outputs the same keys that `CreateVolumeFromSnapshot` did.
