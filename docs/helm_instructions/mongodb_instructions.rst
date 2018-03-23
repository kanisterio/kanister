Kanister-Enabled MongoDB Replica Set
------------------------------------

For basic installation, you can install using a Kanister-enabled Helm
chart that will install an instance of MongoDB Replica Set (a stateful set
with persistent volumes) as well as a Kanister blueprint to be used with it.


.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample MongoDB replica set application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. code-block:: rst

   # Install Kanister-enabled MongoDB Replica Set
   $ helm install kanister/kanister-mongodb-replicaset -n mongodb \
        --namespace mongodb-test \
        --set kanister.s3_endpoint="https://my-custom-s3-provider:9000" \
        --set kanister.s3_api_key="AKIAIOSFODNN7EXAMPLE" \
        --set kanister.s3_api_secret="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
        --set kanister.s3_bucket="kanister-bucket" \
        --set resplicas=1 \
        --set persistentVolume.size=2Gi

The settings in the command above represent the minimum recommended set for
your installation of a single node replica set.

.. note:: The ``s3_endpoint`` parameter is only required if you are using an
  S3-compatible provider different from AWS.

  If you are using an on-premises s3 provider, the endpoint specified needs be
  accessible from within your Kubernetes cluster.

  If, in your environment, the endpoint has a self-signed SSL certificate, include
  ``--set kanister.s3_verify_ssl=false`` in the above command to disable SSL
  verification for the S3 operations in the blueprint.

.. note:: The MongoDB replica set created by the above command will not be secured.
   This is only acceptable for test purposes. If you would like to restrict access,
   please use the ``auth.enabled`` option described below.

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set persistence.storageClass`` option.
