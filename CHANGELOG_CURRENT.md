# Release Notes

## 0.114.0

## New Features

<!-- releasenotes/notes/release-0fde4f9-adding-liveness-readiness-probe-kanister-operator.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Added liveness and readiness probe for Kanister operator.

<!-- releasenotes/notes/release-1c2fda5-adding-patch-operation-kubeops-function.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Support patch operation in the KubeOps function.

<!-- releasenotes/notes/release-f398e80-adding-security-context-pod-container-kanister-operator.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Security Context of the Kanister operator pod can be configured using the helm fields `podSecurityContext` and `containerSecurityContext`.

## Bug Fixes

<!-- releasenotes/notes/release-01e6c0f-restore-log-stream.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Restored log stream functionality to improve debugging and monitoring capabilities.

<!-- releasenotes/notes/release-1b7dce3-fix-copy-container-override-multicontainerrun.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Make container override copied to background and output overrides for MultiContainerRun function.

<!-- releasenotes/notes/release-618246c-adding-failure-reasons-actionset-cr.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Added failure reasons in ActionSet CR.

<!-- releasenotes/notes/release-77ffaf0-updated-s3-profile-validation-documentation.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Improved S3 profile validation error messages.

## Deprecations

<!-- releasenotes/notes/deprecate-volume-snapshot-9fdf5b18604bd734.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Volume snapshot function such as CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot in favour of CSI snapshot functions.

## Other Notes

<!-- releasenotes/notes/deprecate-boringcrypto-3bf65cde59c99ce6.yaml @ b'cb7c6704e8a26b988e8f5eaa6681948989ab989d' -->
* Build process changed from using GODEBUG=boringcrypto to Go1.24 native crypto libraries for FIPS-compliant use.
