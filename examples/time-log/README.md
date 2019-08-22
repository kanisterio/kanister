# Walkthrough of Time Log

This is an example of using Kanister to protect a simple application called Time Log. This application is contrived but useful for demonstrating Kanister's features. Every second it appends the current time to a log file.

Note: This application is used in the Kanister tutorial, for a detailed walkthrough click [here](https://docs.kanister.io/tutorial.html#tutorial).

### 1. Deploy the Application

The following command deploys the example Time Log application in `default` namespace:
```bash
# Create a deployment whose log we'll ship to s3
$ kubectl apply -f ./examples/time-log/time-logger-deployment.yaml
deployment "time-logger" created
```

### 2. Protect the Application

Next create a Blueprint which describes how backup and restore actions can be executed on this application. The Blueprint for this application can be found at `blueprint.yaml`. In order for this example to work, you should update the S3 compatible bucket details in `s3-profile.yaml` to point to an S3 bucket to which you have access. You should also update `secrets.yaml` to include AWS credentials that have read/write access to the S3 bucket. Provide your AWS credentials by setting the corresponding data values for `aws_access_key_id` and `aws_secret_access_key` in `secrets.yaml`. These are encoded using base64. The following commands will create a Secret, Profile and a Blueprint in controller's namespace:

```bash
# Get base64 encoded aws keys
$ echo -n "YOUR_KEY" | base64

# Create secrets containing the necessary AWS credentials
$ kubectl apply -f examples/time-log/secrets.yaml
secret "aws-creds" created

# Create a profile that will dictate where the backup is stored
$ kubectl apply -f examples/time-log/s3-profile.yaml
profile "s3-profile" created

# Create the kanister blueprint that has instructions on how to backup the log
$ kubectl apply -f examples/time-log/blueprint.yaml
blueprint "time-log-bp" created

```

You can now take a backup of Time Log's data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.
```
# Create the actionset that causes the controller to kick off the backup
$ kubectl --namespace kanister create -f examples/time-log/backup-actionset.yaml
actionset "s3backup-f4c4q" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset s3backup-f4c4q -oyaml
```

### 3. Restore the Application

```bash
$ kanctl --namespace kanister create actionset --action restore --from "s3backup-f4c4q"
actionset "restore-s3restore-g235d-23d2f" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset restore-s3restore-g235d-23d2f -oyaml
```

### 4. Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "s3backup-f4c4q"
actionset "delete-s3backup-f4c4q-2jj9n" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset delete-s3backup-f4c4q-2jj9n -oyaml
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:
```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```
