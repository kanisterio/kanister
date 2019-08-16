.. _overview:

Kanister Overview
*****************

.. contents:: Kanister Overview
  :local:

Design Goals
============

The design of Kanister was driven by the following main goals:

1. **Application-Centric:** Given the increasingly complex and distributed nature
   of cloud-native data services, there is a growing need for data management
   tasks to be at the *application* level. Experts who possess domain knowledge
   of a specific application's needs should be able to capture these needs when
   performing data operations on that application.

2. **API Driven:** Data management tasks for each specific application may vary
   widely, and these tasks should be encapsulated by a well-defined API so as to
   provide a uniform data management experience. Each application expert can
   provide an application-specific pluggable implementation that satisfies this
   API, thus enabling a homogeneous data management experience of diverse and
   evolving data services.

3. **Extensible:** Any data management solution capable of managing a diverse set of
   applications must be flexible enough to capture the needs of custom data services
   running in a variety of environments. Such flexibility can only be provided if
   the solution itself can easily be extended.


Getting Started
===============

To get up and running using Kanister, we encourage you to start with
the :ref:`architecture` section first. :ref:`install` will then allow
you to work through the :ref:`tutorial`.

Alternatively, you can start by installing :ref:`Kanister-enabled Helm
applications<helm>` and then using Kanister to manipulate them. See
the Quick Start section below and the documentation on :ref:`helm` for
more information.

Kanister is an open-source project and the source, this documentation,
and more examples can be found on `GitHub
<https://github.com/kanisterio/kanister>`_.



Quick Start
===========

The Kanister operator controller can be installed on a `Kubernetes
<https://kubernetes.io>`_ cluster using the `Helm <https://helm.sh>`_
package manager.

The following commands will install Kanister, Kanister-enabled MySQL
and backup to an AWS S3 bucket.

.. code-block:: bash

  # Add Kanister Charts
  helm repo add kanister http://charts.kanister.io

  # Install the Kanister Controller
  helm install --name myrelease --namespace kanister kanister/kanister-operator --set image.tag=0.20.0

  # Install MySQL and configure its Kanister Blueprint.
  # Also create a Profile CR that can be used in ActionSets
  helm install kanister/kanister-mysql                                      \
      --name mysql-release --namespace mysql-ns                             \
      --set profile.create='true'                                           \
      --set profile.location.type='s3Compliant'                             \
      --set profile.location.bucket='mysql-backup-bucket'                   \
      --set profile.location.endpoint='https://my-custom-s3-provider:9000'  \
      --set profile.aws.accessKey="${AWS_ACCESS_KEY_ID}"                    \
      --set profile.aws.secretKey="${AWS_SECRET_ACCESS_KEY}"                \
      --set kanister.controller_namespace=kanister

  # Perform a backup by creating an ActionSet
  cat << EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: mysql-backup-
    namespace: kanister
  spec:
    actions:
    - name: backup
      blueprint: mysql-release-kanister-mysql-blueprint
      object:
        kind: Deployment
        name: mysql-release-kanister-mysql
        namespace: mysql-ns
      profile:
        apiVersion: v1alpha1
        kind: profile
        name: mysql-release-kanister-mysql-backup-profile
        namespace: mysql-ns
  EOF
