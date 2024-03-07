# Integrating Kopia with Kanister

<!-- toc -->
- [Motivation](#motivation)
- [Introducing Kopia](#introducing-kopia)
- [Scope](#scope)
- [User Experience](#user-experience)
- [Detailed Design](#detailed-design)
  - [Kanister Data Functions](#kanister-data-functions)
  - [Kopia Repository](#kopia-repository)
  - [Kopia Repository Server](#kopia-repository-server)
    - [Custom Resource Definition](#custom-resource-definition)
    - [Repository Server Lifecycle](#repository-server-lifecycle)
    - [Server-side Setup](#server-side-setup)
    - [Client-side Setup](#client-side-setup)
    - [Server Access Users Management](#server-access-users-management)
  - [Secrets Management](#secrets-management)
  - [Replace Kopia CLI with SDK](#replace-kopia-CLI-with-SDK)
<!-- /toc -->

This document proposes all the high-level changes required within Kanister to
use [Kopia](https://kopia.io/) as the primary backup and restore tool.

## Motivation

Kanister offers an in-house capability to perform backup and restore to and
from object stores using some operation-specific Functions like BackupData,
RestoreData, etc.
Although they are useful and simple to use, these Functions can be
significantly improved to provide better reliability, security, and
performance.

The improvements would include:
1. Encryption of data during transfers and at rest
2. Efficient content-based data deduplication
3. Configurable data compression
4. Reduced memory consumption
5. Increased variety of backend storage target for backups

These improvements can be achieved by using Kopia as the primary data movement
tool in these Kanister Functions.

Kanister also provides a command line utility `kando` that can be used to move
data to and from object stores.
This tool internally executes Kopia commands to move the backup data.
The v2 version of the example Kanister Blueprints supports this. However, there
are a few caveats to using these Blueprints.
1. `kando` uses Kopia only when a Kanister Profile of type `kopia` is provided
2. A Kanister Profile of type `kopia` requires a
   [Kopia Repository Server](https://kopia.io/docs/repository-server/) running
   in the same namespace as the Kanister controller
3. A Repository Server requires a
   [Kopia Repository](https://kopia.io/docs/repositories/) to be initialized on
   a backend storage target

Kanister currently lacks documentation and automation to use these features.

## Introducing Kopia

Kopia is a powerful, cross-platform tool for managing encrypted backups in the
cloud.
It provides fast and secure backups, using compression, data deduplication, and
client-side end-to-end encryption.
It supports a variety of backup storage targets, including object stores, which
allows users to choose the storage provider that better addresses their needs.
In Kopia, these storage locations are called repositories.
It is a lock-free system that allows concurrent multi-client operations
including garbage collection.

To explore other features of Kopia, see its
[documentation](https://kopia.io/docs/features/).

## Scope

1. Design and automate the lifecycle of the required Kopia Repository Server.
2. Add new versions of Kanister Data Functions like BackupData,
   RestoreData, etc. with Kopia as the primary data mover tool.

## User Experience

- All the new features mentioned in this document will be opt-in only. Existing
   users will not see any changes in the Kanister controller's behavior.
- Users will be able to continue using their current Blueprints, switch to the
   v2 version of the example Blueprints, or use Blueprints with the new version
   of the Kanister Data Functions.
- Users opting to use the v2 Blueprints and Blueprints with Kopia-based
   Kanister Data Functions will be required to follow instructions to set up
   the required Kopia repository and the repository server before executing the
   actions.
- After setting up the repository and the repository server, users can follow
   the normal workflow to execute actions from the v2 Blueprints. To use the
   new versions of the Kanister Data Functions, users must specify the version
   of the function via the ActionSet Action's `preferredVersion` field. This
   field in an Action is applied to all the Phases in it. If a particular
   Kanister function is not registered with this version, Kanister will fall
   back to the default version of the function.

## Detailed Design

### Kanister Data Functions

- Kanister allows mutliple versions of Functions to be registered with the
   controller.
- Existing Functions are registered with the default `v0.0.0` version. Find
   more information
   [here](https://docs.kanister.io/functions.html#existing-functions).
- The following Data Functions will be registered with a second
   version `v1.0.0-alpha`:

   1. BackupData
   2. BackupDataAll
   3. BackupDataStats
   4. CopyVolumeData
   5. DeleteData
   6. DeleteDataAll
   7. RestoreData
   8. RestoreDataAll

- The purpose, signature and output of these functions will remain intact i.e.
   their usage in Blueprints will remain unchanged. However, their internal
   implementation will leverage Kopia to connect to the Repository Server to
   perform the required data operations.
- As noted above, users will execute these functions by specifying
   `v1.0.0-alpha` as the `preferredVersion` during the creation of
   an ActionSet.
- The version management scheme for these functions is out of scope of this
   document and will be discussed separately.

**Please note an important update here**
- The design for implementing these Kanister Functions
is still a work in-progress.
- The above-mentioned versioning of Kanister Functions may not be the
final design.
- We plan to submit a new Pull Request stating a more detailed design of these Kanister Functions, shortly.

### Kopia Repository

- As mentioned above, the backup storage location is called a "Repository"
   in Kopia.
- It is important to note that the Kanister users are responsible for the
   lifecycle management of the repository, i.e., the creation, deletion,
   upgrades, garbage collection, etc. The Kanister controller will not override
   any user configuration or policies set on the repository. Such direct
   changes might impact Kanister's ability to interact with the repository.
- Kanister documentation will provide instructions for initializing a new
   repository.
- Kanister users can initialize repositories with boundaries defined based on
   their needs. The repository domain can include a single workload, groups of
   workloads, namespaces, or groups of namespaces, etc.
- Only a single repository can exist at a particular path in the backend
   storage location. Users opting to use separate repositories are recommended
   to use unique path prefixes for each repository. For example, a repository
   for a namespace called `monitoring` on S3 storage bucket called
   `test-bucket` could be created at the location
   `s3://test-bucket/<UUID of monitoring namespace>/repo/`.
- Accessing the repository requires the storage location and credential
   information similar to a Kanister Profile CR and a unique password used by
   Kopia during encryption, along with a unique path prefix mentioned above.
   [See](https://kopia.io/docs/features/#end-to-end-zero-knowledge-encryption).
- In the first iteration, users will be required to provide the location and
   repository information to the controller during the creation of the
   repository server in the form of Kubernetes Secrets. Future iterations will
   allow users to use a Key Management Service of choice.

### Kopia Repository Server

- A Kopia Repository Server allows Kopia clients proxy access to the backend
   storage location through it.
- At any time, a repository server can only connect to a single repository. Due
   to this a separate instance of the server will be used for each repository.
- In Kanister, the server will comprise a K8s `Pod`, `Service` and
   a `NetworkPolicy`.
- The pod will execute the Kopia server process exposed to the application via
   the K8s service and the network policy.
- Accessing the server requires the service name, a server username, and a
   password without any knowledge of the backend storage location.
- To authorize access, a list of server usernames and passwords must be added
   prior to starting the server.
- The server also uses TLS certificates to secure incoming connections to it.
- Kanister users can configure a repository server via a newly added Custom
   Resource called `RepositoryServer` as described in the following section.

#### Custom Resource Definition

This design proposes a new Custom Resource Definition (CRD) named
`RepositoryServer`, to represent a Kopia repository server.
It is a namespace-scoped resource, owned and managed by the controller in the
`kanister` namespace. As mentioned above, the CRD offers a set of parameters to
configure a Kopia repository server.

To limit the controller's RBAC scope, all the `RepositoryServer` resources
are created in the `kanister` namespace.

A sample `RepositoryServer` resource created to interact with the Kopia
repository of workloads running in the `monitoring` namespace looks like this:

```yaml
apiVersion: cr.kanister.io/v1alpha1
kind: RepositoryServer
metadata:
   name: repository-monitoring
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
spec:
   storage:
      # required: storage location info
      secretRef:
         name: location
         namespace: kanister
      # required: creds to access the location
      # optional when using location type file-store
      credentialSecretRef:
         name: loc-creds
         namespace: kanister
   repository:
      # repository must be created manually before creating server CR
      # required: path for the repository - will be relative sub path
      # within the path prefix specified in the location
      rootPath: /repo/monitoring/
      # required: password to access the repository
      passwordSecretRef:
         name: repository-monitoring-password
         namespace: kanister
      # optional: if specified, these values will be used by the controller to
      # override default username and hostname when connecting to the
      # repository from the server.
      # if not specified, the controller will use generated defaults
      username: kanisterAdmin
      hostname: monitoring
   server:
      # required: server admin details
      adminSecretRef:
         name: repository-monitoring-server-admin
         namespace: kanister
      # required: TLS certificate required for secure communication between the Kopia client and server
      tlsSecretRef:
         name: repository-monitoring-server-tls
         namespace: kanister
      # required: repository users list
      userAccessSecretRef:
         name: repository-monitoring-server-access
         namespace: kanister
      # required: selector for network policy required to enable cross-namespace access to the server
      networkPolicy:
         namespaceSelector:
            matchLabels:
               app: monitoring
         podSelector:
            matchLabels:
               pod: kopia-client
               app: monitoring
status:
   conditions:
   - lastTransitionTime: "2022-08-20T09:48:36Z"
     lastUpdateTime: "2022-08-20T09:48:36Z"
     status: "True"
     type: RepositoryServerReady
   serverInfo:
      podName: "repository-monitoring-pod-a1b2c3"
      networkPolicyName: "repository-monitoring-np-d4e5f6"
      serviceName: "repository-monitoring-svc-g7h8i9"
      tlsFingerprint: "48537CCE585FED39FB26C639EB8EF38143592BA4B4E7677A84A31916398D40F7"
```

The required `spec.storage.secretRef` refers to a `Secret` resource storing
location-related sensitive data. This secret is provided by the user.
For example,

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: location
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   # required: specify the type of the store
   # supported values are s3, gcs, azure, and file-store
   type: s3
   # required
   bucket: my-bucket
   # optional: specified in case of S3-compatible stores
   endpoint: https://foo.example.com
   # optional: used as a sub path in the bucket for all backups
   path: kanister/backups
   # required, if supported by the provider
   region: us-west-1
   # optional: if set to true, do not verify SSL cert.
   # Default, when omitted, is false
   skipSSLVerify: false
   # required: if type is `file-store`
   # optional, otherwise
   claimName: store-pvc
```

The credentials required to access the location above are provided by the user
in a separate secret referenced by `spec.location.credentialSecretRef`.
The example below shows the credentials required to access AWS S3 and
S3-compatible locations.

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: s3-loc-creds
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: secrets.kanister.io/aws
data:
   # required: base64 encoded value for key with proper permissions for the bucket
   access-key: <redacted>
   # required: base64 encoded value for the secret corresponding to the key above
   secret-acccess-key: <redacted>
   # optional: base64 encoded value for AWS IAM role
   role: <redacted>
```

The credentials secret will follow a different format for different providers.
This secret is optional when using a file store location.
Example secrets for Google Cloud Storage (GCS) and Azure Blob Storage will be
as follows:

GCS:

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: gcs-loc-creds
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   # required: base64 encoded value for project with proper permissions for the bucket
   project-id: <redacted>
   # required: base64 encoded value for the SA with proper permissions for the bucket.
   # This value is base64 encoding of the service account json file when
   # creating a new service account
   service-account.json: <base64 encoded SA json file>
```

Azure:

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: az-loc-creds
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   # required: base64 encoded value for account with proper permissions for the bucket
   azure_storage_account_id: <redacted>
   # required: base64 encoded value for the key corresponding to the account above
   azure_storage_key: <redacted>
   # optional: base64 encoded value for the storage enevironment.
   # Acceptable values are AzureCloud, AzureChinaCloud, AzureUSGovernment, AzureGermanCloud
   azure_storage_environment: <redacted>
```

Kopia identifies users by `username@hostname` and uses the values specified
when establishing connection to the repository to identify backups created in
the session. If the username and hostname values are specified in the
repository section, Kanister controller will override defaults when
establishing connection to the repository from the repository server. Users
will be required to specify the same values when they need to restore or delete
the backups created.
By default, the controller will use generated defaults when connecting to the
repository.

The password used while creating the Kopia repository is provided by the user
in the `Secret` resource referenced by `spec.repository.passwordSecretRef`.

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: repository-monitoring-password
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   repo-password: <redacted>
```

The server admin credentials, and TLS sensitive data are stored in the `Secret`
resources referenced by the `spec.server.adminSecretRef` and
`spec.server.tlsSecretRef` properties. For example,

```yaml
apiVersion: v1
kind: Secret
metadata:
   name: repository-monitoring-server-admin
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   username: <redacted>
   password: <redacted>
---
apiVersion: v1
kind: Secret
metadata:
   name: repository-monitoring-server-tls
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: kubernetes.io/tls
data:
   tls.crt: |
      <redacted>
   tls.key: |
      <redacted>
```

The `spec.server.accessSecretRef` property provides a list of access
credentials used by data mover clients to authenticate with the Kopia
repository server. This is discussed in more detail in the
[Server Access Users Management](#server-access-users-management) section.

The Kopia repository server `Pod` resource can be accessed through a K8s
`Service`. The `spec.server.networkPolicy` property is used to determine the
selector used in the `namespaceSelector` or `podSelector` of the
`NetworkPolicy` resource that controls the ingress traffic to the repository
server from a namespace other than the `kanister` namespace.

The `status` subresource provides conditions and status information to ensure
that the controller does not attempt to re-create the repository server during
restart.

#### Repository Server Lifecycle

When a `RepositoryServer` resource is created, the controller responds by
creating a `Pod`, `Service`, and a `NetworkPolicy` as mentioned above.
These resources are cleaned up when the `RepositoryServer` resource is deleted.

#### Server-side Setup

Once the pod is running, the controller executes a set of Kopia CLI commands as
follows.

> 📝 The examples in this section use S3 for illustration purposes.

- Establish a connection to the Kopia repository. This is equivalent to running
   the following command:

```sh
kopia repository connect s3 \
   --bucket=my-bucket \
   --access-key=<redacted> \
   --secret-access-key=<redacted> \
   [--endpoint=https://foo.example.com \]
   [--prefix=my-prefix \]
   [--region=us-west-1 \]
   --password=<redacted> \
   --override-hostname=<hostname> \
   --override-username=<username>
```

The bucket, endpoint, and region are read from the secret referenced by the
`spec.storage.secret` property while the access-key and secret-access-key are
read from the secret referenced by `spec.location.credentialSecretRef`.
The prefix, username, and hostname values are read from the `spec.repository`
section, and the repository password is derived from the secret referenced by
the `spec.repository.passwordSecretRef` property.

- Start the Kopia repository server as a background process

```sh
kopia server start --address=0.0.0.0:51515 \
   --config-file=/run/kopia/repo.config \
   --log-directory=/run/kopia/log \
   --tls-cert-file=/run/kopia/tls.crt \
   --tls-key-file=/run/kopia/tls.key \
   --server-username=<redacted> \
   --server-password=<redacted> \
   --server-control-username=<redacted> \
   --server-control-password=<redacted> \
   > /dev/null 2>&1 &
```

The `/run/kopia/repo.config` configuration file is generated from the secret
referenced by the `spec.storage.secretRef` and `spec.repository` properties.
See [this](https://kopia.io/docs/reference/command-line/#configuration-file)
documentation and GitHub source
[code](https://github.com/kopia/kopia/blob/ff1653c4d6ee6f729ef16eeb800c98d1b8669b19/repo/local_config.go#L87-L98)
for more information on the configuration file format, and supported
configuration.

The `/run/kopia/tls.crt` and `/run/kopia/tls.key` files contain the TLS x509
certificate and private key read from the secret referenced by
the `spec.server.tlsSecretRef` property.

The credentials for the `--server-username`, `--server-password`,
`--server-control-username` and `--server-control-password` options are read
from the secret referenced by the `spec.server.adminSecretRef` property.

- Register the set of users that have access to the server

```sh
kopia server user add <username>@<hostname> --user-password=<redacted>
```

The username, hostname, and password will be picked up from the
`spec.server.accessSecretRef` property.

- Refresh the server process to enable the newly added users

```sh
kopia server refresh --server-cert-fingerprint=<redacted> \
   --address=0.0.0.0:51515 \
   --server-username=<redacted> \
   --server-password=<redacted>
```

> 📝 The secrets provided in the `RepositoryServer` resource are mounted via the
> pod's `spec.volumes` API.

#### Client-side Setup

The Kopia server is fronted by a K8s `Service` resource. Data mover clients
will connect to it using the equivalence of:

```sh
kopia repository connect server \
   --url=https://<service-name> \
   --config-file=/run/kopia/repo.config \
   --server-cert-fingerprint=<redacted> \
   --override-username=<username> \
   --override-hostname=<hostname> \
   --password=<redacted>
```

The `<service-name>` is the Kopia server's `Service` resource name.
This will be auto-generated by the repository controller and provided via the
`status` subresource of the `RepositoryServer` resource.
The `server-cert-fingerprint` is derived from the TLS certificates provided
during the creation of the server resource and provided via the `status`
subresource.
The username, hostname, and password must match one of the users registered
with the server through the `spec.server.accessSecretRef` property.

Once connected to the server, the data mover clients can utilize the family of
`kopia snapshot` subcommands to manage snapshots.

#### Server Access Users Management

In order for a data mover client to connect to the Kopia server, it needs to
provide [an access username and password](https://kopia.io/docs/repository-server/#configuring-allowed-users---kopia-v08)
for authentication purposes. This section describes the approach to add these
access credentials to the Kopia server.

As mentioned above, when a Kopia server starts, it registers the set of users
defined in the `spec.server.accessSecretRef` property of the `RepositoryServer`
resource.

The permissions of these access users are governed by the Kopia server
[access rules](https://kopia.io/docs/repository-server/#server-access-control-acl).

The secret referenced by the `spec.server.accessSecretRef` property must contain
at least one username/password pair. This secret is mounted to the Kopia server
via the pod's `spec.volumes` API.
The following YAML shows an example of user access credentials that can be used
during the creation of the server resource.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: repository-monitoring-server-access
  namespace: kanister
  labels:
    repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
  <username1>@<hostname1>: <password1>
  <username2>@<hostname2>: <password2>
```

The server also establishes a watch on its access users file. When this file is
updated (due to changes to the underlying secret), the server will also rebuild
its access user list.

### Secrets Management

Instead of assuming full responsibility over the management of different
Kopia credentials, this design proposes the adoption of a shared responsibility
model, where users are responsible for the long-term safekeeping of their
credentials. This model ensures Kanister remains free from a hard dependency on
any crypto packages, and vault-like functionalities.

If misplaced, Kanister will not be able to recover these credentials.


### Replace Kopia CLI with SDK

Currently, we are using Kopia CLI to perform the repository and kopia repository server operations in Kanister.
The repository controller creates a pod, executes commands through `kube.exec` on the pod to perform
repository operations. The commands include: 
- repo connect 
- start server 
- add users 
- refresh server 

Kopia provides an SDK to perform repository operations which can be used instead of CLI. The detailed design is explained in the document
[Replace Kopia CLI with Kopia SDK](https://github.com/kanisterio/kanister/blob/master/design/replace-CLI-with-SDK.md).
