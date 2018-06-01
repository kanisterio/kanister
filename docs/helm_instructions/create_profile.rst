The command will also configure a location where artifacts resulting
from Kanister data operations such as backup should go. This is stored as a
``profiles.cr.kanister.io`` *CustomResource (CR)* which is then referenced in
Kanister ActionSets. Every ActionSet requires a Profile reference whether one
created as part of the application install or not. Support for creating an
ActionSet as part of install is simply for convenience. This CR can be shared
between Kanister-enabled application instances so one option is to only
create as part of the first instance.

.. note:: Prior to creating the Profile CR, you will need to do the following:

   * Create a bucket for artifacts on your S3 store. This will be your
     ``s3_bucket`` parameter to the command.
   * Obtain ``s3_api_key`` and ``s3_api_secret`` credentials for an
     account with access to the bucket that you will use.
   * Configure the permissions on the bucket to allow the account to
     list, put, get, and delete.
   * Make sure that your retention policy allows deletions so that artifacts
     can be reclaimed based on your intended data backup retention.

.. note:: The ``s3_endpoint`` parameter is only required if you are using an
   S3-compatible provider different from AWS.

   If you are using an on-premises s3 provider, the endpoint specified needs be
   accessible from within your Kubernetes cluster.

   If, in your environment, the endpoint has a self-signed SSL certificate, include
   ``--set kanister.s3_verify_ssl=false`` in the above command to disable SSL
   verification for the S3 operations in the blueprint.
