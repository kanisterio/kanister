Kanister-Enabled MySQL
----------------------

For basic installation, you can install using a Kanister-enabled Helm
chart that will install an instance of MySQL (a deployment with a persistent
volume) as well as a Kanister blueprint to be used with it.


.. code-block:: console

   $ helm repo add kanister https://charts.kanister.io/


Then install the sample MySQL application in its own namespace.

.. For some reason using 'console' or 'bash' highlights the snippet weirdly

.. only:: kanister

  .. code-block:: rst

     # Install Kanister-enabled MySQL
     $ helm install kanister/kanister-mysql -n mysql --namespace mysql-test \
          --set kanister.create_profile='true' \
          --set kanister.s3_endpoint="https://my-custom-s3-provider:9000" \
          --set kanister.s3_api_key="AKIAIOSFODNN7EXAMPLE" \
          --set kanister.s3_api_secret="wJalrXUtnFEMI!K7MDENG!bPxRfiCYEXAMPLEKEY" \
          --set kanister.s3_bucket="kanister-bucket" \
          --set kanister.controller_namespace="kanister" \
          --set mysqlRootPassword="asd#45@mysqlEXAMPLE" \
          --set persistence.size=10Gi

.. only:: defaultns

  .. code-block:: rst

     # Install Kanister-enabled MySQL
     $ helm install kanister/kanister-mysql -n mysql --namespace mysql-test \
          --set mysqlRootPassword="asd#45@mysqlEXAMPLE" \
          --set persistence.size=10Gi


The settings in the command above represent the minimum recommended set for
your installation.

.. only:: kanister

  .. include:: ./create_profile.rst

  If not creating a Profile CR, it is possible to use an even simpler command.

  .. code-block:: rst

     # Install Kanister-enabled MySQL
     $ helm install kanister/kanister-mysql -n mysql --namespace mysql-test \
          --set mysqlRootPassword="asd#45@mysqlEXAMPLE" \
          --set persistence.size=10Gi

.. note:: It is highly recommended that you specify an explicit root password
   for the MySQL application you are installing, even through the chart supports
   auto-generating a password. This will prevent future issues if you decide
   to use ``helm update`` to make changes to the application setup.

.. note:: The above command will attempt to use dynamic storage provisioning
   based on the the default storage class for your cluster. You will to need to
   `designate a default storage class <https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass>`_
   or, use a specific storage class by providing a value with the
   ``--set persistence.storageClass`` option.
