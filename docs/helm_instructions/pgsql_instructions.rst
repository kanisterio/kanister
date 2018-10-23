Kanister-Enabled PostgreSQL
---------------------------

This section describes the steps for a basic installation of an instance of
PostgreSQL (a Deployment with persistent volumes) along with
a Kanister Blueprint and a Profile via a Kanister-enabled Helm chart.
In particular, this chart uses `WAL-E <https://github.com/wal-e/wal-e>`_
for continuous archiving of PostgreSQL WAL files and base backups.


.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample PostgreSQL application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. only:: kanister

  .. code-block:: rst

     # Install Kanister-enabled PostgreSQL
     $ helm install kanister/kanister-postgresql -n postgresql \
          --namespace postgresql-test \
          --set profile.create='true' \
          --set profile.profileName='postgres-test-profile' \
          --set profile.s3.bucket="kanister-bucket" \
          --set profile.s3.endpoint="https://my-custom-s3-provider:9000" \
          --set profile.s3.accessKey="AKIAIOSFODNN7EXAMPLE" \
          --set profile.s3.secretKey="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
          --set kanister.controller_namespace="kanister"


.. only:: defaultns

  .. code-block:: rst

     # Install Kanister-enabled PostgreSQL
     $ helm install kanister/kanister-postgresql -n postgresql \
          --namespace postgresql-test

The settings in the command above represent the minimum recommended set for
your installation.

.. warning:: This chart is still in alpha and has known limitations including:

  * Fetching logs and applying them has a timeout value of 10
    hours. If all logs haven't been fetched and applied in that time
    frame, it is possible for the database to restart with only a
    partial restore.

  * More hardening and error-checking is being implemented

.. only:: kanister

  .. include:: ./create_profile.rst

  If not creating a Profile CR, it is possible to use an even simpler command.

  .. code-block:: rst

     # Install Kanister-enabled PostgreSQL
     $ helm install kanister/kanister-postgresql -n postgresql \
          --namespace postgresql-test

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set persistence.storageClass`` option.
