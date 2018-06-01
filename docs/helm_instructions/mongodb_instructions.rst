Kanister-Enabled MongoDB Replica Set
------------------------------------

For basic installation, you can install using a Kanister-enabled Helm
chart that will install an instance of MongoDB Replica Set (a stateful set
with persistent volumes) as well as a Kanister blueprint to be used with it.


.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample MongoDB replica set application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. only:: kanister

  .. code-block:: rst

     # Install Kanister-enabled MongoDB Replica Set
     $ helm install kanister/kanister-mongodb-replicaset -n mongodb \
          --namespace mongodb-test \
          --set kanister.create_profile='true' \
          --set kanister.s3_endpoint="https://my-custom-s3-provider:9000" \
          --set kanister.s3_api_key="AKIAIOSFODNN7EXAMPLE" \
          --set kanister.s3_api_secret="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
          --set kanister.s3_bucket="kanister-bucket" \
          --set kanister.controller_namespace="kanister" \
          --set resplicas=1 \
          --set persistentVolume.size=2Gi

.. only:: defaultns

  .. code-block:: rst

     # Install Kanister-enabled MongoDB Replica Set
     $ helm install kanister/kanister-mongodb-replicaset -n mongodb \
          --namespace mongodb-test \
          --set persistentVolume.size=2Gi

The settings in the command above represent the minimum recommended set for
your installation of a single node replica set.

.. only:: kanister

  .. include:: ./create_profile.rst

  If not creating a Profile CR, it is possible to use an even simpler command.

  .. code-block:: rst

     # Install Kanister-enabled MongoDB Replica Set
     $ helm install kanister/kanister-mongodb-replicaset -n mongodb \
          --namespace mongodb-test \
          --set persistentVolume.size=2Gi


.. note:: The MongoDB replica set created by the above command will not be secured.
   This is only acceptable for test purposes. If you would like to restrict access,
   please use the ``auth.enabled`` option described below.

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set persistence.storageClass`` option.
