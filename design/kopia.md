# Kopia Integration

<!-- toc -->
- [Custom Resource Definition](#custom-resource-definition)
- [Repository Lifecycle](#repository-lifecycle)
  - [Server-Side Setup](#server-side-setup)
  - [Client-Side Setup](#client-side-setup)
- [Server Access Users Management](#server-access-users-management)
  - [Predefined Users List](#predefined-users-list)
  - [Dynamic Users List](#dynamic-users-list)
- [Integration Modes](#integration-modes)
  - [Global Mode](#global-mode)
  - [Namespace Mode](#namespace-mode)
  - [Migration Between Modes](#migration-between-modes)
- [Secrets Management](#secrets-management)
<!-- /toc -->

This document provides design information on concepts and details relevant
to the Kanister's integration with [Kopia][9].

## Custom Resource Definition

This design proposes a new Custom Resource Definition (CRD) named `Repository`,
to represent a [Kopia repository][1]. It is a namespace-scoped resource, owned
and managed by a new `RepositoryController` controller in the `kanister`
namespace. The CRD offers a set of parameters to configure a Kopia repository.

To limit the new controller's RBAC scope, all `Repository` resources are
created in the `kanister` namespace.

A sample `Repository` resource used to store snapshot artifacts of workloads
running in the `monitoring` namespace looks like this:

```yaml
apiVersion: cr.kanister.io/v1alpha1
kind: Repository
metadata:
  name: repository-monitoring
  namespace: kanister
  labels:
    repo.kanister.io/target-namespace: monitoring
spec:
  # immutable
  targetNamespace: monitoring
  location:
    # required
    secretName: repository-monitoring-location
  server:
    # optional - use Kopia's default if omitted
    adminSecretName: repository-monitoring-server-admin
    # optional - no TLS if omitted
    tlsSecretName: repository-monitoring-server-tls
    # optional - use Kopia's default if omitted
    accessSecretNames:
    - repository-monitoring-server-access
  policy:
    addIgnore: ["/data/cache"]
    compression: s2-default
    keepAnnual: 3
    keepDaily: 14
    keepHourly: 48
    keepLatest: 10
    keepMonthly: 24
    keepWeekly: 25
    snapshotInterval: 6h
    snapshotTime: ["00:00","06:00","18:00"]
    # ...
status:
  conditions:
  - lastTransitionTime: "2022-08-20T09:48:36Z"
    lastUpdateTime: "2022-08-20T09:48:36Z"
    status: "True"
    type: RepositoryReady
  - lastTransitionTime: "2022-08-20T09:48:36Z"
    lastUpdateTime: "2022-08-20T09:48:36Z"
    status: "True"
    type: ServerReady
```

The repository contains only snapshots of workloads running in the namespace
specified by the immutable `spec.targetNamespace` property.

The `spec.location.secretName` refers to a `Secret` resource storing location-related
sensitive data:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: repository-monitoring-location
  namespace: kanister
  labels:
    repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
  storage:
    type: s3
    bucket: my-bucket
    endpoint: https://foo.example.com
    prefix: my-prefix
    region: us-west-1
    repo-password: <redacted>
    access-key: <redacted>
    secret-acccess-key: <redacted>
```

The server admin, control access and TLS sensitive data are stored in the
`Secret` resources referenced by the `spec.server.adminSecretName` and
`spec.server.tlsSecretName` properties. E.g.,

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

The `spec.server.accessSecretNames` provides a list of access credentials used
by data mover clients to authenticate with the Kopia server. This is discussed
in more details in the
[Server Access Users Management](#server-access-users-management) section.

The `spec.policy` property can be used to change the snapshot settings of the
respository at the global, user or directory levels. It maps to the
options supported by the `kopia policy set [target]` [command][2].

The `status` subresource provides conditions and status information to ensure
that the controller does not attempt to re-create the repository, nor server
during restart.

## Repository Lifecycle

When a `Repository` resource is created, the `RepositoryController` controller
responds by creating a new Kopia repository using the equivalent of this
command:

```sh
kopia repository create s3 \
  --bucket=my-bucket \
  --access-key=<redacted> \
  --secret-access-key=<redacted> \
  --create-only=true \
  [--endpoint=https://foo.example.com \]
  [--prefix=my-prefix \]
  [--region=us-west-1]
```

> 📝 The examples in this section uses S3 for illustration purposes. Other
> supported storage types include GCS And Azure Blob Storage.

### Server-Side Setup

Once the repository is ready, the controller will schedule a Kopia server
instance to run in the `kanister` namespace, as a `Deployment` workload. The
server is started using the equivalent of this command:

```sh
kopia server start --address=0.0.0.0:51515 \
  --config-file=/run/kopia/repo.config \
  --tls-cert-file=/run/kopia/tls.crt \
  --tls-key-file=/run/kopia/tls.key \
  --server-username=<redacted> \
  --server-password=<redacted> \
  --server-control-username=<redacted> \
  --server-control-password=<redacted> \
```

The `/run/kopia/repo.config` configuration file is generated from the data found
in the secret referenced by the repository's `spec.location.secretName` property.
See the [Kopia documentation][3] and [GitHub source code][4] for more
information on the configuration file format, and supported configuration.

The `/run/kopia/tls.crt` and `/run/kopia/tls.key` files contain the TLS x509
certificate and private key read from the secret referenced by the repository's
`spec.server.tlsSecretName` property.

The credentials for the `--server-username`, `--server-password`,
`--server-control-username` and `--server-control-password` options are read
from the secret referenced by the `spec.server.adminSecretName` property.

### Client-Side Setup

The Kopia server is fronted by a K8s `Service` resource. Data mover clients will
connect to it using the equivalence of:

```sh
kopia repository connect server \
  --url=https://<service-name>.kanister.svc.cluster.local \
  --config-file=/run/kopia/repo.config \
  --server-cert-fingerprint=<redacted> \
  --password=<redacted>
```

The `<service-name>` is the Kopia server's `Service` resource name. By
convention, it includes the repository's `targetedNamespace`. E.g., the
`Service` fronting the Kopia server of a repository serving the
`monitoring` namespace can be `repo-monitoring`. Then its FQDN is
`repo-monitoring.kanister.svc.cluster.local`.

The access username used to authenticate with the Kopia server is added to the
`/run/kopia/repo.config` configuration file, following the
[`repo.ClientOptions`][5] structure.

The content of the configuration file, server certificate fingerprint, and
and access password are exported to the remote data mover clients via the
`KubeExec` functionality over TLS.

Once connected to the server, the data mover clients can utilize the family
of `kopia snapshot` subcommands to manage snapshots.

## Server Access Users Management

In order for a data mover client to connect to the Kopia server, it needs to
provide [an access username and password][6] for authentication purposes. This
section describes two approaches to add these access credentials to the Kopia
server.

### Predefined Users List

When a Kopia server starts, it registers the set of users defined in the
`spec.server.accessSecretNames` property of the `Repository` resource. This
property refers to a list of `Secret` resources containing credentials for the
[server access users][6]. The permissions of these users are governed by the
Kopia server [access rules][7].

This configuration is most suitable for setup models that have their own
workload credential-generation reconcilers.

Each `Secret` contains at least one username/password pairs, keyed off the
workload identifiers.

The following YAML shows an example of access credentials that can be used by
data mover clients in the `monitoring` namespace, to authenticate with the Kopia
server:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: repository-monitoring-server-access
  namespace: kanister
  labels:
    repo.kanister.io/target-namespace: monitoring
type: Opaque
stringData:
  mysql-blue: |
    username: <redacted>
    password: <redacted>
  mysql-green: |
    username: <redacted>
    password: <redacted>
  mysql-staging: |
    username: <redacted>
    password: <redacted>
  pgsql-blue: |
    username: <redacted>
    password: <redacted>
  pgsql-green: |
    username: <redacted>
    password: <redacted>
  pgsql-staging: |
    username: <redacted>
    password: <redacted>
```

When the Kopia server starts, it add these users to its list of access users
with the equivalent of this command:

```sh
kopia server user add <username>@<namespace>.<workload-identifier> \
  --user-password=<password> \
  --password=<repo-password>
```

### Dynamic Users List

To support use cases without predefined users list, the controller can
auto-generate access credentials for workloads which are labeled with the
`repo.kanister.io/authn-mode: service-account` label.

This label selection opt-in mechanism ensures that credentials are created only
for selected stateful workloads, instead of all workloads within the namespace,
many of which may not require data protection.

The username is comprised of the workload's service account name, namespace and
the workload identifier. The password is generated by invoking the K8s
[`TokenRequest` API][8] to issue an API token bound to the workload's
service account.

E.g., given the following `StatefulSet` workload:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
  namespace: data-farm
  labels:
    repo.kanister.io/authn-mode: service-account
spec:
  template:
    spec:
      serviceAccountName: mysql-sa
```

the `RepositoryController` controller then issues a `TokenRequest` request for
an API token bound to the `mysql-sa` service account.

The auto-generated access username is:

```sh
mysql-sa@data-farm.mysql
```

The trust boundary established by this service-account-based approach implies
that all workloads sharing the same service accounts trust each other, and can
access each other's snapshots.

The controller continues to watch for `CREATE`, `UPDATE` and `DELETE` events
of labeled workloads in the cluster, in order to reconcile the server's list of
access credentials accordingly.

## Integration Modes

This design proposes two integration modes with Kopia; namely, `global` and
`namespace`, in order to mitigate the trade-offs between the flexibility and
security of the per-namespace repository model and operational overhead.

The integration mode is decided by a start-up option of the controller named,
`--repository-scope`, with `namespace` and `global` being the only supported
values. If this option is omitted, the controller runs in `global` mode.

### Global Mode

The `global` mode aims at reducing opt-in friction. It's great for
"getting started" scenarios, where there is a pre-established trust between the
workloads and users on the cluster.

In this mode, the controller is started with an universal Kopia server serving
a single repository. The global repository is used to store snapshot artifacts
of all workloads in the cluster. Data mover clients in the cluster also share
the same client access credential.

When the controller starts up, a singleton instance of the `Repository` resource
is created. The Kanister validating webhook ensures that no additional
`Repository` resources can be added afterwards.

This is what a sample singleton `Repository` resource looks like:

```yaml
apiVersion: cr.kanister.io/v1alpha1
kind: Repository
metadata:
  name: repository-global
  namespace: kanister
spec:
  location:
    # required
    secretName: repository-global
  server:
    # optional - use Kopia's default if omitted
    adminSecretName: repository-global-server-admin
    # optional - no TLS if omitted
    tlsSecretName: repository-global-server-tls
    # optional - use Kopia's default if omitted
    accessSecretNames:
    - repository-global-access
  policy:
    addIgnore: ["/data/cache"]
    compression: s2-default
    keepAnnual: 3
    keepDaily: 14
    keepHourly: 48
    keepLatest: 10
    keepMonthly: 24
    keepWeekly: 25
    snapshotInterval: 6h
    snapshotTime: ["00:00","06:00","18:00"]
    # ...
status:
  conditions:
  - lastTransitionTime: "2022-08-20T09:48:36Z"
    lastUpdateTime: "2022-08-20T09:48:36Z"
    status: "True"
    type: RepositoryReady
  - lastTransitionTime: "2022-08-20T09:48:36Z"
    lastUpdateTime: "2022-08-20T09:48:36Z"
    status: "True"
    type: ServerReady
```

There will only be a single
[server access credential](#server-access-users-management) generated from the
service account of the `RepositoryController` controller.

The controller running in this mode is annotated with the
`repo.kanister.io/repo-scope: global` annotation.

### Namespace Mode

In the `namespace` mode, user uses the new `Repository` CRD to create new Kopia
repositories on-demand. Each repository is scoped to a Kubernetes namespace. A
namespace is allowed to have more than one respositories, each pointing to a
different backend storage target.

In this mode, the global repository will not be created.

The controller running in this mode is annotated with the
`repo.kanister.io/repo-scope: namespace` annotation.

### Migration Between Modes

Migration between the two modes are not supported. If a user started Kanister
with Kopia integration in `global` mode, then they can't change Kanister to work
in `namespace` mode during the next upgrade. The same restriction applies to the
reverse setup.

Questions:

* Any guardrails we can implement to prevent the mode switch?

## Secrets Management

Instead of resuming full responsibility over the management of the different
level of Kopia credentials, this design proposes the adoption of a shared
responsibility model, where users are responsible for the long-term safekeeping
of their credentials. This model ensures Kanister remains free from a hard
dependency on any crypto packages, and vault-like functionalities.

If misplaced, Kanister will not be able to recover these credentials.

[1]: https://kopia.io/docs/repositories/
[2]: https://kopia.io/docs/reference/command-line/common/policy-set/
[3]: https://kopia.io/docs/reference/command-line/#configuration-file
[4]: https://github.com/kopia/kopia/blob/ff1653c4d6ee6f729ef16eeb800c98d1b8669b19/repo/local_config.go#L87-L98
[5]: https://github.com/kopia/kopia/blob/ff1653c4d6ee6f729ef16eeb800c98d1b8669b19/repo/local_config.go#L24-L38
[6]: https://kopia.io/docs/repository-server/#configuring-allowed-users---kopia-v08
[7]: https://kopia.io/docs/repository-server/#server-access-control-acl
[8]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#tokenrequest-v1-authentication-k8s-io
[9]: https://kopia.io/
