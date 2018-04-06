Kanister-Enabled PostgreSQL
---------------------------

For basic installation, you can install using a Kanister-enabled Helm
chart that will install an instance of PostgreSQL (a Deployment with
persistent volumes) as well as a Kanister blueprint to be used with
it. In particular, this chart uses `WAL-E
<https://github.com/wal-e/wal-e>`_ for continuous archiving of
PostgreSQL WAL files and base backups.


.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample PostgreSQL application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. only:: kanister

  .. code-block:: rst

     # Install Kanister-enabled PostgreSQL
     $ helm install kanister/kanister-postgresql -n postgresql \
          --namespace postgresql-test \
          --set kanister.s3_endpoint="https://my-custom-s3-provider:9000" \
          --set kanister.s3_api_key="AKIAIOSFODNN7EXAMPLE" \
          --set kanister.s3_api_secret="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
          --set kanister.s3_bucket="kanister-bucket" \
          --set kanister.controller_namespace="kanister"


.. only:: defaultns

  .. code-block:: rst

     # Install Kanister-enabled PostgreSQL
     $ helm install kanister/kanister-postgresql -n postgresql \
          --namespace postgresql-test \
          --set kanister.s3_endpoint="https://my-custom-s3-provider:9000" \
          --set kanister.s3_api_key="AKIAIOSFODNN7EXAMPLE" \
          --set kanister.s3_api_secret="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
          --set kanister.s3_bucket="kanister-bucket"


The settings in the command above represent the minimum recommended set for
your installation.

.. warning:: This chart is still in alpha and has known limitations including:

  * Fetching logs and applying them has a timeout value of 100
    minutes. If all logs haven't been fetched and applied in that time
    frame, it is possible for the database to restart with only a
    partial restore.

  * Currently, it only fetches the latest base backup and applies all
    available logs after that. Using Point-In-Time-Recovery (PITR)
    values will be supported in an upcoming release.

  * If ``kanister.s3_endpoint`` is not specified, you are using AWS S3, and the
    S3 bucket is not in the default ``us-east-1`` region, you'll need to include
    the bucket's region using the ``kanister.s3_region`` parameter.

  * More hardening and error-checking is being implemented

.. note:: The ``s3_endpoint`` parameter is only required if you are using an
  S3-compatible provider different from AWS.

  If you are using an on-premises s3 provider, the endpoint specified needs be
  accessible from within your Kubernetes cluster.

  If, in your environment, the endpoint has a self-signed SSL certificate, include
  ``--set kanister.s3_verify_ssl=false`` in the above command to disable SSL
  verification for the S3 operations in the blueprint.

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set persistence.storageClass`` option.
