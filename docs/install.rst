.. _install:

Installing Kanister
===================

.. contents:: Installation Overview
  :local:

Quickstart 
----------

This will install the controller in the default namespace

.. code-block:: bash

   # pull the kanister repo
   $ git clone git@github.com:kanisterio/kanister.git

   # install Kanister operator controller in the default namepsace
   $ kubectl apply -f kanister/bundle.yaml


Prerequisites
-------------

* Kubernetes 1.7 or higher

* `kubectl <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed
  and setup

* docker

Building and Deploying from Source
----------------------------------

Use the following commands to build, package, and deploy the controller to your
Kubernetes cluster. It will push the controller docker image to your docker repo
`"<MY REGISTRY>"` and the controller will be deployed in the default namespace.

.. code-block:: bash

   # build controller binary
   $ make build

   # package the binary in a docker image and push it to your image registry
   $ make push REGISTRY=<MY REGISTRY>

   # deploy the controller to your Kubernetes repo
   $ make deploy REGISTRY=<MY REGISTRY>


Deploy a Released Version
+++++++++++++++++++++++++

To deploy a released version of the controller, issue the command below. Modify
the namespace fields in `bundle.yaml.in` to deploy in a namespace of your
choice. By default, the controller will be deployed into the `default`
namespace.

.. code-block:: bash

   # deploy controller version 0.1.0 to Kubernetes
   $ make deploy VERSION="0.1.0"

