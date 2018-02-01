.. _templates:

Template Parameters
====================

Template Parameters are use to render templates in Blueprints. A `TemplateParam`
struct is constructed based on fields in an ActionSet.

The TemplateParam struct is defined as:

.. code-block:: go

  // TemplateParams are use to render templates in Blueprints
  type TemplateParams struct {
      StatefulSet  StatefulSetParams
      Deployment   DeploymentParams
      ArtifactsIn  map[string]crv1alpha1.Artifact // A Kanister Artifact
      ArtifactsOut map[string]crv1alpha1.Artifact
      ConfigMaps   map[string]v1.ConfigMap
      Secrets      map[string]v1.Secret
      Time         string
  }

Rendering Templates
-------------------

Output Artifacts and templates in BlueprintPhases are rendered using `go
templating engine <https://golang.org/pkg/text/template/>`_. In addition to the
standard go template functions, Kanister imports all the `sprig
<http://masterminds.github.io/sprig/>`_ functions.

.. literalinclude:: ../pkg/param/render.go
  :linenos:
  :language: go
  :lines: 41-51

Protected Objects
-----------------
Kanister operates on the granularity of a `ProtectedObject`. As of the current
release, a Protected Object is a workload, specifically, a Deployment or 
StatefulSet. The TemplateParams struct has one field for each potential protected
object, which is effectively a union in go.

Each ProtectedObject param struct is a set of useful fields related to the
ProtectedObject. 

StatefulSet  
+++++++++++  

StatefulSetParams include the names of the Pods, Containers, and PVCs that
belong to the StatefulSet being acted on.

.. literalinclude:: ../pkg/param/param.go
  :linenos:
  :language: go
  :lines: 30-36

Deployment   
++++++++++   

DeploymentParams are identical to StatefulSetParams.

.. literalinclude:: ../pkg/param/param.go
  :linenos:
  :language: go
  :lines: 39-45

Artifacts
---------

Artifacts reference data that Kanister has externalized. Kanister can use them
as inputs or outputs to Actions. 

Artifacts are key-value pairs. In go this looks like:

.. literalinclude:: ../pkg/apis/cr/v1alpha1/types.go
  :linenos:
  :language: go
  :lines: 134-135

Go's templating engine allows us to easily access the values inside the
artifact. This functionality is documented `here
<https://golang.org/pkg/text/template/#hdr-Arguments>`_. 

.. note::

  When using this feature, we recommend using alphanumeric Artifact keys since
  the templating engine may not be able to use the `.` notation for non-standard
  characters.


Input Artifacts
+++++++++++++++

A Blueprint consumes parameters through template strings. If any template
parameters are absent at render time, the controller will log a rendering error
and fail that action.  In order to make a Blueprint's dependencies clear, some
types of template parameters are named explicitly as dependencies. If a
dependency is named in the Blueprint, then Kanister will validate that an
artifact  matching that name is present in the ActionSet.  Input Artifacts are
one such type of dependency.

Any input Artifacts required by a Blueprint are added to the
`inputArtifactNames` field in Blueprint actions. These named Artifacts
must be present in any ActionSetAction that uses that Blueprint.

For example:

+++++++++
Blueprint
+++++++++

.. literalinclude:: ../examples/time-log/blueprint.yaml
  :linenos:
  :language: yaml
  :lines: 1-5,28-42

+++++++++
ActionSet
+++++++++

.. literalinclude:: ../examples/time-log/restore-actionset.yaml
  :linenos:
  :language: yaml


Output Artifacts
++++++++++++++++

Output Artifacts are the only template parameter that themselves are rendered.
This allows users to customize them based on runtime configuration. Once an
output artifact is rendered, it is added to the status of the ActionSet.

A common reason for templating an output Artifact is to choose a location using
values from a ConfigMap.

Configuration
-------------

A Blueprint contains actions for a specific application - it should not need to
change unless the application itself changes. The ActionSet provides all the
necessary information to resolve the runtime configuration.

Time         
+++++++++++++

Time is provided as a template parameter. It is evaluated before any of the
phases begin execution and remains the unchanged between phases.

The time field is the current time in UTC, in the RFC3339Nano format. Using the
`sprig date <http://masterminds.github.io/sprig/date.html>`_ template functions,
you can parse this string convert it to your desired precision and format.

For example, if you only care about the "kitchen" time, use the following
template string: 

.. code-block:: go

  "{{ toDate "2006-01-02T15:04:05.999999999Z07:00" .Time  | date "3:04PM" }}"

ConfigMaps   
+++++++++++++

Like input Artifacts, ConfigMaps are named in Blueprints. Unlike input
Artifacts, ConfigMaps are not fully specified in the ActionSet. Rather, the
ActionSet contains a namespace/name reference to the ConfigMap. When creating
the template parameters, the controller will query the Kubernetes API server for
the ConfigMaps and adds them to the template params. 

The name given by the Blueprint is different than the Kubernetes API Object
name. An ActionSet action may map any ConfigMap to the name specified in the
Blueprint. This level of indirection allows configuration changes every time an
action is invoked.

Templating makes consuming the ConfigMaps easy. The example below illustrates a 
Blueprint that requires a ConfigMap named location.

First, we create a ConfigMap that contains configuration information about an S3
bucket:

.. code-block:: yaml

  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: backup-s3-location
    namespace: default
  data:
    bucket: s3://my.backup.bucket
    region: us-west-1

We can then reference this ConfigMap from the ActionSet as follows:

.. code-block:: yaml

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
  spec:
    actions:
    - name: backup
      blueprint: my-blueprint
      object:
        kind: deployment
        name: my-deployment
        namespace: default
      configMaps:
        location:
          name: backup-s3-location # The ConfigMap API object name
          namespace: default


Finally, we can access the ConfigMap's data inside the Blueprint using
templating:

.. code-block:: go

  "{{ .ConfigMaps.location.Data.bucket }}"
  "{{ .ConfigMaps.location.Data.region }}"

Secrets      
+++++++++++++

Secrets are handled the same way as ConfigMaps. They are named in a Blueprint.
This name is mapped to a reference in an ActionSet, and that reference is resolved
by the controller. This resolution consequently makes the Secret available to templates
in the Blueprint.

For example, let's say we have a secret which contains AWS credentials needed to access
an S3 bucket:

.. code-block:: yaml

  apiVersion: v1
  kind: Secret
  metadata:
    name: aws-creds
    namespace: default
  type: Opaque
  data:
    aws_access_key_id: MY_BASE64_ENCODED_AWS_ACCESS_KEY_ID
    aws_secret_access_key: MY_BASE64_ENCODED_AWS_SECRET_ACCESS_KEY

We create an ActionSet that has a reference to the Secret:

.. code-block:: yaml

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
  spec:
    actions:
    - name: backup
      blueprint: my-blueprint
      object:
        kind: deployment
        name: my-deployment
        namespace: default
      secrets:
        aws:
          name: aws-creds # The Secret API object name
          namespace: default

We can access the Secret's data inside the Blueprint using templating. Since
secrets `Data` field has the type `[]byte`, we use sprig's `toString function
<http://masterminds.github.io/sprig/conversion.html>`_ to cast the values to
usable strings.

.. code-block:: yaml

  # We've named this secret `aws` in the Blueprint:
  secretNames:
    - aws

  ...  

  # We access the secret values via templating:
  "{{ .Secrets.aws.Data.aws_access_key_id | toString }}"
  "{{ .Secrets.aws.Data.aws_secret_access_key | toString }}"
