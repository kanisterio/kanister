S3 Configuration
================

The Kanister blueprints provided use an S3-compatible object store
to manage data artifacts. You will need to do the following before
installing the Kanister-enabled version of MySQL or MongoDB below:

* Create a bucket for artifacts on your S3 store.
* Discover the endpoint URL for your object store. You don't need this
  if you are using AWS S3.
* Obtain ``s3_api_key`` and ``s3_api_secret`` credentials for an
  account with access to the bucket that you will use.
* Configure the permissions on the bucket to allow the account to
  list, put, get, and delete.
* Make sure that your retention policy allows deletions so that artifacts
  can be reclaimed based on your intended data backup retention.
