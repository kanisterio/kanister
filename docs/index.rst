.. kanister documentation master file, created by
   sphinx-quickstart on Tue Nov 28 22:36:58 2017.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

Welcome to Kanister's documentation!
====================================

Kanister allows domain experts to capture application specific data management
tasks in Blueprints which can be easily shared and extended. The framework takes
care of the tedious details around execution on Kubernetes and presents a
homogeneous operational experience across applications at scale.

The design of Kanister was driven by the following main goals:

1. Application-Centric: Given the increasingly complex and distributed nature
   of cloud-native data services, there is a growing need for data management
   tasks to be at the *application* level. Experts who possess domain knowledge
   of a specific application's needs should be able to capture these needs when
   performing data operations on that application.

2. API Driven: Data management tasks for each specific application may vary
   widely, and these tasks should be encapsulated by a well-defined API so as to
   provide a uniform data management experience. Each application expert can
   provide an application-specific pluggable implementation that satisfies this
   API, thus enabling a homogeneous data management experience of diverse and
   evolving data services.

3. Extensible: Any data management solution capable of managing a diverse set of
   applications must be flexible enough to capture the needs of custom data services
   running in a variety of environments. Such flexibility can only be provided if
   the solution itself can easily be extended.


Getting Started
---------------

Kanister is open source and more information can be found on `github
<https://github.com/kanisterio/kanister>`_. We encourage you to start
with the :ref:`architecture` first.

To get up and running using Kanister, we recommend :ref:`install` and
then working through the :ref:`tutorial`.


Quick Start
-----------
The Kanister operator controller can be installed on a `Kubernetes
<https://kubernetes.io>`_ cluster using the `helm <https://helm.sh>`_
package manager.

.. code-block:: bash

   # install Kanister operator controller using helm
   $ helm install stable/kanister-operator

   # install your application
   $ kubectl apply -f examples/mongo-sidecar/mongo-cluster.yaml

   # use an existing Blueprint, tweak one, or create one yourself
   $ kubectl apply -f examples/mongo-sidecar/blueprint.yaml

   # perform operations
   $ kubectl apply -f examples/mongo-sidecar/backup-actionset.yaml



.. toctree::
   :hidden:

   self
   architecture
   install
   tutorial
   functions
   templates
