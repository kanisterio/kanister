This document is going to show you how you can backup the ETCD cluster that is running as part of your Kubernetes control plane. The
commands are run into a cluster that is setup using [Kubeadm](https://github.com/kubernetes/kubeadm) but it should work on any other single or multi node ETCD cluster.

**Note**

This blueprint will only work if the ETCD pod has `tar` binary available
because `kubectl cp` command in the blueprint, requires `tar` binary to be
present on the ETCD pod.

## Prerequisites Details

* Kubernetes 1.9+ with Beta APIs enabled, and you are not on managed Kubernetes
* PV support on the underlying infrastructure
* Kanister version 0.32.0 with `profiles.cr.kanister.io` CRD, [`kanctl`](https://docs.kanister.io/tooling.html#install-the-tools) Kanister tool installed

# Integrating with Kanister

Once we have made sure that the prequisites are met, when we say integrating with Kanister we mean creating some CRs, for example Blurprint and Actionset
that would help perform the actions on the the ETCD instance that we are running.

## Create Profile resource

```bash
» kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <bucket-name> --region <region-name> \
        --namespace kanister
secret 's3-secret-7umv91' created
profile 's3-profile-nnvmm' created
```
This command creates a profile which we will use later.

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

## Create Blueprint

If you are running ETCD on production, there would be some authentication mechanism that your cluster is using. Since we are taking an example
of the cluster that is setup through [kubeadm](https://github.com/kubernetes/kubeadm), I am assuming TLS based authentication is being used.

To specify the location of the CA, certificate, key and other details, we will have to create a secret in a new or an existing namespace. We are free
to choose the name and namespace of the secret and it should have below fields

- **cacert** : CA (certificate authority) cert, would usually be `/etc/kubernetes/pki/etcd/ca.crt` on Kubeadm clusters

- **cert** : Certificate that is used to secure the ETCD cluster, would usually be `/etc/kubernetes/pki/etcd/server.crt` on Kubeadm clusters

- **endpoints** : ETCD server client listen URL, https://[127.0.0.1]:2379

- **key** : TLS key file, would be `/etc/kubernetes/pki/etcd/server.key` in case of Kubeadm cluster

- **labels** : Labels using which Kanister will identify the etcd members in a namespace, in case of multi member etcd cluster

- **etcdns** : Namespace in which the etcd pods are running


```
# Create a namespace where we are going to have the secret created
» kubectl create ns etcd-backup

# Create secret with all the details
» kubectl create secret generic etcd-kube-system \
    --from-literal=cacert=/etc/kubernetes/pki/etcd/ca.crt \
    --from-literal=cert=/etc/kubernetes/pki/etcd/server.crt \
    --from-literal=endpoints=https://127.0.0.1:2379 \
    --from-literal=key=/etc/kubernetes/pki/etcd/server.key \
    --from-literal=labels=component=etcd,tier=control-plane \
    --from-literal=etcdns=kube-system \
    --namespace etcd-backup
secret/etcd-kube-system created
```

**Note**

Please make sure that you have correct path of these certificate files. If any of the path is incorrect the etcd snapshot will fail.
These paths can be found either by describing the running ETCD pod or looking at the static pod's manifest files. The static pod's manifest
files would most probably be in `/etc/kubernetes/manifests`.

Once secret is created, let's go ahead and create Blueprint in the same namespace as the Kanister controller

```
» kubectl create -f etcd-incluster-blueprint.yaml -n kanister
blueprint.cr.kanister.io/etcd-blueprint created
```

## Create test namespace

Now we can create a test namespace, and delete it after taking the ETCD backup so that we can make sure that the namespace is restored
after we restore the ETCD.

```
» kubectl create namespace nginx
namespace/nginx created

» kubectl create deployment nginx -n nginx --image nginx
deployment.apps/nginx created

» kubectl get all -n nginx
NAME                         READY   STATUS    RESTARTS   AGE
pod/nginx-86c57db685-ztb7l   1/1     Running   0          37s

NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nginx   1/1     1            1           40s

NAME                               DESIRED   CURRENT   READY   AGE
replicaset.apps/nginx-86c57db685   1         1         1       40s
```

## Protect the Application

We can now take snapshot of the ETCD server that is running by creating backup actionset that is going to execute backup phase from the blueprint that we have
created above

**Note**

Please make sure to change the **profile name**, **blueprint name**, **secret name** and **secret-namespace** in the `backup-actionset.yaml` manifest file, where secret-name is the name of secret that has all the details.

```
# find the profile name
» kubectl get profile -n kanister
NAME               AGE
s3-profile-nnvmm   54s

# find blueprint name
» kubectl get blueprint -n kanister
NAME             AGE
etcd-blueprint   41s

# create actionset
» kubectl create -f backup-actionset.yaml -n kanister
actionset.cr.kanister.io/backup-hnp95 created

# you can check the status of the actionset by describing it and making sure that it is succeeded
» kubectl describe actionsets.cr.kanister.io -n kanister backup-hnp95

# You should see events like this
...
...
Events:
  Type    Reason           Age   From                 Message
  ----    ------           ----  ----                 -------
  Normal  Started Action   34s   Kanister Controller  Executing action backup
  Normal  Started Phase    34s   Kanister Controller  Executing phase takeSnapshot
  Normal  Ended Phase      34s   Kanister Controller  Completed phase takeSnapshot
  Normal  Started Phase    34s   Kanister Controller  Executing phase uploadSnapshot
  Normal  Ended Phase      0s    Kanister Controller  Completed phase uploadSnapshot
  Normal  Started Phase    0s    Kanister Controller  Executing phase removeSnapshot
  Normal  Ended Phase      0s    Kanister Controller  Completed phase removeSnapshot
  Normal  Update Complete  0s    Kanister Controller  Updated ActionSet 'backup-hnp95' Status->complete
```

Once the backup actionset is complete, we can check the object storage to make the backup is uploaded successfully.

## Imitate Disaster

Now let's assume something went wrong and we lost the test namespace that we created eariler. Please delete the namespace manually

```
» kubectl delete ns nginx
namespace "nginx" deleted

» kubectl get all -n nginx
No resources found in nginx namespace.
```

## Restore the ETCD cluster

To restore the ETCD cluster you will have to have the backup location, where the backup action uploaded the snapshot so that you can download the snapshot. To figure out
the backup location describe the actionset to check the output of the `uploadSnapshot` phase.

Below command can be used to get the backup path from the backup actionset, using the way that is described above

```
kubectl get actionsets.cr.kanister.io -n kanister backup-nfw5g -ojsonpath='{.status.actions[?(@.name=="backup")].phases[?(@.name=="uploadSnapshot")].output.backupLocation}'
# which gives us
etcd_backups/kube-system/etcd-ubuntu-s-4vcpu-8gb-blr1-01-master-1/2020-08-07T11:21:23Z/etcd-backup.db.gz
```

Once we have the backup location, we can go ahead with manually restoring the ETCD. SSH into the node where ETCD is running, most usually it would be Kubernetes leader node.

These tools should be installed on the leader node
- Based on the object storage that you used, you should have CLI installed, in our case since we are using AWS S3 as oject storage make sure aws CLI is installed
- ETCD command line tool `etcdctl`

Download the backup using the backup location that we figured out in previous step and download the ETCD snapshot using below command

```
aws s3 cp  s3://<bucket-name>/etcd_backups/kube-system/etcd-ubuntu-s-4vcpu-8gb-blr1-01-master-1/2020-08-07T11:21:23Z/etcd-backup.db.gz ./
download: s3://<bucket-name>/etcd_backups/kube-system/etcd-ubuntu-s-4vcpu-8gb-blr1-01-master-1/2020-08-07T11:21:23Z/etcd-backup.db.gz to ./etcd-backup.db.gz
```
Once we have the snapshot we can restore the snapshot using the below command to a new dir lets say `/var/lib/etcd-from-backup`

```
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  --data-dir="/var/lib/etcd-from-backup" \
  --initial-cluster="ubuntu-s-4vcpu-8gb-blr1-01-master-1=https://127.0.0.1:2380" \
  --name="ubuntu-s-4vcpu-8gb-blr1-01-master-1" \
  --initial-advertise-peer-urls="https://127.0.0.1:2380" \
  --initial-cluster-token="etcd-cluster-1" \
  snapshot restore /tmp/etcd-backup.db
2020-08-07 12:09:05.626175 I | mvcc: restore compact to 153873
2020-08-07 12:09:05.641147 I | etcdserver/membership: added member e92d66acd89ecf29 [https://127.0.0.1:2380] to cluster 7581d6eb2d25405b
```

and after that the etcd snapshot should have been restored into the new dir that we provided i.e. `/var/lib/etcd-from-backup`. And we will just have
to instruct the ETCD that is running to use this new dir instead of the dir that it uses by default.
To do that open the static pod manifest for ETCD, that would be `etcd.yaml` in the dir `/etc/kubernetes/manifests` and
- change the `data-dir` for the etcd container's command to have `/var/lib/etcd-from-backup`
- add another argument in the command `--initial-cluster-token=etcd-cluster-1 ` as we have seen in the restore command
- change the volume (named `etcd-data`) to have new dir `/var/lib/etcd-from-backup`
- change volume mount (named `etcd-data`) to new dir `/var/lib/etcd-from-backup`

once you save this manifest, new ETCD pod will be created with new data dir. Please wait for the ETCD pod to be up and running.

Once you see the etcd pod in `kube-system` namespace is running fine you can list all the resource from the our test namespace once again
to make sure it has been restored successfully.

```
» kubectl get all -n nginx
NAME                         READY   STATUS    RESTARTS   AGE
pod/nginx-86c57db685-ztb7l   1/1     Running   0          9m

NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nginx   1/1     1            1           10m

NAME                               DESIRED   CURRENT   READY   AGE
replicaset.apps/nginx-86c57db685   1         1         1       10m
```

and as you can see the workload of the namespace is back to its previos state.
If you go ahead and describe the etcd pod from kube-system namespace, you would be able to see that the new dir that we provided (i.e. `/var/lib/etcd-from-backup`) is being used
as the etcd data dir.

### Restoring ETCD snapshot in case of Multi Node ETCD cluster

If your Kubernetes cluster is setup in such a way that you have more than one memeber of ETCD up and running, you will have to follow almost the same steps that we have
already seen with some minor changes.
So you have one snapshot file from backup and as the [ETCD documentation](https://etcd.io/docs/v3.4.0/op-guide/recovery/) says all the members should restore from the same snapshot. What we would do is choose one leader node that we will be using to restore the backup that we have taken and stop the static pods from all other leader nodes.
To stop the static pods from other leader nodes you will have to move the static pod manifests from the static pod path, which in case of kubeadm is `/etcd/kubernetes/manifests`.
Once you are sure that the containers on the other follower nodes have been stopped, please follow the step that is mentioned previously (`Restore the ETCD cluster`) on all the leader nodes sequentially.

If we take a look into the bellow command that we are actually going to run to restore the snapshot

```
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  --data-dir="/var/lib/etcd-from-backup" \
  --initial-cluster="ubuntu-s-4vcpu-8gb-blr1-01-master-1=https://127.0.0.1:2380" \
  --name="ubuntu-s-4vcpu-8gb-blr1-01-master-1" \
  --initial-advertise-peer-urls="https://127.0.0.1:2380" \
  --initial-cluster-token="etcd-cluster-1" \
  snapshot restore /tmp/etcd-backup.db
```

Make sure to change the of node name for the flag `--initial-cluster` and `--name` because this is going to change based on which leader node you are running the command on.
We want be changing the value of `--initial-cluster-token` because `etcdctl restore` command creates a new member and we want all these new members to have same token, so
that would belong to one cluster and accidently wouldnt join any other one.

To explore more about this we can look into the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/#backing-up-an-etcd-cluster).


## Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "backup-hnp95" --namespacetargets kanister
actionset "delete-backup-vqmdw-5n8nz" created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-vqmdw-5n8nz
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```
