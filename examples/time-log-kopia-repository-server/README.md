# Protecting TimeLog application

This is an example of using Kanister to protect a simple TimeLog application. This application is contrived but useful for demonstrating Kanister's features. Every second it appends the current time to a log file.

Note: This application is used in the Kanister tutorial, for a detailed walkthrough click [here](https://docs.kanister.io/tutorial.html#tutorial).

### 1. Deploy the Application

The following command deploys the example Time Log application in `time-log` namespace:

```bash
$ kubectl create namespace time-log

# Create a deployment whose log we'll ship to s3 using Kanister functions (that would eventually use Kopia repository server)
$ kubectl apply -f ./examples/time-log/time-logger-deployment.yaml -n time-log
deployment "time-logger" created

### 2. Protect the Application

#### Create Repository Server CR

Create Kopia Repository Server CR if not created already

- Create Kopia Repository using S3 as the location storage

```bash
$ kopia --log-level=error --config-file=/tmp/kopia-repository.config \
--log-dir=/tmp/kopia-cache repository create --no-check-for-updates \
--cache-directory=/tmp/cache.dir --content-cache-size-mb=0 \
--metadata-cache-size-mb=500 --override-hostname=mysql.app \
--override-username=kanisterAdmin s3 --bucket=<s3_bucket_name> \
--prefix=/repo-controller/ --region=<s3_bucket_region> \
--access-key=<aws_access_key> --secret-access-key=<aws_secret_access_key> --password=<repository_password>
```

- Generate TLS Certificates and create TLS secret for Kopia Repository Server for secure communication between Kopia Repository Server and Client

```bash
$ openssl req -newkey rsa:2048 -nodes -keyout key.pem -x509 -days 365 -out certificate.pem

$ kubectl create secret tls repository-server-tls-cert --cert=certificate.pem --key=key.pem -n kanister
```

- Create Location Secrets for Kopia Repository

```bash
# The following file s3_location_creds.yaml is a sample file for creating s3 credentials secrets. It contains the credentials for accessing the s3 bucket.
$ vi s3_location_creds.yaml

apiVersion: v1
kind: Secret
metadata:
   name: s3-creds
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: secrets.kanister.io/aws
data:
   # required: base64 encoded value for key with proper permissions for the bucket
   aws_access_key_id: <base64_encoded_aws_access_key>
   # required: base64 encoded value for the secret corresponding to the key above
   aws_secret_access_key: <base64_encoded_aws_secret_access_key>
```

```
$ kubectl create -f s3_location_creds.yaml -n kanister
```

```bash
# The following file s3_location.yaml is a sample file for creating s3 location secrets. It contains the details of the s3 bucket.
$ vi s3_location.yaml

apiVersion: v1
kind: Secret
metadata:
   name: s3-location
   namespace: kanister
   labels:
      repo.kanister.io/target-namespace: monitoring
type: Opaque
data:
   # required: specify the type of the store
   # supported values are s3, gcs, azure, and file-store
   type: czM=
   bucket: <base64_encoded_s3_bucket_name>
   # optional: used as a sub path in the bucket for all backups
   path: <base_64_encoded_prefix_provided_when_creating_kopia_repository>
   # required, if supported by the provider
   region: <base64_encoded_s3_bucket_region>
   # optional: if set to true, do not verify SSL cert.
   # Default, when omitted, is false
   #skipSSLVerify: false
   # required: if type is `file-store`
   # optional, otherwise
   #claimName: store-pvc
```

```
$ kubectl create -f s3_location.yaml -n kanister
```

- Apply Secrets for Kopia Repository Server User Access, Admin Access and Repository Access

```bash
# The following command creates secrets for kopia repository server user access.
kubectl create secret generic repository-server-user-access -n kanister --from-literal=localhost=<suitable_password_for_repository_server_user>

# The following command creates secrets for kopia repository server admin access.
kubectl create secret generic repository-admin-user -n kanister --from-literal=username=<suitable_admin_username_for_repository_server> --from-literal=password=<suitable_password_for_repository_server_admin>

# The following command creates secrets for kopia repository access.
kubectl create secret generic repo-pass -n kanister --from-literal=repo-password=<repository_password_set_while_creating_kopia_repository>
```

- Create Repository Server CR

```bash
vi repo-server-cr.yaml 
```
```
apiVersion: cr.kanister.io/v1alpha1
kind: RepositoryServer
metadata:
  labels:
    app.kubernetes.io/name: repositoryserver
    app.kubernetes.io/instance: repositoryserver-sample
    app.kubernetes.io/part-of: kanister
    app.kuberentes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kanister
  name: kopia-repo-server-1
  namespace: kanister
spec:
  storage:
    secretRef:
      name: s3-location
      namespace: kanister
    credentialSecretRef:
      name: s3-creds
      namespace: kanister
  repository:
    rootPath: /repo-controller/
    passwordSecretRef:
      name: repo-pass
      namespace: kanister
    username: kanisterAdmin
    hostname: time-log.app
  server:
    adminSecretRef:
      name: repository-admin-user
      namespace: kanister
    tlsSecretRef:
      name: repository-server-tls-cert
      namespace: kanister
    userAccess:
      userAccessSecretRef:
        name: repository-server-user-access
        namespace: kanister
      username: kanisteruser
```

```bash
$ kubectl create -f repo-server-cr.yaml -n kanister
```
**NOTE:**

Make Sure the Repository Server is in ServerReady State before creating actionsets.
You could check the status of the Repository Server CR by running following command
```bash
$ kubectl get repositoryservers.cr.kanister.io kopia-repo-server-1 -n kanister -o yaml
```

**NOTE:**

The above command will configure a kopia repository server, which manages artifacts resulting from Kanister
data operations such as backup.
This is stored as a `repositoryservers.cr.kanister.io` *CustomResource (CR)* which is then referenced in Kanister ActionSets.

#### Create Blueprint

Create Blueprint in the same namespace as the Kanister controller

```bash
$ kubectl create -f ./examples/time-log/time-log-blueprint.yaml -n kanister
```

You can now take a backup of Time Log's data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.
```
# Create the actionset that causes the controller to kick off the backup
$ kanctl create actionset --action backup --namespace kanister --blueprint time-log --deployment time-log/time-logger --repository-server kanister/kopia-repo-server-1

actionset "s3backup-f4c4q" created

# View the status of the actionset

$ kubectl describe actionsets.cr.kanister.io -n kanister s3backup-f4c4q -oyaml
```

### 3. Restore the Application

```bash
$ kanctl -n kanister create actionset --action restore --from "s3backup-f4c4q"

Warning: Neither --profile nor --repository-server flag is provided.
Action might fail if blueprint is using these resources.
actionset "restore-s3restore-g235d-23d2f" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset restore-s3restore-g235d-23d2f -oyaml
```

### 4. Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "s3backup-f4c4q"

Warning: Neither --profile nor --repository-server flag is provided.
Action might fail if blueprint is using these resources.
actionset "delete-s3backup-f4c4q-2jj9n" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset delete-s3backup-f4c4q-2jj9n -oyaml
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:
```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```
