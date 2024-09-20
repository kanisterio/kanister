# Release Notes

## 0.111.0

### New Features

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Add support for Read-Only and Write Access Modes when connecting to the Kopia Repository Server in `kando`.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Add support for Cache Size Limits to the `kopia server start` command.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Add support to pass labels and annotations to the methods that create/clone VolumeSnapshot and VolumeSnapshotContent resources.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Support `image` argument for `ExportRDSSnapshotToLocation` and `RestoreRDSSnapshot` functions to override default postgres-kanister-tools image.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Added support to customise the labels and annotations of the temporary pods that are created by some Kanister functions.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Added two new fields, `podLabels` and `podAnnotations`, to the ActionSet. These fields can be used to configure the labels and annotations of the Kanister function pod run by an ActionSet.

### Security Issues

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Update Go to 1.22.7 to pull in latest security updates.

### Other Notes

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1227.1725849298.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
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
