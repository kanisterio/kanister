# Integrating Kopia with Kanister

This document proposes all the changes required within Kanister to encourage the use of Kopia backup/restore tool.

## Introducing Kopia

Kopia is a simple, cross-platform tool for managing encrypted backups in the cloud.
It provides fast, incremental backups, secure, client-side end-to-end encryption, compression and data deduplication.
It is one of the fastest backup/restore tool in the field and with a pluggable backend engine, it allows user to decide which storage the backups should go to.
It is a lock-free system that, unlike other similar tools, allows for concurrent multi-client operations including garbage collection.

To learn more about Kopia and it's features, visit kopia.io

## Goal

Kopia can focus on its core data transformation as part of various Kanister functions.
While, Kanister takes care of executing these functions from it's application-centric data protection workflows called Blueprints.

##  Proposed Work

### Kanister Functions 

We already have a rich repository of Kanister functions present on path `pkg/function` that enable application-level data protection in various use-cases.
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

We should also create Kanister functions same as above, but leverage the use of [Kopia in it's server mode](https://kopia.io/docs/repository-server/).
These Kanister functions would act like Kopia clients trying to securely proxy access to the repository storage without exposing sensitive storage credentials.

Following are the names of a few that could be implemented;

1. BackupDataToServer - To perform backup via the KopiaAPIserver
2. RestoreDataFromServer - To restore from the KopiaAPIServer

### Limitations to using Kopia API Server

Before starting Kopia API Server, it demands a list of usernames and passwords that are allowed access.
We can make use of the application namespace ID to generate usernames and a random alphanumeric encryption key to create passwords.
These would be stored in a secret for later use. But;
- How should we store the server username and server password securely?
- At which point in Kanister, should Kopia API Server be started?
- How could this Kopia API Server be disabled in case downstream Kanister users bring their own Kopia API Server?
