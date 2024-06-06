This document is going to show you how you can backup the ETCD of your OpenShift cluster. The
commands are run into an [OCP](https://www.openshift.com/products/container-platform) but it should work on any other OpenShift cluster.


## Prerequisites Details

* OpenShift (OCP) cluster
* PV support on the underlying infrastructure
* Kanister version 0.32.0 with `profiles.cr.kanister.io` CRD, [`kanctl`](https://docs.kanister.io/tooling.html#install-the-tools) Kanister tool installed


# Integrating with Kanister

When we say integrating with Kanister we mean creating some CRs, for example Blueprint and ActionSet
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

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Before actually creating the Blueprint, we will have to create a secret in a new or an existing namespace. This
secret is going to have the details about the ETCD members that are running on your cluster

- **endpoints** : ETCD server client listen URL, https://[127.0.0.1]:2379
- **labels** : These labels will be used to identify the ETCD pods that are running, for ex `app=etcd,etcd=true`
- **etcdns** : Namespace where the etcd pods are running

Below command can be used to create the secret, assuming the ETCD pods are running in the `openshift-etcd` namespace

```
# Create a new namespace
» oc create ns etcd-backup
namespace/etcd-backup created

» oc create secret generic etcd-openshift-etcd \
    --from-literal=endpoints=https://10.0.133.5:2379 \
    --from-literal=labels=app=etcd,etcd=true \
    --from-literal=etcdns=openshift-etcd \
    --namespace etcd-backup
secret/etcd-openshift-etcd created
```

Once the secret is created below command can be used to create the Blueprint

```
» oc apply -f etcd-incluster-ocp-blueprint.yaml -n kanister
blueprint.cr.kanister.io/etcd-blueprint configured
```

## Protect the Application

Before actually taking the backup of ETCD let's first create a dummy namespace and some resources in that namespace, and after taking the ETCD backup we will delete
this namespace, so that we can check if this namespace has actually been restored after restoring the ETCD.

```
root@workmachine:/repo# oc create ns nginx
namespace/nginx created
root@workmachine:/repo# oc create deployment -n nginx nginx --image nginx
deployment.apps/nginx created
root@workmachine:/repo# oc get all -n nginx
NAME                        READY     STATUS             RESTARTS   AGE
pod/nginx-f89759699-k6f5n   0/1       CrashLoopBackOff   2          45s

NAME                    READY     UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nginx   0/1       1            0           46s

NAME                              DESIRED   CURRENT   READY     AGE
replicaset.apps/nginx-f89759699   1         1         0         47s

```

We can now take snapshot of the ETCD server that is running by creating backup ActionSet that is going to execute backup phase from the Blueprint that we have
created above

**Note**

Please make sure to change the **profile-name**, **blueprint name**, **secret-name** and **secret-namespace** in the `backup-actionset.yaml` manifest file. Where `secret-name` is the name of secret that has all the details and we created earlier.

```
# find the profile name
» oc get profiles.cr.kanister.io -n kanister
NAME               AGE
s3-profile-2lhk8   52s

# find the Blueprint name
» oc get blueprint -n kanister
NAME             AGE
etcd-blueprint   85m

# create actionset
» oc create -f backup-actionset.yaml --namespace kanister
actionset.cr.kanister.io/backup-4f6jn created

# you can check the status of the actionset to make sure it has been completed
» oc describe actionset -n kanister backup-4f6jn
Name:         backup-4f6jn
Namespace:    kanister
Labels:       <none>
...
...
Events:
  Type    Reason           Age   From                 Message
  ----    ------           ----  ----                 -------
  Normal  Started Action   3m    Kanister Controller  Executing action backup
  Normal  Started Phase    3m    Kanister Controller  Executing phase takeSnapshot
  Normal  Ended Phase      3m    Kanister Controller  Completed phase takeSnapshot
  Normal  Started Phase    3m    Kanister Controller  Executing phase uploadSnapshot
  Normal  Ended Phase      2m    Kanister Controller  Completed phase uploadSnapshot
  Normal  Started Phase    2m    Kanister Controller  Executing phase removeSnapshot
  Normal  Ended Phase      2m    Kanister Controller  Completed phase removeSnapshot
  Normal  Update Complete  2m    Kanister Controller  Updated ActionSet 'backup-4f6jn' Status->complete
```

## Imitate Disaster

After the backup has successfully been taken, let's go ahead and delete the dummy namespace that we created to imitate the disaster

```
root@workmachine:/repo# oc delete ns nginx
namespace "nginx" deleted

root@workmachine:/repo# oc get all -n nginx
No resources found.
```


## Restore ETCD cluster

To restore the ETCD cluster we can follow the [documentation](https://docs.openshift.com/container-platform/4.5/backup_and_restore/disaster_recovery/scenario-2-restoring-cluster-state.html) that is provided by the OpenShift team.
The restore script (`cluster-restore.sh`) mentioned above requires modification as it expects the static pods manifests which are not backed up in our case. The modified restore script can be found in this repo.

You can follow the steps that are mentioned below along with the documentation that is mentioned above, most of the steps that are mentioned here are either directly taken from the documentation or are modified version of it. Among all the running leader nodes choose one node to be the restore node, make sure you have SSH connectivity to all of the leader nodes including the one that you have chosen to be restore node.

You will have to have a command line utility that can be used to download the ETCD snapshot that we have taken in the eariler step, that will depend on the object
storage that you used. For example if you used the object storage to be AWS S3, you will need `aws` cli to download the ETCD snapshot. Once you have the CLI installed
on the restore host, below steps can be followed to restore ETCD:

- Download the ETCD snapshot on the restore host using the aws cli on a specific path let's say `/var/home/core/etcd-backup`
- Stop the static pods from all other leader hosts (not the recovery host) by copying the manifests out of the static pod path dir, i.e. `/etc/kubernetes/manifests`

  ```
  # move etcd pod manifest
  sudo mv /etc/kubernetes/manifests/etcd-pod.yaml /tmp

  # make sure etcd pod has been stopped
  sudo crictl ps | grep etcd

  # move api server pod
  sudo mv /etc/kubernetes/manifests/kube-apiserver-pod.yaml /tmp
  ```

- Move the etcd data dir to a different location
  ```
  sudo mv /var/lib/etcd/ /tmp
  ```

  Repeat these steps on all other leader hosts that are not the restore host

- Run the `cluster-ocp-restore.sh` script with the location where you have downloaded the etcd snapshot that in our case is `/var/home/core/etcd-backup`

  ```
  sudo ./cluster-ocp-restore.sh /var/home/core/etcd-backup
  ```

- Restart `kubelet` service on all the leader nodes

  ```
  sudo systemctl restart kubelet.service
  ```

- Verify single ETCD node has been started, run below from recovery host to check if ETCD container is up

  ```
  sudo crictl ps | grep etcd

  # you can also verify that the ETCD pod is running now.
  root@workmachine:/repo# oc get pods -n openshift-etcd
  NAME                                                           READY     STATUS      RESTARTS   AGE
  etcd-ip-10-0-149-197.us-west-1.compute.internal                1/1       Running     0          3m57s
  installer-2-ip-10-0-149-197.us-west-1.compute.internal         0/1       Completed   0          7h54m
  installer-2-ip-10-0-166-99.us-west-1.compute.internal          0/1       Completed   0          7h53m
  installer-2-ip-10-0-212-253.us-west-1.compute.internal         0/1       Completed   0          7h52m
  revision-pruner-2-ip-10-0-149-197.us-west-1.compute.internal   0/1       Completed   0          7h51m
  revision-pruner-2-ip-10-0-166-99.us-west-1.compute.internal    0/1       Completed   0          7h51m
  revision-pruner-2-ip-10-0-212-253.us-west-1.compute.internal   0/1       Completed   0          7h51m

  ```

- Force ETCD deployment, you can run below command from the terminal you have cluster access
  ```
  oc patch etcd cluster -p='{"spec": {"forceRedeploymentReason": "recovery-'"$( date --rfc-3339=ns )"'"}}' --type=merge

  # Verify all nodes are updated to latest version
  oc get etcd -o=jsonpath='{range .items[0].status.conditions[?(@.type=="NodeInstallerProgressing")]}{.reason}{"\n"}{.message}{"\n"}'
  ```

  And you will get message like

  ```
  3 nodes are at revision 2; 0 nodes have achieved new revision 3
  ```
  Please wait for some time make sure the component has been updated to the latest version, and then the message would look somewhat like this

  ```
  oc get etcd -o=jsonpath='{range .items[0].status.conditions[?(@.type=="NodeInstallerProgressing")]}{.reason}{"\n"}{.message}{"\n"}'
  AllNodesAtLatestRevision
  3 nodes are at revision 3

  ```

  that depicts that all the three nodes have been updated to the latest version.

- Force rollout for the control plance components

  ```
  # API Server
  oc patch kubeapiserver cluster -p='{"spec": {"forceRedeploymentReason": "recovery-'"$( date --rfc-3339=ns )"'"}}' --type=merge
  # Wait for version update
  oc get kubeapiserver -o=jsonpath='{range .items[0].status.conditions[?(@.type=="NodeInstallerProgressing")]}{.reason}{"\n"}{.message}{"\n"}'
  # again you will have to wait until you get message like
  # 3 nodes are at revision 6

  # kubecontrollermanager
  oc patch kubecontrollermanager cluster -p='{"spec": {"forceRedeploymentReason": "recovery-'"$( date --rfc-3339=ns )"'"}}' --type=merge
  # wait for the revision
  oc get kubecontrollermanager -o=jsonpath='{range .items[0].status.conditions[?(@.type=="NodeInstallerProgressing")]}{.reason}{"\n"}{.message}{"\n"}'
  # 3 nodes are at revision 9

  #kubescheduler
  oc patch kubescheduler cluster -p='{"spec": {"forceRedeploymentReason": "recovery-'"$( date --rfc-3339=ns )"'"}}' --type=merge
  # wait for reviosn to update
  oc get kubescheduler -o=jsonpath='{range .items[0].status.conditions[?(@.type=="NodeInstallerProgressing")]}{.reason}{"\n"}{.message}{"\n"}'
  # 3 nodes are at revision 7; 0 nodes have achieved new revision 8
  # 3 nodes are at revision 8

  ```

- Verify that all etcd pods are running fine

  ```
  root@workmachine:/repo# oc get pods -n openshift-etcd | grep etcd
  etcd-ip-10-0-149-197.us-west-1.compute.internal                4/4       Running     0          19m
  etcd-ip-10-0-166-99.us-west-1.compute.internal                 4/4       Running     0          20m
  etcd-ip-10-0-212-253.us-west-1.compute.internal                4/4       Running     0          20m
  ```

- Now that, we can see all the ETCD pods have been restored we can make sure the dummy namespace that we created and then deleted, has been restored or not

  ```
  root@workmachine:/repo# oc get all -n nginx
  NAME                        READY     STATUS             RESTARTS   AGE
  pod/nginx-f89759699-k6f5n   0/1       CrashLoopBackOff   9          46m

  NAME                    READY     UP-TO-DATE   AVAILABLE   AGE
  deployment.apps/nginx   0/1       1            0           46m

  NAME                              DESIRED   CURRENT   READY     AGE
  replicaset.apps/nginx-f89759699   1         1         0         46m

  ```

  and as you can see we have successfully restored the namespace that we deleted.
