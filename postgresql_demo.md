# Postgres Backup/Restore Demo — `KubeTaskWithBackupPVC` + `KubeTaskWithRestorePVC`

> Presenter's walkthrough. Each section has a **NARRATIVE** (what to say) and
> **COMMANDS** (what to run, ideally in a side-by-side terminal so the audience
> sees state changing in real time).

---

## 0. Pre-demo (10 minutes before)

Quietly verify the cluster is in a clean, demo-ready state. **Don't show this to the audience.**

```bash
# 1. AWS STS creds still valid (they expire every ~12h)?
kubectl -n backup-csi-driver logs daemonset/backup-csi-driver --tail=20 2>&1 | grep -iE 'token has expired|error' | head -3
# If you see "token has expired", refresh kopia-storage-credentials secret + bounce CSI driver pods BEFORE the demo.

# 2. Kanister controller on the expected image
kubectl -n kasten-io get deployment kanister-svc -o jsonpath='{.spec.template.spec.containers[*].image}{"\n"}'
# Expect: vdckastenacrdev.azurecr.io/kanister/controller:20260529-1826

# 3. Workload exists + annotated for our blueprint
kubectl -n demo-postgres get statefulset pg-postgresql -o jsonpath='{.metadata.annotations.kanister\.kasten\.io/blueprint}{"\n"}'
# Expect: postgres-pvc-backup-action-blueprint

# 4. Blueprint + policy live
kubectl -n kasten-io get blueprint postgres-pvc-backup-action-blueprint
kubectl -n kasten-io get policy postgres-pvc-backupaction-test

# 5. Database has data
PGPW=$(kubectl -n demo-postgres get secret pg-postgresql -o jsonpath='{.data.postgres-password}' | base64 -d)
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c 'SELECT count(*) FROM customers;'"
# Expect: 20

# 6. Namespace clean — no orphans from earlier runs
kubectl -n demo-postgres get pvc,pods
# Expect: only data-pg-postgresql-0 + pg-postgresql-0
```

If anything's off, fix it now — not during the demo.

---

## 1. Set the scene (2 min)

**NARRATIVE:**
> "We have a Postgres database running in `demo-postgres`. It's a real workload — a StatefulSet using a standard Bitnami chart. Inside, we have a `testdb` database with a `customers` table — 20 customer records.
>
> Today I'm going to show you how our two new Kanister functions —
> `KubeTaskWithBackupPVC` and `KubeTaskWithRestorePVC` — handle the full
> backup-and-restore lifecycle in **one phase each**, with **zero K10 patches**, fully integrated with Kasten K10."

**COMMANDS (show on screen):**

```bash
# The workload
kubectl -n demo-postgres get statefulset,pod,pvc

# The data we'll be protecting
PGPW=$(kubectl -n demo-postgres get secret pg-postgresql -o jsonpath='{.data.postgres-password}' | base64 -d)
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c 'SELECT count(*) FROM customers;'"
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c 'SELECT * FROM customers LIMIT 3;'"

# The blueprint — show this is ALL the customer has to write (39 lines)
cat "/Users/an.tiwari/veeam products/feature-long-running-pvc/blueprints/postgres-pvc-backup-action-blueprint.yaml"
```

**Key talking points:**
- ONE phase for backup. ONE phase for restore.
- Output artifact carries only the snapshot **name**. Everything else (size, namespace, content) is read from the VolumeSnapshot CR itself on restore.
- `env.valueFrom.secretKeyRef` — the password never appears in any Kanister/K10 CR.

---

## 2. Watch terminal — open this on a side screen (will stay open the whole demo)

In a second terminal, start this watch and leave it visible:

```bash
kubectl -n demo-postgres get pvc,pods,volumesnapshot -w
```

This is the **single most important visualization** — the audience sees PVCs and snapshots being created and cleaned up by the function in real time.

---

## 3. Trigger the backup (5 min)

**NARRATIVE:**
> "I'll trigger the backup from the Kasten UI. Watch the second terminal — you'll see exactly what the function does, in order."

**STEPS:**
1. Open Kasten UI → Applications → `demo-postgres` → click `Run` on `postgres-pvc-backupaction-test` policy.
2. **As things happen on the watch terminal, narrate:**

