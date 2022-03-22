# Walkthrough of Picture Gallery

This is an example of using Kanister to protect the volumes of a simple application called Picture Gallery. It demonstrates the use of kanister functions to take individual snapshots of each PVC associated with an appication and to restore the PVCs when needed.

Note: This example demo will only work on AWS cluster for now.

### 1. Deploy the Application

The following command deploys the example Picture Gallery application in `default` namespace:

```bash
# Create a picture gallery deployment
$ kubectl apply -f ./examples/picture-gallery-pvc-snapshot/picture-gallery-deployment.yaml
deployment "picture-gallery" created
```

### 2. Protect the PVCs

Next create a Blueprint which describes how backup and restore actions can be executed on the PVCs of this application. The Blueprint for this application can be found at `blueprint.yaml`. In order for this example to work, you should install kanister profile using the command specified below. Do not forget to replace the default s3 credentials (bucket and region) with your credentials before you run this command. The following commands will install kanister Profile and create a Blueprint in controller's namespace:

```bash
# Configure access to an S3 Bucket
$ helm install profile kanister/profile                 \
    --namespace kanister --create-namespace     \
    --set defaultProfile=true                 \
    --set location.type='s3Compliant' \
    --set location.bucket="my-kanister-bucket"      \
    --set location.region="us-west-2"              \
    --set aws.accessKey="${AWS_ACCESS_KEY_ID}" \
    --set aws.secretKey="${AWS_SECRET_ACCESS_KEY}"

# Create the kanister blueprint that has instructions on how to backup the PVCs
$ kubectl apply -f examples/picture-gallery-pvc-snapshot/blueprint.yaml
blueprint "picture-gallery" created

```

You can now create an ActionSet defining backup action, which will take individual snapshots of every PVC bound to Picture Gallery. The ActionSet should be created in the same namespace as the controller.

```
# Create the actionset that causes the controller to kick off the backup
$ kubectl --namespace kanister create -f examples/picture-gallery-pvc-snapshot/backup-actionset.yaml
actionset "pic-gal-pvc-snapshot--f4c4q" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset pic-gal-pvc-snapshot--f4c4q -oyaml

# Check the controller logs to see if the backup action is complete before moving to next step
$ kubectl --namespace kanister logs -l app=kanister-operator
```

### 3. Restore the PVCs

Now that you have taken the snapshots of all the PVCs, you can test the backup action by running restore command using the `kanctl` tool as shown below. Once the restore is completed, you can see that the PVCs are up and the Picture Gallery pod is running

```bash
# List all PVCs assoicated with Picture Gallery
$ kubectl get pvc

# Restore all PVCs for Picture Gallery
$ kanctl --namespace kanister create actionset --action restore --from "pic-gal-pvc-snapshot-f4c4q"
actionset "restore-pic-gal-pvc-snapshot-hc7tt-6c6fk" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset restore-pic-gal-pvc-snapshot-hc7tt-6c6fk -oyaml

# check the controller logs to see if the restore action is complete before moving to next step
$ kubectl --namespace kanister logs -l app=kanister-operator

#View the status of the PVCs and Picture Gallery pod
$ kubectl get pvc
$ kubectl get pods
```

### 4. Delete the Snapshots

The snapshots created by backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "pic-gal-pvc-snapshot-f4c4q"
actionset "delete-pic-gal-pvc-snapshot-hc7tt-6c6fk" created

# View the status of the actionset
$ kubectl --namespace kanister get actionset delete-pic-gal-pvc-snapshot-hc7tt-6c6fk -oyaml
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```
