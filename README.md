# Kanister

## Using the Kanister Operator

### Build Controller

Note: VERSION will be automatically determined from git, but can be overridden.

```bash
# Build the binary inside a container.
make build

# Build and push the docker container to your repo.
make push

```

### Deploy the Operator

```bash
# Use your current kubectl context to deploy the kanister controller
$ make deploy

# Wait for the pod status to be Running
$ kubectl get pod -l app=kanister-operator
NAME                                 READY     STATUS    RESTARTS   AGE
kanister-operator-2733194401-l79mg   1/1       Running   1          12m

# Look at the CRDs
$ kubectl get crd
NAME                        AGE
actionsets.cr.kanister.io   30m
blueprints.cr.kanister.io   30m
```

### Example Applications

#### Time Log

```bash
# Create a deployment whose log we'll ship to s3
kubectl apply -f examples/time-log/time-logger-deployment.yaml

# Create a configmap that will dictate where the log is written
kubectl apply -f examples/time-log/s3-location-configmap.yaml

# Create the kanister blueprint that has instructions on how to backup the log
kubectl apply -f examples/time-log/blueprint.yaml

# Create that actionset that cause the controller to kick off the backup
kubectl apply -f examples/time-log/backup_actionset.yaml
```


#### Mongo w/ Sidecar

Note: Please follow (TODO) to install the modifiend mongodb helm chart.

```bash
# Create the blueprint with the backup implementation.
$ kubectl create -f ./examples/mongo-sidecar/blueprint.yaml

# Take a backup
$ kubectl create -f ./examples/mongo-sidecar/backup_actionset.yaml

# Check the backup progress
$ kubectl get actionsets.kasten.io
```

### Logs

We can get the logs from the controller
```bash
$ kubectl logs -l app=kanister-operator
```

### Cleanup

```bash
kubectl delete -f bundle.yaml
kubectl delete crd {actionsets,blueprints}.cr.kanister.io
```
