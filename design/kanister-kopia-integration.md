# Integrating Kopia with Kanister

This document proposes all the changes required within Kanister to use [Kopia](https://kopia.io/) as the primary backup and restore tool.

## Problems

Kanister offers an in-house capability to perform backup and restore using some operation-specific Functions like BackupData, RestoreData, etc.
Although, they are much easier to use and significantly improve the user experience of Blueprint authors, these Functions have a lot of issues.

Few of those problems are;
1. They do not encrypt in-flight data during transfer
2. The backup data at-rest on the cloud stores are not secure
3. Operations are slow compared to other tools in the market

Therefore, they are not ideally recommended to be used for protecting application data.

Instead, we make use of v2 Blueprints whose phases call a `kando` utility to perform data snapshots.
This tool internally executes `Kopia` commands to manage backups in the cloud.

This approach is again not a robust solution because;
1. We have to create a separate kopia profile and use it within kando command to pass necessary kopia server info
2. We have to follow the correct kando command syntax in the Blueprint while backing up and mention `--output-name` in the Action outputArtifacts
3. Kopia errors are not logged when kando commands fail in the acion phase  

## Introducing Kopia

Kopia is a powerful, cross-platform tool for managing encrypted backups in the cloud.
It provides fast and secure backups, using compression, data deduplication and client-side end-to-end encryption.
It supports a variety of backup storage targets, including object stores, which allows users to choose the storage provider that better addresses their needs.
It is a lock-free system that allows for concurrent multi-client operations including garbage collection.

To explore other features of Kopia, see its [documentation](https://kopia.io/docs/features/).

## Goals

To solve existing problems;
1. We should replace the entire existing functionality in operation-specific Kanister Functions like BackupData, RestoreData, CopyVolumeData, etc. with  Kopia workflows/
2. We should leverage the use of Kopia Repository Server within these Functions since server operations allow access to only resgistered users and effectively hide the cloud storage credentials. 

##  Scope

### Kanister Functions 

We already have a rich repository of Kanister functions present on path `pkg/function` that enable application-level data protection in various use cases.
Few of these functions need to be re-worked to use Kopia. For starters, these functions are;

1. BackupData
2. BackupDataAll
3. BackupDataStats
4. CopyVolumeData
5. DeleteData
6. DeleteDataAll
7. RestoreData
8. RestoreDataAll

Please note that the functions stated above will only be **refactored** to use Kopia underneath.
There should not be any changes with respect to the objective of these functions.
The motive, arguments and usage for each function stays intact as defined on Kanister docs https://docs.kanister.io/functions.html#existing-functions

### Kanister Functions that use Kopia API Server 

We should also create Kanister functions that leverage the use of [Kopia in it's server mode](https://kopia.io/docs/repository-server/).
These Kanister functions would act like Kopia clients that securely proxy access repository storage without exposing sensitive storage credentials.

Following is a list of few such functions;

1. BackupDataToServer - To perform backup via the KopiaAPIserver
2. RestoreDataFromServer - To restore from the KopiaAPIServer

### Limitations to using Kopia API Server

Before starting Kopia API Server, it requires the creation of users that are allowed access.
We can make use of the application namespace ID to generate usernames and a random alphanumeric encryption key to create passwords.
These would be stored in a secret for later use. But;
- How should we store the server username and server password securely?
- At which point in Kanister, should Kopia API Server be started?
- How could this Kopia API Server be disabled in case downstream Kanister consumers bring their own Kopia API Server?
