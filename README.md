![Kanister Logo](./graphic/graphic.png)

# Kanister

[![Go Report Card](https://goreportcard.com/badge/github.com/kanisterio/kanister)](https://goreportcard.com/report/github.com/kanisterio/kanister)
[![Build Status](https://travis-ci.org/kanisterio/kanister.svg?branch=master)](https://travis-ci.org/kanisterio/kanister)


A framework for data management in Kubernetes.  It allows domain experts to
define application-specific data management workflows through Kubernetes API
extensions. Kanister makes it easy to integrate your application's data with
your storage infrastructure.

## Features

- **Tasks Execute Anywhere:** Exec into running containers or spin up new ones.
- **Object Storage:** Efficiently and securely transfer data between your app and
  Object Storage  using Restic.
- **Block Storage:** Backup, restore, and copy data using your storage's APIs.
- **Kubernetes Workload Integration:** Easily perform common workload operations
  like scaling up/down, acting on all mounted PVCs and many more.
- **Application Centric:** A single Blueprint handles workflows for every
  instance of your app.
- **Kubernetes Native APIs:** APIs built using CRDs that play nicely with the
  Kubernetes ecosystem.
- **Secured by RBAC:** Prevent unauthorized access to your workflows using RBAC.
- **Reporting:** Watching, logging and eventing let you know the impact of your
  workflows.

## Community Applications

Stable Helm charts that have been updated with Kanister support.
- **[Cassandra](./examples/stable/cassandra)**
- **[Couchbase](./examples/stable/couchbase)**
- **[Elasticsearch](./examples/stable/elasticsearch)**
- **[FoundationDB](./examples/stable/foundationdb)**
- **[MongoDB](./examples/stable/mongodb)**
- **[MySQL-deploymentconfig](./examples/stable/mysql-deploymentconfig)**
- **[MySQL](./examples/stable/mysql)**
- **[PostgreSQL with Point In Time Recovery (PITR)](./examples/stable/postgresql-wale)**
- **[PostgreSQL](./examples/stable/postgresql)**

## Resources

To get started or to better understand kanister, see the
[documentation](https://docs.kanister.io/).

For troubleshooting help, you can email the [mailing
list](https://groups.google.com/forum/#!forum/kanisterio), reach out to us on
[Slack](https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk), or file a [Github
issue](https://github.com/kanisterio/kanister/issues).

## Presentations

- [SIG Apps Demo](https://youtu.be/uzIp-CjsX1c?t=82)
- [Percona Live 2018](https://www.youtube.com/watch?v=dS0kv0k8D_E)

## License
Apache License 2.0, see [LICENSE](https://github.com/kanisterio/kanister/blob/master/LICENSE).
