.. _tooling:

Kanister Tooling
****************

.. contents:: Kanister Tooling
  :local:

There are two command-line tools that are built within the Kanister repository.

Kanctl
======

Although all Kanister actions can be run using kubectl, there are situations
where this may be cumbersome. Many actions depend on the Artifacts created by
another action. The canonical example is backup/restore. Manually creating a
restore ActionSet requires copying Artifacts from the status of the complete
backup ActionSet, which is an error prone process.

`kanctl` helps make running dependent ActionSets more robust.  Kanctl is a
command-line tool that makes it easier to create ActionSets.

To demonstrate backup/restore ActionSet chaining, we'll perform "`kanctl perform
<action> --from`".

.. code-block:: bash

  $ kanctl perform -h
  Perform an action on the artifacts from <parent>

  Usage:
    kanctl perform <action> [flags]

  Flags:
    -f, --from string   specify name of the action set(required)
    -h, --help          help for perform

  Global Flags:
    -n, --namespace string   Override namespace obtained from kubectl context

.. code-block:: bash

  # perform backup
  $ kubectl --namespace kanister create -f examples/time-log/backup-actionset.yaml
  actionset "s3backup-j4z6f" created

  # restore from the backup we just created
  $ kanctl --namespace kanister perform restore --from s3backup-j4z6f
  actionset "restore-s3backup-j4z6f-s1wb7" created

  # View the actionset
  kubectl --namespace kanister get actionset restore-s3backup-j4z6f-s1wb7 -oyaml

Similarly, we can also delete the backup file using the following `kanctl` command

.. code-block:: bash

  # delete the backup we just created
  $ kanctl --namespace kanister perform delete --from s3backup-j4z6f
  actionset "delete-s3backup-j4z6f-2jj9n" created

  # View the actionset
  $ kubectl --namespace kanister get actionset delete-s3backup-j4z6f-2jj9n -oyaml


Kando
=====

A common use case for Kanister is to transfer data between Kubernetes and an
object store like AWS S3. We've found it can be cumbersome to pass Profile
configuration to tools like the AWS command line from inside Blueprints.

`kando` is a tool to simplify object store interactions from within blueprints.
It has two commands:

* `location push`

* `location pull`

The usage for these commands can be displayed using the `--help` flag:

.. code-block:: console

  $ kando location pull --help
  Pull from s3-compliant object storage to a file or stdout

  Usage:
    kando location pull <target> [flags]

  Flags:
    -h, --help   help for pull

  Global Flags:
    -s, --path string      Specify a path suffix (optional)
    -p, --profile string   Pass a Profile as a JSON string (required)

.. code-block:: console

  $ kando location push --help
  Push a source file or stdin stream to s3-compliant object storage

  Usage:
    kando location push <source> [flags]

  Flags:
    -h, --help   help for push

  Global Flags:
    -s, --path string      Specify a path suffix (optional)
    -p, --profile string   Pass a Profile as a JSON string (required)

The following snippet is an example of using kando from inside a Blueprint.

.. code-block:: console

  kando location push --profile '{{ .Profile }}' --path '{{ .ArtifactsOut }}' -


Docker Image
============

These tools, especially `kando` are meant to be invoked inside containers via
Blueprints. Although suggest using the released image when possible, we've also
made it simple to add these tools to your container.

The released image, `kanisterio/kanister-tools:0.10.0`, is hosted by
`dockerhub <https://cloud.docker.com/swarm/kanisterio/repository/docker/kanisterio/kanister-tools/general>`_.

The Dockerfile for this image is in the
`kanister github repo <https://github.com/kanisterio/kanister/blob/master/docker/tools/Dockerfile>`_.

To add these tools to your own image, you can add the following command to your
Dockerfile:

.. code-block:: console

    RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
