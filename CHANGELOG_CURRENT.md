# Release Notes

## 0.111.0

## New Features

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

## Security Issues

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Update Go to 1.22.7 to pull in latest security updates.

## Other Notes

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1227.1725849298.

<!-- releasenotes/notes/pre-release-0.111.0-478149ddf5d56f80.yaml @ b'07949285eea9a1c7f0768bd8c8354d64278b0d82' -->
* Add `stylecheck`, `errcheck`, and `misspel` linters to test packages.
