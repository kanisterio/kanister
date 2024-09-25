.. _rbac:

RBAC Configuration
******************

Earlier, the Kanister operator was bound to the `edit` `ClusterRole`
using a `ClusterRoleBinding` provided by the Helm Chart.

The `edit` `ClusterRole` is a built-in Kubernetes system role that offers
permissions to modify most objects within a namespace, excluding roles,
role bindings, and resource quotas. This role allowed access to create,
update, delete, and view resources such as Deployments, Pods, Services,
ConfigMaps, PersistentVolumeClaims, and more.

To enhance security, the `edit` `ClusterRoleBinding` has been removed from
the Kanister Helm Chart. Users are required to create their own
`Role`/`RoleBinding` in the application's namespace to grant the necessary
permissions to Kanister's Service Account, providing more control over
the specific permissions granted.

Creating a RoleBinding with edit ClusterRole
============================================

To allow Kanister to perform backup/restore operations in the application
namespace, create a `RoleBinding` in the application namespace that assigns
the `edit` `ClusterRole` to Kanister's Service Account:

.. code-block:: bash

  kubectl create rolebinding kanister-edit-binding --clusterrole=edit \
  --serviceaccount=<release-namespace>:<release-name>-kanister-operator \
  --namespace=<application-namespace>

Creating a Role with Granular Permissions
=========================================

If Blueprint doesn't require access to all the resources that are included
in the `edit` ClusterRole, you can create a `Role` in application namespace
with just the specific resources and verbs that Blueprint needs, and a `RoleBinding`
in application namespace to bind the `Role` to Kanister's Service Account.
This approach enhances security by granting only the necessary permissions.

1. Create a `Role` with the permissions required by the Blueprint:

.. code-block:: yaml

  apiVersion: rbac.authorization.k8s.io/v1
  kind: Role
  metadata:
    name: kanister-role
    namespace: <application-namespace>
  rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "persistentvolumeclaims" ,"secrets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets"]
    verbs: ["get", "list", "watch"]

2. Create a `RoleBinding` to bind the `Role` to Kanister's Service Account:

.. code-block:: bash

  kubectl create rolebinding kanister-role-binding --role=kanister-role \
  --serviceaccount=<release-namespace>:<release-name>-kanister-operator \
  --namespace=<application-namespace>

After setting up the required `Role`/`RoleBinding`, Kanister will be able
to successfully perform snapshot and restore operations in the application's
namespace.
