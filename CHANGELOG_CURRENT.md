# Release Notes

## 0.110.0

## New Features

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Split parallelism helm value into dataStore.parallelism.upload and dataStore.parallelism.download to be used separately in BackupDataUsingKopiaServer and RestoreDataUsingKopiaServer

## Bug Fixes

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Make pod writer exec wait for cat command to finish. Fixes race condition between cat cat command end exec termination.

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Make sure all storage providers return similar error if snapshot doesn't exist, which is expected by DeleteVolumeSnapshot

## Other Notes

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1194

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Update errkit to v0.0.2

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Switch pkg/app to errkit

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Switch pkg/kopia to errkit

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'ee13e8df9850ff4a5ead22c922120b13c80614a5' -->
* Switch pkg/kube to errkit
