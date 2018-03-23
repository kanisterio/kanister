.. _helm:

Kanister Helm Charts
********************

.. contents:: Kanister Helm Charts
  :local:

To make it easier to experiment with Kanister, we have modified a few
upstream Helm charts to add Kanister Blueprints as well as easily
configure the application via Helm. The following sections document
how to install these Kanister-enabled Helm charts. Once installed, you
will need to create :ref:`ActionSets <tutorial>` to perform data
management actions on the data service.

.. include:: s3_config.rst


Kanister Helm Setup
===================

Prior to install you will need to have the Kanister Helm repository added to your
local setup. To do so, please run the following command.

.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/

You also need to install the Kanister controller

.. code-block:: console

   $ helm install --name myrelease --namespace kanister stable/kanister-operator --set image.tag=v0.3.0

Kanister-Enabled Applications
=============================

The following application-specific instructions are available:

.. toctree::
   :maxdepth: 1

   helm_instructions/mysql_instructions.rst
   helm_instructions/mongodb_instructions.rst
