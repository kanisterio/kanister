# Release Notes

## 0.120.0

## New Features

<!-- releasenotes/notes/azure-federated-identity-sas-05a2214eadf859c8.yaml @ b'8958767636c088cec9251d03a38583a38e400e44' -->
* Added support for Azure Workload (Federated) Identity when moving backup data to Azure Blob Storage. The controller mints a short-lived, container-scoped, least-privilege user-delegation SAS token and injects it into the data-mover pods, so no static storage account key is needed. The controller's managed identity must be able to fetch a user-delegation key on the storage account (e.g. the `Storage Blob Data Contributor` role).

<!-- releasenotes/notes/waitv2-render-timeout-bf427954363813e5.yaml @ b'58c883080158ae7d3ea49400c69f5fbec00c503d' -->
* Added template support for the `timeout` argument of the `WaitV2` function [https://github.com/kanisterio/kanister/pull/4185](https://github.com/kanisterio/kanister/pull/4185)

## Upgrade Notes

<!-- releasenotes/notes/pre-release-0.120.0-1b868fe733138f87.yaml @ b'5371a896abb4aa4dbe1e1829c3e9d1bbe3f37777' -->
* Upgraded Go version to 1.27.3 [https://github.com/kanisterio/kanister/pull/4281](https://github.com/kanisterio/kanister/pull/4281)

## Deprecations

<!-- releasenotes/notes/pre-release-0.120.0-1b868fe733138f87.yaml @ b'5371a896abb4aa4dbe1e1829c3e9d1bbe3f37777' -->
* Remove build support for kafka-adobe-s3 connector image [https://github.com/kanisterio/kanister/pull/4115](https://github.com/kanisterio/kanister/pull/4115)

<!-- releasenotes/notes/remove-repository-server-crd-3ff3016890.yaml @ b'5371a896abb4aa4dbe1e1829c3e9d1bbe3f37777' -->
* Clean up stale code for `repositoryserver`.
