Kanister-Enabled MongoDB Replica Set
------------------------------------

This section describes the steps for a basic installation of an instance of
a MongoDB Replica Set (a stateful set with persistent volumes) along with
a Kanister Blueprint and a Profile via a Kanister-enabled Helm chart.

.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample MongoDB replica set application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. only:: kanister

  .. code-block:: rst

     # Install Kanister-enabled MongoDB Replica Set
     $ helm install kanister/kanister-mongodb-replicaset -n mongodb \
          --namespace mongodb-test \
          --set profile.create='true' \
          --set profile.profileName='mongo-test-profile' \
          --set profile.s3.bucket="kanister-bucket" \
          --set profile.s3.endpoint="https://my-custom-s3-provider:9000" \
          --set profile.s3.accessKey="AKIAIOSFODNN7EXAMPLE" \
          --set profile.s3.secretKey="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
          --set kanister.controller_namespace="kanister" \
          --set replicas=1 \
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
   This is only acceptable for test purposes. Information to configure authentication can be found
   `here <https://github.com/kanisterio/kanister/tree/master/examples/helm/kanister/kanister-mongodb-replicaset>`_.

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set persistence.storageClass`` option.
