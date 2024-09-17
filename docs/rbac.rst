.. _rbac:

RBAC Configuration
******************

To enhance security, the `edit` `ClusterRoleBinding` has been removed from
the Kanister Helm Chart. Users now need to create their own `Role` / `RoleBinding`
in the application's namespace to grant the necessary permissions to
Kanister's Service Account.

Creating a RoleBinding with edit ClusterRole
============================================

To allow Kanister to perform backup/restore operations in a specific
namespace, create a `RoleBinding` that assigns the `edit` `ClusterRole`
to Kanister's Service Account:

.. code-block:: bash

  kubectl create rolebinding kanister-edit-binding --clusterrole=edit \
  --serviceaccount=<release-namespace>:kanister-kanister-operator \
  --namespace=<application-namespace>

Creating a Role with Granular Permissions
=========================================

If you prefer to fine-tune the access, you can create a `Role` with
specific permissions and bind it to Kanister's Service Account:

1. Create a `Role` with the required permissions:

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
  --serviceaccount=<release-namespace>:kanister-kanister-operator \
  --namespace=<application-namespace>