| What appears on the watch | What it means |
|---|---|
| `pg-postgresql-backup-XXXXXX` PVC, status `Pending` → `Bound` | Function created a staging PVC (`<workload>-backup-<random6>`) with `kanister.io/staging-pvc=true` + workload-scoped labels |
| `kanister-backup-pvc-XXXXX` Pod, status `ContainerCreating` → `Running` | Function spawned a backup pod with the staging PVC mounted at `/backup`, runs `pg_dumpall` |
| Pod stays `Running` for ~30s+ after dump finishes | This is the **keep-alive** — function wraps the user command with `sleep`, holds the FUSE mount alive while the CSI driver finalizes the snapshot |
| `VolumeSnapshot` `pg-postgresql-backup-XXXXXX-snapshot-YYYYY` appears, `READYTOUSE=false` | Function called `CreateCSISnapshot` internally — the backup-csi-driver is now streaming data to kopia/S3 |
| `READYTOUSE=true` | Snapshot finalized in S3. Kopia archive complete. |
| Pod `kanister-backup-pvc-*` disappears | Function actively deleted the keep-alive pod (deferred call) — mount released |
| `pg-postgresql-backup-XXXXXX` PVC disappears | Function deleted the staging PVC (deferred call, LIFO after pod kill) — VolumeSnapshot survives via the class's `Retain` deletion policy |
| Namespace is clean — only `data-pg-postgresql-0` + workload pod remain | End state. The Kopia snapshot is your backup artifact. |

**While that's running, in a third terminal show the live ActionSet:**

```bash
# In another terminal, see Kanister's view of what's happening
kubectl -n kasten-io get actionset --sort-by=.metadata.creationTimestamp -o yaml | tail -80
```

Highlight:
- `spec.actions[*].name` is `backup` — K10 invoked our `actions.backup` blueprint action directly.
- `status.actions[*].phases[*].output` shows `volumeSnapshotName`, `restoreSize`, etc.
- `status.actions[*].artifacts.snapshotInfo` shows the rendered output artifact.

---

## 4. Prove the backup is real (2 min)

**NARRATIVE:**
> "The backup is done. I want to prove it's really there — let me show you the artifacts."

**COMMANDS:**

```bash
# 1. The VolumeSnapshot CR (kopia-backed)
kubectl -n demo-postgres get volumesnapshot --sort-by=.metadata.creationTimestamp -o wide | tail -3
kubectl -n demo-postgres get volumesnapshot <the-new-one> -o yaml | grep -E 'readyToUse|restoreSize|snapshotContent|driver'

# 2. The Kasten RestorePoint
kubectl -n demo-postgres get restorepoint --sort-by=.metadata.creationTimestamp | tail -3
# Pick the newest one — that's the artifact users restore from.

# 3. Show the ActionSet output (proof our function did its job)
kubectl -n kasten-io get actionset --sort-by=.metadata.creationTimestamp -o json 2>&1 | jq '.items[-1].status.actions[0].phases[0].output'
```

---

## 5. Destroy the data (1 min, the dramatic part)

**NARRATIVE:**
> "Now the scary part. I'm going to drop the `customers` table. This is the kind of thing that ruins your week if you don't have a backup."

**COMMANDS:**

```bash
PGPW=$(kubectl -n demo-postgres get secret pg-postgresql -o jsonpath='{.data.postgres-password}' | base64 -d)

# Before
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c '\dt'"
# Shows the customers table

# Drop
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c 'DROP TABLE customers;'"

# After
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c '\dt'"
# Did not find any tables.
```

> "Table is gone. testdb is empty. Let's restore it."

---

## 6. Trigger the restore (5 min)

**NARRATIVE:**
> "Restoring is exactly the same shape — one phase. Watch the same terminal."

**STEPS:**
1. Kasten UI → applications → `demo-postgres` → pick the RestorePoint we just created → click Restore.
2. **On the watch terminal, narrate as things happen:**

| What appears | What it means |
|---|---|
| `pg-postgresql-restore-XXXXXX` PVC, `Pending` → `Bound` | Function called `restoreCSISnapshot` internally — created a new PVC with `dataSource: VolumeSnapshot/<name from artifact>`. CSI driver re-hydrates from kopia. |
| `kanister-restore-pvc-XXXXXX` Pod, `ContainerCreating` → `Running` | Function spawned restore pod with the restored PVC mounted at `/restore` (read-only) |
| Inside the pod, `psql` runs the dump | `psql -U postgres < /restore/full.sql` against the live workload — CREATE TABLE, INSERTs flow in |
| Pod `Completed` / disappears | Restore command finished cleanly |
| `pg-postgresql-restore-XXXXXX` PVC disappears | Function's deferred cleanup deleted the staging PVC. The original VolumeSnapshot still survives — usable for future restores. |
| Namespace clean again | End state, identical to the start. |

