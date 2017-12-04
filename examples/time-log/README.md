# Walkthrough of Time Log

This is an example of using Kanister to protect a simple application called Time Log. This application is contrived but useful for demonstrating Kanister's features. Every second it appends the current time to a log file.

Note: This application is used in the Kanister tutorial, for a detailed walkthrough click [here](https://docs.kanister.io/tutorial.html#tutorial).

### 1. Deploy the Application

```bash
# Create a deployment whose log we'll ship to s3
$ kubectl apply -f ./examples/time-log/time-logger-deployment.yaml
deployment "time-logger" created
```

### 2. Protect the Application

Next we use a blueprint to protect the application. Since this blueprint references a config map and secrets, we first need to create those. You should update s3-location-configmap.yaml to point to an S3 bucket to which you have access. You should also update secrets.yaml to include your AWS credentials that provide read/write access to the bucket you specified in s3-location-configmap.yaml. TODO: Add secrets yaml file

```bash
# Create a configmap that will dictate where the log is written
$ kubectl apply -f examples/time-log/s3-location-configmap.yaml
configmap "s3-location" created

# Create secrets containing the necessary AWS credentials
$ kubectl apply -f examples/time-log/secrets.yaml
secret "aws-creds" created

# Create the kanister blueprint that has instructions on how to backup the log
$ kubectl apply -f examples/time-log/blueprint.yaml
blueprint "time-log-bp" created

# Create that actionset that causes the controller to kick off the backup
$ kubectl create -f examples/time-log/backup-actionset.yaml
actionset "s3backup-f4c4q" created

# View the status of the actionset
kubectl get actionset s3backup-f4c4q -oyaml
```

### 3. Restore the Application

```bash
$ kanctl perform from restore "s3backup-f4c4q"
actionset "restore-s3restore-g235d-23d2f" created

# View the status of the actionset
kubectl get actionset restore-s3restore-g235d-23d2f -oyaml
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:
```bash
kubectl logs -l app=kanister-operator
```
