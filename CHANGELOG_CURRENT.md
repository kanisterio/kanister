# Release Notes

## 0.113.0

## New Features

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* Added gRPC call to support sending of UNIX signals to `kando` managed processes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* Added command line option to follow stdout/stderr of `kando` managed processes

<!-- releasenotes/notes/rds-credentials-1fa9817a21a2d80a.yaml @ b'c4534cdbb7167c6f854c4d7915dd22483f9486f9' -->
* Enable RDS functions to accept AWS credentials using a Secret or ServiceAccount.

## Bug Fixes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* The Kopia snapshot command output parser now skips the ignored and fatal error counts

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* Set default namespace and serviceaccount for MultiContainerRun pods

## Upgrade Notes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* Upgrade to K8s 1.31 API

## Deprecations

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* K8s VolumeSnapshot is now GA, remove support for beta and alpha APIs

## Other Notes

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* Change `TIMEOUT_WORKER_POD_READY` environment variable to `KANISTER_POD_READY_WAIT_TIMEOUT`

<!-- releasenotes/notes/pre-release-0.113.0-591b9333c935aae6.yaml @ b'213b025c275a5eba8600b9f48942a851f85e8853' -->
* Errors are now handled with [https://github.com/kanisterio/errkit](https://github.com/kanisterio/errkit) across the board
