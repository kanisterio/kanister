# Integrating Kopia with Kanister

This document proposes all the high-level changes required within Kanister to use [Kopia](https://kopia.io/) as the primary backup and restore tool.

## Motivation

Kanister offers an in-house capability to perform backup and restore to and from object stores using some operation-specific Functions like BackupData, RestoreData, etc.
Although they are useful and simple to use, these Functions can be significantly improved to provide better reliability, security, and performance.

The improvements would include:
1. Encryption of data during transfers and at rest
2. Efficient content-based data deduplication
3. Configurable data compression
4. Reduced memory consumption
5. Increased variety of backend storage target for backups

These improvements can be achieved by using `Kopia` as the primary data movement tool in these Kanister Functions.

Kanister also provides a command line utility `kando` that can be used to move data to and from object stores.
This tool internally executes `Kopia` commands to move the backup data.
The v2 version of the example Kanister Blueprints supports this. However, there are a few caveats to using these Blueprints.
1. `kando` uses `Kopia` only when a Kanister Profile of type `Kopia` is provided
2. A Kanister Profile of type `Kopia` requires a [Kopia Repository Server](https://kopia.io/docs/repository-server/) running in the same namespace as the Kanister controller
3. A Repository Server requires a [Kopia Repository](https://kopia.io/docs/repositories/) to be initialized on a backend storage target
   
Kanister currently lacks documentation and automation to use these features.

## Introducing Kopia

Kopia is a powerful, cross-platform tool for managing encrypted backups in the cloud.
It provides fast and secure backups, using compression, data deduplication and client-side end-to-end encryption.
It supports a variety of backup storage targets, including object stores, which allows users to choose the storage provider that better addresses their needs.
It is a lock-free system that allows for concurrent multi-client operations including garbage collection.

To explore other features of Kopia, see its [documentation](https://kopia.io/docs/features/).

## Scope

1. Design the usage of [Kopia Repository Server](https://kopia.io/docs/repository-server/) as an separate on-demand workload
   - In order to make use of user access control and cloud storage credential abstraction offered in the server-based operations 
2. Re-work Kanister Functions like BackupData, RestoreData, CopyVolumeData, etc. with  Kopia workflows
   - And leverage the use of Kopia Repository Server workload within these Functions

### Design the usage of Kopia Repository Server



### Re-work Kanister Functions 

We already have a rich repository of Kanister Functions present on path `pkg/function` that enable application-level data protection in various use cases.
Some of these functions need to be re-worked to follow the Kopia Repository Server workflow. For starters, these functions are;

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

## Backward Compatibility

### Kanister Functions

- Kanister offers the facility to register Functions on multiple versions
- We intend to make use of this functionality to maintain two separate version groups of Kanister Functions
- `v0.0.0` is the current default version where all the existing Kanister Functions are registered
- Proposed version `v1.0.0-alpha` would consist of all the new re-worked Kanister Functions
- Users can either continue with `v0.0.0` to use the existing Kanister Functions or they can switch to `v1.0.0-alpha` for Kopia-based Kanister Functions

Proper steps to toggle between the versions of Kanister Functions will be documented here later. By default, the version would be `v0.0.0`.

### Kopia Repository Server

- We plan to introduce a feature flag during Kanister installation to enable or disable Kopia Repository Server workload
- Kanister Functions in `v1.0.0-alpha` would only work when this feature flag is enabled 
- To use Kanister Functions in `v0.0.0`, users may or may not disable Kopia Repository Server workload

Proper steps to toggle between the Kopia Server workload support will be documented here later. By default, the feature flag would be disabled.

Q: Instead of a new feature flag could we use the Kanister Function Version itself to enable or disable Kopia Repository Server workload/CR?

## User Experience

- Existing users and downstream consumers can continue to make use of previous functionality as per the above 'Backward Compatibility' section
- Which means upgrading the Kanister controller version wouldn't need any changes in existing blueprint workflows
- However, the user experience for new Kanister Functions is expected to change based on the design decisions for Kopia Repository Server workload usage
- Mainly, the user might have to perform CRUD on the Repository Server workload which would be an added prerequisite to work with Kopia-based Kanister Functions
- In case, the users switch off the feature flag for server workload and wish to create a server by themselves, 
  we aim to provide them with detailed steps to create a server workload as part of "bring-your-own-server" BYOS model

## Limitations to using Kopia API Server

Before starting Kopia API Server, it requires the creation of users that are allowed access.
We can make use of the application namespace ID to generate usernames and a random alphanumeric encryption key to create passwords.
These would be stored in a secret for later use. But;
- How should we store the server username and server password securely?
- At which point in Kanister, should Kopia API Server be started?
- How could this Kopia API Server be disabled in case downstream Kanister consumers bring their own Kopia API Server?
