# Profile CustomResource

Profile CustomResources (CRs) are used to provide configuration information to
[Kanister](https://kanister.io), a framework that enables application-level data
management on Kubernetes.

## TL;DR;

```bash
# Add the Kanister helm repo
$ helm repo add kanister https://charts.kanister.io/

# Create a Profile with the default name in the kanister namespace and AWS credentials set
$ helm install kanister/profile --namespace kanister \
     --set defaultProfile=true \
     --set location.type='s3Compliant' \
     --set aws.accessKey="${AWS_ACCESS_KEY}" \
     --set aws.secretKey="${AWS_SECRET_KEY}" \
     --set location.bucket='my-kanister-bucket'

# Create a Profile with GCP credentials set
$ helm install kanister/profile --namespace kanister \
     --set defaultProfile=true \
     --set location.type='gcs' \
     --set gcp.projectID="my-project-ID" \
     --set-file gcp.serviceKey='path-to-json-file-containing-google-app-credentials' \
     --set location.bucket='my-kanister-bucket'
```

## Overview

This chart installs a Profile CR for [Kanister](http://kanister.io) using the
[Helm](https://helm.sh) package manager.

Profiles provide strongly-typed configuration for Kanister.  Because a Profile
is structured, the Kanister framework is able to provide support for advanced
features. Rather than relying on one-off implementations in Blueprints that
consume ConfigMaps Kanister introspect and use configuration from Profiles.

The schema for Profiles is specified by the CustomResourceDefinition (CRD),
which can be found [here](https://github.com/kanisterio/kanister/blob/master/pkg/apis/cr/v1alpha1/types.go#L234).

Currently Profiles can be used to configure access to object storage compatible
with the [S3 protocol](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html).

## Prerequisites

- Kubernetes 1.7+ with Beta APIs enabled or 1.9+ without Beta APIs.
- Kanister version 0.10.0 with `profiles.cr.kanister.io` CRD installed

> **Note**: The Kanister controller will create the Profile CRD at Startup.

## Configuration

The following table lists the configurable PostgreSQL Kanister blueprint and
Profile CR parameters and their default values. The Profile CR parameters are
passed to the profile sub-chart.

| Parameter        | Description                                                                                                                        | Default   |
| ---              | ---                                                                                                                                | ---       |
| `defaultProfile` | (Optional) Set to ``true`` to create a profile with name `default-profile`.                                                        | ``false`` |
| `profileName`    | (Required if `! defaultProfile`) Name of the Profile CR.                                                                           | `nil`     |
| `aws.accessKey`   | (Required if gcp creds not set) API Key for an s3 compatible object store.                                                                              | `nil`     |
| `aws.secretKey`   | (Required if gcp creds not set) Corresponding secret for `accessKey`.                                                                                   | `nil`     |
| `gcp.projectID`      | (Required if aws creds not set) Project ID of the google application.                                          | `nil`     |
| `gcp.serviceKey`     | (Required if aws creds not set) Path to json file containing google application credentials.                                          | `nil`     |
| `location.type`      | (Optional) Location type: s3Compliant or gcs.                                          | `nil`     |
| `location.bucket`      | (Required if location.type is set) Bucket used to store Kanister artifacts.<br><br>The bucket must already exist.                                          | `nil`     |
| `location.region`      | (Optional) Region to be used for the bucket.                                                                                       | `nil`     |
| `location.endpoint`    | (Optional) The URL for an s3 compatible object store provider. Can be omitted if provider is AWS. Required for any other provider. | `nil`     |
| `verifySSL`      | (Optional) Set to ``false`` to disable SSL verification on the s3 endpoint.                                                        | `true`    |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm
install`. For example:

```bash
$ helm install kanister/profile my-profile-release --namespace kanister \
     --set profileName='my-profile' \
     --set location.type='s3Compliant' \
     --set location.endpoint='https://my-custom-s3-provider:9000' \
     --set aws.accessKey="${AWS_ACCESS_KEY}" \
     --set aws.secretKey="${AWS_SECRET_KEY}" \
     --set location.bucket='my-kanister-bucket' \
     --set verifySSL='true'
```
