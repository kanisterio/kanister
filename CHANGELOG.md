# Release Notes

## 0.112.0

### New Features

<!-- releasenotes/notes/multi-container-run-function-d488516c0f3b22c6.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Introduced new Kanister function `MultiContainerRun` to run pods with two containers connected by shared volume.

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Introduced a GRPC client/server to `kando` to run/check processes.

### Security Issues

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'a9edc6dc95d8772de502cb09b386f55e20baa33f' -->
* Enhanced security by removing the default `edit` `ClusterRoleBinding` assignment, minimizing the risk of excessive permissions.

### Upgrade Notes

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'a9edc6dc95d8772de502cb09b386f55e20baa33f' -->
* Users upgrading from previous versions should note that the `edit` `ClusterRoleBinding` is no longer included by default. They must now create their own `Role` / `RoleBinding` with appropriate permissions for Kanister's Service Account in the application's namespace.

### Other Notes

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1227.1726694542.

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Add `gci` and `unparam` linters to test packages.

## 0.111.0

### New Features

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Add support for Read-Only and Write Access Modes when connecting to the Kopia Repository Server in `kando`.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Add support for Cache Size Limits to the `kopia server start` command.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Add support to pass labels and annotations to the methods that create/clone VolumeSnapshot and VolumeSnapshotContent resources.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Support `image` argument for `ExportRDSSnapshotToLocation` and `RestoreRDSSnapshot` functions to override default postgres-kanister-tools image.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Added support to customise the labels and annotations of the temporary pods that are created by some Kanister functions.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Added two new fields, `podLabels` and `podAnnotations`, to the ActionSet. These fields can be used to configure the labels and annotations of the Kanister function pod run by an ActionSet.

### Security Issues

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Update Go to 1.22.7 to pull in latest security updates.

### Other Notes

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1227.1725849298.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'd207c416a800fdff15f570275f1e3dfe0ede4ffe' -->
* Add `stylecheck`, `errcheck`, and `misspel` linters to test packages.

## 0.110.0

### New Features

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Split parallelism helm value into dataStore.parallelism.upload and dataStore.parallelism.download to be used separately in BackupDataUsingKopiaServer and RestoreDataUsingKopiaServer

### Bug Fixes

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Make pod writer exec wait for cat command to finish. Fixes race condition between cat cat command end exec termination.

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Make sure all storage providers return similar error if snapshot doesn't exist, which is expected by DeleteVolumeSnapshot

### Other Notes

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1194

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Update errkit to v0.0.2

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Switch pkg/app to errkit

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Switch pkg/kopia to errkit

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'fffef729e348ce0cf8bba3646303460d5e37fe16' -->
* Switch pkg/kube to errkit
