# Release Notes

## 0.119.0

## New Features

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Added `DeepCopy` method for `PodOptions` to support safe copying of pod configuration [https://github.com/kanisterio/kanister/pull/3820](https://github.com/kanisterio/kanister/pull/3820)

## Bug Fixes

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Fixed AWS STS role assumption to correctly thread the location region through the credential chain [https://github.com/kanisterio/kanister/pull/4050](https://github.com/kanisterio/kanister/pull/4050)

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Fixed Portworx CSI transient errors to be retried instead of failing the operation immediately [https://github.com/kanisterio/kanister/pull/3962](https://github.com/kanisterio/kanister/pull/3962)

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Removed unnecessary `EndpointSlice` polling during service readiness checks [https://github.com/kanisterio/kanister/pull/3897](https://github.com/kanisterio/kanister/pull/3897)

## Upgrade Notes

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Upgraded Go version to 1.26.1 [https://github.com/kanisterio/kanister/pull/3941](https://github.com/kanisterio/kanister/pull/3941)

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Upgraded `postgres-kanister-tools` base image to `postgres:18-bookworm` [https://github.com/kanisterio/kanister/pull/4046](https://github.com/kanisterio/kanister/pull/4046)

<!-- releasenotes/notes/pre-release-0.119.0-c20a35e84b1f1912.yaml @ b'1794e6a50c63e3b33233dd94ccaac75b6c7f6f34' -->
* Migrated AWS SDK from v1 to v2 in `pkg/aws` [https://github.com/kanisterio/kanister/pull/3945](https://github.com/kanisterio/kanister/pull/3945)
