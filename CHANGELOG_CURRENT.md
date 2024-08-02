# Release Notes

## 0.110.0

## New Features

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Split  helm value into  and  to be used separately in  and 

## Bug Fixes

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Make pod writer exec wait for cat command to finish. Fixes race condition between cat cat command end exec termination.

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Make sure all storage providers return similar error if snapshot doesn't exist, which is expected by 

## Other Notes

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Update ubi-minimal base image to ubi-minimal:9.4-1194

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Update errkit to v0.0.2

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Switch pkg/app to errkit

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Switch pkg/kopia to errkit

<!-- releasenotes/notes/pre-release-0.110.0-a47623540224894a.yaml @ b'825b6e52c8e7c3f161ab5f7048e3bc5958ca3f7e' -->
* Switch pkg/kube to errkit
