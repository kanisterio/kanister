# Release Notes

## 0.118.0

### New Features

<!-- releasenotes/notes/pre-release-0.118.0-8816fe190614713e.yaml @ b'19086171e8029e588dce663abff08cd2a274e333' -->
* Added support for overriding default `initContainer` specifications in the `MultiContainerRun` function [https://github.com/kanisterio/kanister/pull/3824](https://github.com/kanisterio/kanister/pull/3824)

### Deprecations

<!-- releasenotes/notes/pre-release-0.118.0-8816fe190614713e.yaml @ b'19086171e8029e588dce663abff08cd2a274e333' -->
* Moved example Blueprints to a separate repository [https://github.com/kanisterio/blueprints](https://github.com/kanisterio/blueprints) [https://github.com/kanisterio/kanister/pull/3788](https://github.com/kanisterio/kanister/pull/3788)

## 0.117.0

### Bug Fixes

<!-- releasenotes/notes/pre-release-0.117.0-fef5a26613c8c0e2.yaml @ b'0637f3682f9d2618a8b8fa000271cfd46c750fdf' -->
* Updated MSSQL blueprint to use the correct `sqlcmd` binary path in the updated tools image [https://github.com/kanisterio/kanister/pull/3732](https://github.com/kanisterio/kanister/pull/3732)

<!-- releasenotes/notes/pre-release-0.117.0-fef5a26613c8c0e2.yaml @ b'0637f3682f9d2618a8b8fa000271cfd46c750fdf' -->
* Added a cleanup step in DataSuite unit tests to remove leftover bucket objects after test failures [https://github.com/kanisterio/kanister/pull/3752](https://github.com/kanisterio/kanister/pull/3752)

### Upgrade Notes

<!-- releasenotes/notes/pre-release-0.117.0-fef5a26613c8c0e2.yaml @ b'0637f3682f9d2618a8b8fa000271cfd46c750fdf' -->
* Upgraded the base images for all tooling containers to Go 1.25–based images [https://github.com/kanisterio/kanister/pull/3720](https://github.com/kanisterio/kanister/pull/3720)

## 0.116.0

### New Features

<!-- releasenotes/notes/pre-release-0.116.0-c98ca63f11dae458.yaml @ b'60c4551536404609713133e6f501ab43527141da' -->
* Switched to bitnamilegacy image repository for mysql, postgres, maria, mongo & cassandra example apps [https://github.com/kanisterio/kanister/pull/3616](https://github.com/kanisterio/kanister/pull/3616) [https://github.com/kanisterio/kanister/pull/3617](https://github.com/kanisterio/kanister/pull/3617)

### Bug Fixes

<!-- releasenotes/notes/pre-release-0.116.0-c98ca63f11dae458.yaml @ b'60c4551536404609713133e6f501ab43527141da' -->
* Fixed unit test TestContextTimeout to work on GKE clusters [https://github.com/kanisterio/kanister/pull/3632](https://github.com/kanisterio/kanister/pull/3632)

## 0.115.0

### New Features

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Add support for NetworkPolicy, Service, PVC, and Pod to pkg/ephemeral appliers [https://github.com/kanisterio/kanister/pull/3576](https://github.com/kanisterio/kanister/pull/3576)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Removed deprecated functions CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot in favour of CSI snapshot functions [https://github.com/kanisterio/kanister/pull/3581](https://github.com/kanisterio/kanister/pull/3581)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Add kanctl binary to kanister-tools Docker Image [https://github.com/kanisterio/kanister/pull/3578](https://github.com/kanisterio/kanister/pull/3578)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Support configmaps as phase objects [https://github.com/kanisterio/kanister/pull/3500](https://github.com/kanisterio/kanister/pull/3500)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Harden Job Pod Service Account RBAC Settings [https://github.com/kanisterio/kanister/pull/3542](https://github.com/kanisterio/kanister/pull/3542)

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Support container image override for BackupDataStats, CopyVolumeData, DeleteData and DeleteDataAll functions

<!-- releasenotes/notes/prepare_data_fail-2740d1b81db18a85.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Added new argument to PrepareData to enable command failure propagation [https://github.com/kanisterio/kanister/pull/3533](https://github.com/kanisterio/kanister/pull/3533)

### Bug Fixes

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Fixed use-case when CopyVolumeData followed by RestoreData [https://github.com/kanisterio/kanister/pull/3524](https://github.com/kanisterio/kanister/pull/3524)

### Upgrade Notes

<!-- releasenotes/notes/pre-release-0.115.0-5b3cbfef0ca0f77f.yaml @ b'7b8e4e4b654af6a38883f6b0fc351fc052f7ea3f' -->
* Volume snapshot functions CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot were deleted. Use CSI snapshot functions.

## 0.114.0

### New Features

<!-- releasenotes/notes/release-0fde4f9-adding-liveness-readiness-probe-kanister-operator.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Added liveness and readiness probe for Kanister operator.

<!-- releasenotes/notes/release-1c2fda5-adding-patch-operation-kubeops-function.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Support patch operation in the KubeOps function.

<!-- releasenotes/notes/release-f398e80-adding-security-context-pod-container-kanister-operator.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Security Context of the Kanister operator pod can be configured using the helm fields `podSecurityContext` and `containerSecurityContext`.

### Bug Fixes

<!-- releasenotes/notes/release-01e6c0f-restore-log-stream.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Restored log stream functionality to improve debugging and monitoring capabilities.

<!-- releasenotes/notes/release-1b7dce3-fix-copy-container-override-multicontainerrun.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Make container override copied to background and output overrides for MultiContainerRun function.

<!-- releasenotes/notes/release-618246c-adding-failure-reasons-actionset-cr.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Added failure reasons in ActionSet CR.

<!-- releasenotes/notes/release-77ffaf0-updated-s3-profile-validation-documentation.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Improved S3 profile validation error messages.

### Deprecations

<!-- releasenotes/notes/deprecate-volume-snapshot-9fdf5b18604bd734.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Volume snapshot function such as CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot in favour of CSI snapshot functions.

### Other Notes

<!-- releasenotes/notes/deprecate-boringcrypto-3bf65cde59c99ce6.yaml @ b'86f4a2a6deed2527c1535822d9acf6ad683848f8' -->
* Build process changed from using GODEBUG=boringcrypto to Go1.24 native crypto libraries for FIPS-compliant use.

## 0.113.0

### New Features

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* Added gRPC call to support sending of UNIX signals to `kando` managed processes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* Added command line option to follow stdout/stderr of `kando` managed processes

<!-- releasenotes/notes/rds-credentials-1fa9817a21a2d80a.yaml @ b'c4534cdbb7167c6f854c4d7915dd22483f9486f9' -->
* Enable RDS functions to accept AWS credentials using a Secret or ServiceAccount.

### Bug Fixes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* The Kopia snapshot command output parser now skips the ignored and fatal error counts

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* Set default namespace and serviceaccount for MultiContainerRun pods

### Upgrade Notes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* Upgrade to K8s 1.31 API

### Deprecations

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* K8s VolumeSnapshot is now GA, remove support for beta and alpha APIs

### Other Notes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* Change `TIMEOUT_WORKER_POD_READY` environment variable to `KANISTER_POD_READY_WAIT_TIMEOUT`

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'63c73f551aea7696a6dcaa77b628c24a9a53ea2b' -->
* Errors are now handled with [https://github.com/kanisterio/errkit](https://github.com/kanisterio/errkit) across the board

## 0.112.0

### New Features

<!-- releasenotes/notes/multi-container-run-function-d488516c0f3b22c6.yaml @ b'a72741deb67462a80a93856794d8a5c4425bb7c1' -->
* Introduced new Kanister function `MultiContainerRun` to run pods with two containers connected by shared volume.

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'a72741deb67462a80a93856794d8a5c4425bb7c1' -->
* Introduced a GRPC client/server to `kando` to run/check processes.

### Security Issues

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'a72741deb67462a80a93856794d8a5c4425bb7c1' -->
* Enhanced security by removing the default `edit` `ClusterRoleBinding` assignment, minimizing the risk of excessive permissions.

### Upgrade Notes

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'a72741deb67462a80a93856794d8a5c4425bb7c1' -->
* Users upgrading from previous versions should note that the `edit` `ClusterRoleBinding` is no longer included by default. They must now create their own `Role` / `RoleBinding` with appropriate permissions for Kanister's Service Account in the application's namespace.

### Other Notes

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'a72741deb67462a80a93856794d8a5c4425bb7c1' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1227.1726694542.

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'a72741deb67462a80a93856794d8a5c4425bb7c1' -->
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