---

## 7. Prove the restore worked (1 min)

**NARRATIVE:**
> "And here's the proof: data's back."

```bash
PGPW=$(kubectl -n demo-postgres get secret pg-postgresql -o jsonpath='{.data.postgres-password}' | base64 -d)

# Tables
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c '\dt'"

# Row count
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c 'SELECT count(*) FROM customers;'"

# Sample rows
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c 'SELECT id, name FROM customers ORDER BY id;'"
```

Expect: 20 customer records restored.

---

## 8. Wrap-up: what we just saw (2 min)

**NARRATIVE:**
> Recap, in the audience's language:
> - **One-phase backup, one-phase restore.** No `RestoreCSISnapshot` boilerplate, no `KubeOps delete`, no `WaitV2`.
> - **The function owns the full lifecycle**: creates the staging PVC, runs the dump, holds the mount during snapshot, deletes the pod, deletes the PVC. All deferred-cleanup-safe — runs even on error.
> - **Restored from a kopia snapshot in S3** — not from a local PVC. Survives cluster destruction.
> - **No K10 source patches** — the `actions.backup`/`actions.restore` blueprint mechanism is K10-native; K10 sees our blueprint and skips its own snapshot pipeline.
> - **Real artifact plumbing** — `outputArtifacts.snapshotInfo.name` from backup is `ArtifactsIn.snapshotInfo.KeyValue.name` on restore. Native Kanister, no label-scanning hacks.
> - **39-line blueprint.** That's the whole customer-facing surface.

---

## Honest discussion — what could go wrong (be prepared for these questions)

These are real issues the function has — be upfront about them rather than getting caught off-guard.

### Operational risks

1. **AWS STS token expiry**
   `kopia-storage-credentials` Secret carries short-lived STS tokens that expire every ~12h. If they expire mid-backup, the FUSE mount hangs and the function times out at `WaitForPodReady` (~15 min). Function returns error; deferred cleanup kicks in.
   *Mitigation today:* refresh creds via script/cron. *Production path:* IAM Role for ServiceAccount (IRSA) — eliminates manual rotation entirely.

2. **Orphan state on hard crashes**
   If the controller pod itself crashes (OOM, eviction) BEFORE the defers run, you can leak a staging PVC + keep-alive pod. The next run is unaffected because PVC names are unique-random, but cluster storage accumulates.
   *Mitigation:* a janitor cronjob that label-selects orphans older than the function timeout.

3. **5Gi default `restoreSize`**
   When the CSI driver doesn't populate `VolumeSnapshot.status.RestoreSize` (kopia / streaming drivers leave it nil), the function falls back to 5Gi. For SQL dumps that's plenty; for raw block-level kopia archives of large workloads this could be too small.
   *Mitigation:* blueprint author explicitly sets `restoreSize: 50Gi` if expecting larger payloads.

4. **`keepPodAliveForSnapshot` shell-form restriction**
   The keep-alive wrapper only supports `bash|sh -c <script>` command form (so we can compose a sleep around the user script). Direct exec form `["pg_dumpall", "-U", "postgres"]` would be rejected.
   *Mitigation:* always use shell form in blueprints. Documented in arg comment.

5. **`workload PVC` not restored under actions.backup pattern**
   In `actions.backup` mode, K10 skips its snapshot phase entirely. That means it doesn't track the **workload's own** PVC (`data-pg-postgresql-0`) — so on restore, K10 recreates the StatefulSet fresh with an empty PVC. Our function then replays the dump into that fresh PVC. This is correct behavior, but if a customer expected K10 to also restore the workload's block-level state independently of our dump, that's a surprise.
   *Mitigation:* document explicitly — the dump IS the backup; the workload PVC is just a runtime detail.

