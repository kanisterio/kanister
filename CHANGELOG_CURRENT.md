# Release Notes

## 0.115.0

## New Features

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Add support for NetworkPolicy, Service, PVC, and Pod to pkg/ephemeral appliers [https://github.com/kanisterio/kanister/pull/3576](https://github.com/kanisterio/kanister/pull/3576)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Removed deprecated functions CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot in favour of CSI snapshot functions [https://github.com/kanisterio/kanister/pull/3581](https://github.com/kanisterio/kanister/pull/3581)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Add kanctl binary to kanister-tools Docker Image [https://github.com/kanisterio/kanister/pull/3578](https://github.com/kanisterio/kanister/pull/3578)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Support configmaps as phase objects [https://github.com/kanisterio/kanister/pull/3500](https://github.com/kanisterio/kanister/pull/3500)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Harden Job Pod Service Account RBAC Settings [https://github.com/kanisterio/kanister/pull/3542](https://github.com/kanisterio/kanister/pull/3542)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Support container image override for BackupDataStats, CopyVolumeData, DeleteData and DeleteDataAll functions

<!-- releasenotes/notes/prepare_data_fail-2740d1b81db18a85.yaml @ b'aa78d08bfb30c16136da1d94352fbf3bd0ee3de0' -->
* Added new argument to PrepareData to enable command failure propagation [https://github.com/kanisterio/kanister/pull/3533](https://github.com/kanisterio/kanister/pull/3533)

## Bug Fixes

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Fixed use-case when CopyVolumeData followed by RestoreData [https://github.com/kanisterio/kanister/pull/3524](https://github.com/kanisterio/kanister/pull/3524)

## Upgrade Notes

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'e69dfba5fc7a0ebab8ca7e6eecf4dd1384d770cf' -->
* Volume snapshot functions CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot were deleted. Use CSI snapshot functions.
