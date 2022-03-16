.. _install:

Installing Kanister
*******************

.. contents:: Installation Overview
  :local:


Prerequisites
=============

* Kubernetes ``1.16`` or higher. For cluster version lower than ``1.16``,
  we recommend installing Kanister version ``0.62.0`` or lower.

* `kubectl <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed
  and setup

* `helm <https://helm.sh>`_ installed and initialized using the command ``helm init``

* :ref:`kanctl <tooling>` installed

* Access to an S3 compatible bucket and credentials.

* Docker (for source-based installs only)


Deploying via Helm
==================

This will install the Kanister controller in the ``kanister`` namespace

.. substitution-code-block:: bash

   # Add Kanister charts
   $ helm repo add kanister https://charts.kanister.io/

   # Install the Kanister operator controller using helm
   $ helm install --name myrelease --namespace kanister kanister/kanister-operator --set image.tag=|version|

   # Create an S3 Compliant Kanister profile using kanctl
   $ kanctl create profile s3compliant --bucket <bucket> --access-key ${AWS_ACCESS_KEY_ID} \
                                       --secret-key ${AWS_SECRET_ACCESS_KEY}               \
                                       --region <region>                                   \
                                       --namespace kanister


Kanister Custom Resource Definitions (CRDs)
===========================================

Kanister defines a few `Custom Resource Definitions (CRDs) <https://docs.kanister.io/architecture.html#custom-resources>`_
to express and execute data protection workflows. By default, these CRDs are
managed (created/updated) by the Kanister controller.

If the Kanister controller doesn't have the permissions to create or
update CRDs, you can disable that using the ``controller.updateCRDs`` Helm
flag. If set to ``false``, Helm will manage the CRDs instead of the Kanister
controller.


Building and Deploying from Source
==================================

Use the following commands to build, package, and deploy the controller to your
Kubernetes cluster. It will push the controller docker image to your docker repo
``<MY REGISTRY>`` and the controller will be deployed in the default namespace.

.. code-block:: bash

   # Build controller binary
   $ make build

   # Package the binary in a docker image and push it to your image registry
   $ make release-controller REGISTRY=<MY REGISTRY>

   # Deploy the controller to your Kubernetes repo
   $ make deploy REGISTRY=<MY REGISTRY>


Deploy a Released Version
-------------------------

To deploy a released version of the controller, issue the command below. Modify
the namespace fields in ``bundle.yaml.in`` to deploy in a namespace of your
choice. By default, the controller will be deployed into the ``default``
namespace.

.. substitution-code-block:: bash

   # Deploy controller version |version| to Kubernetes
   $ make deploy VERSION="|version|"