6. **Snapshot class `deletionPolicy`**
   Our function deletes the staging PVC at function exit. The VolumeSnapshot must have a `VolumeSnapshotClass` with `deletionPolicy: Retain` — otherwise deleting the VS deletes the underlying data. We rely on the cluster admin having configured this correctly.
   *Mitigation:* document as a prerequisite in the function's docs/functions.md entry.

### Function design limits (honest)

7. **No live-data backup**
   Function does logical dump (`pg_dumpall`) → not transactionally consistent at the block level. For multi-second SQL dumps, you can get inconsistencies if writes are happening. For consistent backups, use `pg_dumpall --serializable-deferrable` or pause writes.

8. **Concurrency cap**
   Two simultaneous backups of the same workload would produce two staging PVCs with different random suffixes — no collision. But the CSI driver may serialize FUSE sessions per workload node, slowing things down.

9. **Function output artifact is consumed by Kanister templating**
   The full output map (`pvcName`, `namespace`, `volumeSnapshotName`, etc.) lives in the ActionSet's status. Anyone with `get actionset` permission can see PVC + snapshot names. **No secrets** are emitted, but operational metadata is visible.

### Questions you might get and good answers

**Q: "Why not just use `CreateCSISnapshot` + `KubeOps` separately?"**
A: You can. We did at first (3-phase blueprint). Absorbing them into the function gives: (1) one-phase blueprint, (2) deferred cleanup that fires on every exit path including errors, (3) keep-alive coupled to snapshot lifecycle so we kill the pod the moment the snapshot is terminal. The 3-phase version had to be hand-orchestrated by the blueprint author.

**Q: "Does it work without K10 / Kasten?"**
A: Yes. Set `takeSnapshot=true` (default) + a `snapshotClass` arg. Function works with any CSI driver supporting VolumeSnapshots — Azure Disk, AWS EBS, GCE PD, OpenEBS, longhorn. Set `takeSnapshot=false` and it works on any storage class period (no snapshot needed).

**Q: "What if the snapshot fails mid-creation?"**
A: Function blocks on `WaitOnReadyToUse` until terminal state. Three terminal states: `readyToUse=true`, `status.error` set (driver reported failure), or `ctx.Done()` (cancellation/timeout). All three release the wait. Deferred cleanups run. PVC is deleted regardless.

**Q: "What if my workload restarts during backup?"**
A: Backup pod is independent of the workload pod — they only share the workload's secret. Workload restart doesn't affect the backup pod or the staging PVC. The pg_dumpall connection might break though — function reports the error loudly.

**Q: "Multi-tenant / shared namespace?"**
A: Each workload's staging PVC is labelled with `workload-name` + `workload-namespace`. Function defers capture local-scope PVC names — they only delete the PVC THIS execution created. Multi-workload safe.

---

## Demo failure recovery — what to do if something goes wrong on stage

1. **CSI mount hangs (`ContainerCreating` for 2+ min)** — likely expired AWS creds. *Don't try to fix live.* Switch to: "and this is an example of the loud-error path — let me show what happens next" → wait for the 15-min timeout to demonstrate clean error handling. Or skip restoration demo, show prior successful artifact.

2. **Snapshot never goes `readyToUse=true`** — explain it'd time out at `WaitOnReadyToUse`'s ctx deadline; deferred cleanup still kicks in. Show the deferred behavior on a separate (dry-run) failed run if you have one in history.

3. **Orphan from earlier session** — show your janitor command:
   ```bash
   kubectl -n demo-postgres delete pod -l kanister.io/keep-alive-for-snapshot=true --grace-period=0 --force
   kubectl -n demo-postgres delete pvc -l kanister.io/staging-pvc=true --wait=false
   ```
   Mention you have a CronJob for this in production (or plan to).

---

## After the demo — cleanup state

Reset for the next presenter:

```bash
# Optional: restore data to a known state if anyone dropped something else during Q&A
PGPW=$(kubectl -n demo-postgres get secret pg-postgresql -o jsonpath='{.data.postgres-password}' | base64 -d)
kubectl -n demo-postgres exec pg-postgresql-0 -- bash -c "PGPASSWORD='$PGPW' psql -U postgres -d testdb -c '\\dt'"

# Verify nothing got left behind
kubectl -n demo-postgres get pvc,pods,volumesnapshot
# Expect: only data-pg-postgresql-0 + pg-postgresql-0; recent kopia VolumeSnapshots are EXPECTED (they're the backup artifacts!)
```
