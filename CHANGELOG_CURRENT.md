# Release Notes

## 0.112.0

## New Features

<!-- releasenotes/notes/multi-container-run-function-d488516c0f3b22c6.yaml @ b'7723a4ac5ad0efc7e41c61053ae1f29a68400f66' -->
* Introduced new Kanister function `MultiContainerRun` to run pods with two containers connected by shared volume

## Security Issues

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'1f40f03d8432e8dc80fe248d306c1e201808ec59' -->
* Enhanced security by removing the default   assignment, minimizing the risk of excessive permissions.

## Upgrade Notes

<!-- releasenotes/notes/limit-rbac-kanister-operator-3c933af021b8d48a.yaml @ b'1f40f03d8432e8dc80fe248d306c1e201808ec59' -->
* Users upgrading from previous versions should note that the   is no longer included by default. They must now create their own  /  with appropriate permissions for Kanister's Service Account in the application's namespace.
