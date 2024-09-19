# Release Notes

## 0.111.0

### New Features

<!-- releasenotes/notes/actionset-podlabels-annotations-915f1dfa7ee86978.yaml @ b'e3b8b72ec338faba1b915ae02862cba106fe1551' -->
* Added two new fields, `podLabels` and `podAnnotations`, to the ActionSet. These fields can be used to configure the labels and annotations of the Kanister function pod run by an ActionSet.

<!-- releasenotes/notes/label-annotations-to-functions-903e5ffdff79a415.yaml @ b'c3c3bc982ba3a4521d3146dbc46b278917f31c64' -->
* Added support to customise the labels and annotations of the temporary pods that are created by some Kanister functions.

<!-- releasenotes/notes/postgress-tools-image-override-4882c70780e8a496.yaml @ b'd4be0962a8521e4674de581590fd4b026ca5dce8' -->
* Support `image` argument for `ExportRDSSnapshotToLocation` and `RestoreRDSSnapshot` functions to override default postgres-kanister-tools image.

<!-- releasenotes/notes/support-annotation-on-snapshotter-function-ff9b7ba2daf10427.yaml @ b'ea6cb88542d601776f5f5dc0736d532af7ba0c3a' -->
* Add support to pass labels and annotations to the methods that create/clone VolumeSnapshot and VolumeSnapshotContent resources.

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
