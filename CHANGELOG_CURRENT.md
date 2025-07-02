# Release Notes

## 0.114.0

## New Features

<!-- releasenotes/notes/nodename-podoptions-b76f6c68a6d646a0.yaml @ b'7047d008ef90baf4b69f31dbfe0b2ab6fbcc0cbd' -->
* Type `PodOptions` can now be used to configure the node name of the pod that is going to be created.

## Known Issues

<!-- releasenotes/notes/pre-release-0.114.0-cde047dfd4c5ad27.yaml @ b'f398e801e346d632a83feca6f49ec24f4552bfed' -->
* Security Context of the Kanister operator pod can be configured using the helm fields  and .

## Deprecations

<!-- releasenotes/notes/deprecate-volume-snapshot-9fdf5b18604bd734.yaml @ b'40006340a36663f73b8b89a221eaa2cd0187db08' -->
* Volume snapshot function such as CreateVolumeSnapshot, WaitForSnapshotCompletion, CreateVolumeFromSnapshot and DeleteVolumeSnapshot in favour of CSI snapshot functions

## Other Notes

<!-- releasenotes/notes/deprecate-boringcrypto-3bf65cde59c99ce6.yaml @ b'ccab82b184a1988c9a5b6d369eeb9ebd31af7b3f' -->
* Build process changed from using GODEBUG=boringcrypto to Go1.24 native crypto libraries for FIPS-compliant use
