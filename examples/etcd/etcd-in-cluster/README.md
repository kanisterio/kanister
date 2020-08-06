This document is going to show you how you can backup the ETCD cluster that is running as part of your Kubernetes control plane. The
commands are run into a cluster that is setup using [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) but it should work on any other single or multi node ETCD cluster.

## Prerequisites Details

* Kubernetes 1.9+ with Beta APIs enabled, and you are not on managed Kubernetes
* PV support on the underlying infrastructure
* Kanister version 0.32.0 with `profiles.cr.kanister.io` CRD installed

# Integrating with Kanister

Once we have made sure that the prequisites are met, when we say integrating with Kanister we mean creating some CRs, for example Blurprint and Actionset
that would help perform the actions on the the ETCD instance that we are running.

## Create Profile resource

```bash
» kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <bucket-name> --region ap-south-1 \
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
of the cluster that is setup through Kind ([kubeadm](https://github.com/kubernetes/kubeadm)), I am assuming TLS based authentication is being used.

To specify the location of the CA, certificate and key we will have to create a secret in the same namespace where your ETCD pod is running. This
secret is going to have the name of the format `etcd-<etcd-pod-namespace>` with these fields

- **cacert** : CA (certificate authority) cert, would usually be `/etc/kubernetes/pki/etcd/ca.crt` on Kind clusters

- **cert** : Certificate that is used to secure the ETCD cluster, would usually be `/etc/kubernetes/pki/etcd/server.crt` on Kind clusters

- **endpoints** : ETCD server client listen URL, https://[127.0.0.1]:2379

- **key** : TLS key file, would be `/etc/kubernetes/pki/etcd/server.key` in case of Kind cluster


```
» kubectl create secret generic etcd-kube-system --from-literal=cacert=/etc/kubernetes/pki/etcd/ca.crt --from-literal=cert=/etc/kubernetes/pki/etcd/server.crt --from-literal=endpoints=https://\[127.0.0.1\]:2379 --from-literal=key=/etc/kubernetes/pki/etcd/server.key -n kube-system
secret/etcd-kube-system created
```

**Note**
Please make sure that you have correct path of these certificate files. If any of the path is incorrect the etcd snapshot will fail.

Once secret is created, let's go ahead and create Blueprint in the same namespace as the Kanister controller

```
» kubectl create -f etcd-incluster-blueprint.yaml -n kanister
blueprint.cr.kanister.io/etcd-blueprint created
```

## Protect the Application

We can now take snapshot of the ETCD server that is running by creating backup actionset that is going to execute backup phase from the blueprint that we have
created above

**Note**
Pleae make sure to change the **profile name**, the **ETCD pod name** and **blueprint name** in the `backup-actionset.yaml` manifest file.

```
# find the profile name
» kubectl get profile -n kanister
NAME               AGE
s3-profile-nnvmm   54s

# fine blueprint name
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
