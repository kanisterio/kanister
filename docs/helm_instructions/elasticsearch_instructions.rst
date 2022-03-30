Kanister-Enabled Elasticsearch
------------------------------

This section describes the steps for a basic installation of an instance of
Elasticsearch (a stateful set application running master, client and data
nodes with persistent volumes attached to master and data nodes) along with
a Kanister Blueprint and a Profile via a Kanister-enabled Helm chart.

.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample Elasticsearch application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. only:: kanister

  .. code-block:: rst

     # Install Kanister-enabled Elasticsearch
     $ helm install elasticsearch kanister/kanister-elasticsearch \
          --namespace es-test \
          --set profile.create='true' \
          --set profile.profileName='es-test-profile' \
          --set profile.location.type='s3Compliant' \
          --set profile.location.bucket='kanister-bucket' \
          --set profile.location.endpoint='https://my-custom-s3-provider:9000' \
          --set profile.aws.accessKey='AKIAIOSFODNN7EXAMPLE' \
          --set profile.aws.secretKey='wJalrXUtnFEMI%K7MDENG%bPxRfiCYEXAMPLEKEY' \
          --set kanister.controller_namespace="kanister"

.. only:: defaultns

  .. code-block:: rst

     # Install Kanister-enabled Elasticsearch
     $ helm install elasticsearch kanister/kanister-elasticsearch --namespace es-test


The settings in the command above represent the minimum recommended set for
your installation.

.. only:: kanister

  .. include:: ./create_profile.rst

  If not creating a Profile CR, it is possible to use an even simpler command.

  .. code-block:: rst

     # Install Kanister-enabled Elasticsearch
     $ helm install elasticsearch kanister/kanister-elasticsearch elasticsearch --namespace es-test

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set master.persistence.storageClass`` and ``--set data.persistence.storageClass`` option.
