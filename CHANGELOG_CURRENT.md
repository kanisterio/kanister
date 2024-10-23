# Release Notes

## 0.112.0

## New Features

<!-- releasenotes/notes/multi-container-run-function-d488516c0f3b22c6.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Introduced new Kanister function `MultiContainerRun` to run pods with two containers connected by shared volume.

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Introduced a GRPC client/server to `kando` to run/check processes.

## Security Issues

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'a9edc6dc95d8772de502cb09b386f55e20baa33f' -->
* Enhanced security by removing the default `edit` `ClusterRoleBinding` assignment, minimizing the risk of excessive permissions.

## Upgrade Notes

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'a9edc6dc95d8772de502cb09b386f55e20baa33f' -->
* Users upgrading from previous versions should note that the `edit` `ClusterRoleBinding` is no longer included by default. They must now create their own `Role` / `RoleBinding` with appropriate permissions for Kanister's Service Account in the application's namespace.

## Other Notes

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1227.1726694542.

<!-- releasenotes/notes/pre-release-0.112.0-78fed87c3f58d801.yaml @ b'278e79a1ab5bc54fb1aa55d13474a49b6f319836' -->
* Add `gci` and `unparam` linters to test packages.
