![Kanister Logo](./graphic/graphic.png)

# Kanister

[![Go Report Card](https://goreportcard.com/badge/github.com/kanisterio/kanister)](https://goreportcard.com/report/github.com/kanisterio/kanister)
[![GitHub Actions](https://github.com/kanisterio/actions/kanister/workflows/main.yaml/badge.svg)](https://github.com/kanisterio/kanister/actions)
[![Build Status](https://travis-ci.com/kanisterio/kanister.svg?branch=master)](https://travis-ci.com/kanisterio/kanister)

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
- **[Cassandra](./examples/cassandra)**
- **[Couchbase](./examples/couchbase)**
- **[Elasticsearch](./examples/elasticsearch)**
- **[FoundationDB](./examples/foundationdb)**
- **[MongoDB on OpenShift using DeploymentConfig](./examples/mongodb-deploymentconfig)**
- **[MongoDB](./examples/mongodb)**
- **[MySQL on OpenShift using DeploymentConfig](./examples/mysql-deploymentconfig)**
- **[MySQL](./examples/mysql)**
- **[PostgreSQL with Point In Time Recovery (PITR)](./examples/postgresql-wale)**
- **[ETCD](./examples/etcd/etcd-in-cluster)**
- **[PostgreSQL on OpenShift using DeploymentConfig](./examples/postgresql-deploymentconfig)**
- **[PostgreSQL](./examples/postgresql)**


## Kanister in action for MySQL Database

[![asciicast](https://asciinema.org/a/303478.svg)](https://asciinema.org/a/303478?speed=1.5)


## Community Meetings

We hold public community meetings, for roadmap and other design discussions, once every two weeks on Thursday at 06:00 PM CET.

- Agenda and meeting notes can be found in [this document](https://docs.google.com/document/d/17LiqwVkeK0MVyfvGwsHPKhaz-nvoaafyAsd7I1R6K3Y/edit?usp=sharing).
- To get yourself added into the regular Community meetings invite, please drop a mail to vivek@kasten.io.
- Meeting joining details can be found in the meeting invite itself.

## Code of Conduct

Kanister is for everyone. We ask that our users and contributors take a few
minutes to review our [Code of Conduct](CODE_OF_CONDUCT.md).

## Security

See [SECURITY.md](SECURITY.md) for our security policy, including how to report
vulnerabilities.

## Resources

To get started or to better understand kanister, see the
[documentation](https://docs.kanister.io/).

For troubleshooting help, you can email the [mailing
list](https://groups.google.com/forum/#!forum/kanisterio), reach out to us on
[Slack](https://join.slack.com/t/kanisterio/shared_invite/enQtNzg2MDc4NzA0ODY4LTU1NDU2NDZhYjk3YmE5MWNlZWMwYzk1NjNjOGQ3NjAyMjcxMTIyNTE1YzZlMzgwYmIwNWFkNjU0NGFlMzNjNTk), or file a [Github
issue](https://github.com/kanisterio/kanister/issues).

## Presentations

- [DoK - Kanister: Application Level Data Operations on Kubernetes](https://www.youtube.com/watch?v=ooJFt0bid1I&t=791s)
- [Kanister Overview 2021 ](https://www.youtube.com/watch?v=wFD42Zpbfts&t=1s)
- [CNCF Webinar - Integrating Backup Into Your GitOps CI/CD Pipeline](https://www.youtube.com/watch?v=2zik5jDjVvM)
- [SIG Apps Demo](https://youtu.be/uzIp-CjsX1c?t=82)
- [Percona Live 2018](https://www.youtube.com/watch?v=dS0kv0k8D_E)

## License

Apache License 2.0, see [LICENSE](https://github.com/kanisterio/kanister/blob/master/LICENSE).
