.. _templates:

Template Parameters
*******************

Template Parameters are used to render templates in Blueprints. A ``TemplateParam``
struct is constructed based on fields in an ActionSet.

The TemplateParam struct is defined as:

.. code-block:: go

  // TemplateParams are use to render templates in Blueprints
  type TemplateParams struct {
    StatefulSet      *StatefulSetParams
    DeploymentConfig *DeploymentConfigParams
    Deployment       *DeploymentParams
    PVC              *PVCParams
    Namespace        *NamespaceParams
    ArtifactsIn      map[string]crv1alpha1.Artifact
    ConfigMaps       map[string]v1.ConfigMap
    Secrets          map[string]v1.Secret
    Time             string
    Profile          *Profile
    Options          map[string]string
    Object           map[string]interface{}
    Phases           map[string]*Phase
    DeferPhase       *Phase
    PodOverride      crv1alpha1.JSONMap
  }

Rendering Templates
===================

Output Artifacts and templates in BlueprintPhases are rendered using `go
templating engine <https://golang.org/pkg/text/template/>`_. In addition to the
standard go template functions, Kanister imports all the `sprig
<http://masterminds.github.io/sprig/>`_ functions.

.. code-block:: go
  :linenos:

  case reflect.Map:
    ras := make(map[interface{}]interface{}, val.Len())
    for _, k := range val.MapKeys() {
      rk, err := render(k.Interface(), tp)
      if err != nil {
        return nil, err
      }
      rv, err := render(val.MapIndex(k).Interface(), tp)
      if err != nil {
        return nil, err
      }
      ras[rk] = rv
    }
    return ras, nil

.. note:: Kanister will error during template rendering if FIPS non-compliant
  sprig functions are used. The unsupported functions include: ``bcrypt``,
  ``derivePassword``, ``htpasswd``, and ``genPrivateKey`` for key type ``dsa``.


Objects
=======

Kanister operates on the granularity of an ``Object``. As of the current
release, well known Object types are ``Deployment``, ``StatefulSet``,
``PersistentVolumeClaim``, ``Namespace`` or OpenShift's ``DeploymentConfig``.
The TemplateParams struct has one field for each well known object type,
which is effectively a union in go.

Other than the types mentioned above, Kanister can also act on any Kubernetes
object such as a CRD and the :ref:`object` field in TemplateParams is populated with the
unstructured content of those.

Each param struct described below is a set of useful fields related to the
Object.

StatefulSet
-----------

StatefulSetParams include the names of the Pods, Containers, and PVCs that
belong to the StatefulSet being acted on.

.. code-block:: go
  :linenos:

  // StatefulSetParams are params for stateful sets.
  type StatefulSetParams struct {
    Name                   string
    Namespace              string
    Pods                   []string
    Containers             [][]string
    PersistentVolumeClaims [][]string
  }

For example, to access the first pod of a StatefulSet use:

.. code-block:: go

  "{{ index .StatefulSet.Pods 0 }}"

Deployment
----------

DeploymentParams are identical to StatefulSetParams.

.. code-block:: go
  :linenos:

  // DeploymentParams are params for deployments
  type DeploymentParams struct {
    Name                   string
    Namespace              string
    Pods                   []string
    Containers             [][]string
    PersistentVolumeClaims [][]string
  }

For example, to access the Name of a Deployment use:

.. code-block:: go

  "{{ index .Deployment.Name }}"

DeploymentConfig
----------------

DeploymentConfig resources are specific to OpenShift clusters and are
almost like Deployment resource but have some significant differences.
Details about DeploymentConfig can be read
`on this document <https://docs.openshift.com/container-platform/4.1/applications/deployments/what-deployments-are.html>`_.
DeploymentConfigParams similar to DeploymentParams.

.. code-block:: go
  :linenos:

  // DeploymentConfigParams are params for DeploymentConfig
  type DeploymentConfigParams struct {
    Name                   string
    Namespace              string
    Pods                   []string
    Containers             [][]string
    PersistentVolumeClaims map[string]map[string]string
  }

For example, to access the Name of a Deployment use:

.. code-block:: go

  "{{ index .DeploymentConfig.Name }}"


Namespace
---------

NamespaceParams includes the name of the namespace
that is being acted on when the ActionSet ``Object`` is
specifies a Namespace

.. code-block:: go
  :linenos:

  // NamespaceParams are params for a Namespace
  type NamespaceParams struct {
    Name              string
  }

For example, to access the Name of a Namespace, use:

.. code-block:: go

  "{{ .Namespace.Name }}"

PVC
---

PVCParams includes the name and namespace of the persistent volume claim
that is being acted on.

.. code-block:: go
  :linenos:

  // PVCParams are params for a PVC
  type PVCParams struct {
    Name                   string
    Namespace              string
  }

For example, to access the Name of a persistent volume claim, use:

.. code-block:: go

  "{{ .PVC.Name }}"

.. _object:

Object
------

Object includes the unstructured representation of the underlying
Kubernetes object. This allows the flexibility of writing blueprints
that operate on objects that are not well known to Kanister such as
CRD's

.. code-block:: go
  :linenos:

  type TemplateParams struct {
    ...
    Object       map[string]interface{}
    ...
  }

For example, to access the Name in the Kubernetes ObjectMeta of an
arbitrary object, use:

.. code-block:: go

  "{{ .Object.metadata.name }}"

Artifacts
=========

Artifacts reference data that Kanister has externalized. Kanister can use them
as inputs or outputs to Actions.

Artifacts are key-value pairs. In go this looks like:

.. code-block:: go
  :linenos:

  // Artifact tracks objects produced by an action.
  type Artifact struct {
    KeyValue    map[string]string   `json:"keyValue"`
  }

The specific schema that Artifacts use is up to the Blueprint author.

Go's templating engine allows us to easily access the values inside the
artifact. This functionality is documented `here
<https://golang.org/pkg/text/template/#hdr-Arguments>`_.

.. note::

  When using this feature, we recommend using alphanumeric Artifact keys since
  the templating engine may not be able to use the ``.`` notation for non-standard
  characters.


Input Artifacts
---------------

A Blueprint consumes parameters through template strings. If any template
parameters are absent at render time, the controller will log a rendering error
and fail that action.  In order to make a Blueprint's dependencies clear, some
types of template parameters are named explicitly as dependencies. If a
dependency is named in the Blueprint, then Kanister will validate that an
artifact  matching that name is present in the ActionSet. Input Artifacts are
one such type of dependency.

Any Input Artifacts required by a Blueprint are added to the
``inputArtifactNames`` field in Blueprint actions. These named Artifacts
must be present in any ActionSetAction that uses that Blueprint. Always
create ActionSet in the same namespace as the controller.

For example, with the following snippet from the time-log example Blueprint:

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: Blueprint
  metadata:
    name: time-log-bp
    namespace: kanister
  actions:
    backup:
      configMapNames:
      - location
      secretNames:
      - aws
      outputArtifacts:
        timeLog:
          keyValue:
            path: 's3://{{ .ConfigMaps.location.Data.path }}/time-log/{{ toDate "2006-01-02T15:04:05.999999999Z07:00" .Time  | date "2006-01-02" }}'

      ...
    restore:
      inputArtifactNames:
        - exampleArtifact
      ...

The ActionSet for restore will need to look like:

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: time-log-restore-
    namespace: kanister
  spec:
    actions:
    - name: restore
      blueprint: time-log-bp
      object:
        kind: Deployment
        name: time-logger
        namespace: default
      secrets:
        aws:
          name: aws-creds
          namespace: kanister
      artifacts:
        timeLog:
          keyValue:
            path: s3://time-log-test-bucket/tutorial/time-log/time.log


Output Artifacts
----------------

Output Artifacts are the only template parameter that themselves are rendered.
This allows users to customize them based on runtime configuration. Once an
output artifact is rendered, it is added to the status of the ActionSet.

A common reason for templating an output Artifact is to choose a location using
values from a ConfigMap.

Configuration
=============

A Blueprint contains actions for a specific application - it should not need to
change unless the application itself changes. The ActionSet provides all the
necessary information to resolve the runtime configuration.

Time
----

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
----------

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

First, in the kanister controller's namespace, we create a ConfigMap that
contains configuration information about an S3 bucket:

.. code-block:: yaml
  :linenos:

  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: backup-s3-location
    namespace: kanister
  data:
    bucket: s3://my.backup.bucket
    region: us-west-1

We can then reference this ConfigMap from the ActionSet as follows:

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
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
          namespace: kanister


Finally, we can access the ConfigMap's data inside the Blueprint using
templating:

.. code-block:: go

  "{{ .ConfigMaps.location.Data.bucket }}"
  "{{ .ConfigMaps.location.Data.region }}"

Secrets
-------

Secrets are handled the same way as ConfigMaps. They are named in a Blueprint.
This name is mapped to a reference in an ActionSet, and that reference is resolved
by the controller. This resolution consequently makes the Secret available to templates
in the Blueprint.

For example, consider the following secret which contains AWS credentials
needed to access an S3 bucket:

.. code-block:: yaml
  :linenos:

  apiVersion: v1
  kind: Secret
  metadata:
    name: aws-creds
    namespace: kanister
  type: Opaque
  data:
    aws_access_key_id: MY_BASE64_ENCODED_AWS_ACCESS_KEY_ID
    aws_secret_access_key: MY_BASE64_ENCODED_AWS_SECRET_ACCESS_KEY

When creating an ActionSet include a reference to the Secret:

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
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
          namespace: kanister

The data of the Secret is then available inside the Blueprint using
templating. Since secrets ``Data`` field has the type ``[]byte``, use
sprig's
`toString function <http://masterminds.github.io/sprig/conversion.html>`_
to cast the values to usable strings.

.. code-block:: yaml

  # This secret is named `aws` in the Blueprint:
  secretNames:
    - aws

  ...

  # Access the secret values via templating:
  "{{ .Secrets.aws.Data.aws_access_key_id | toString }}"
  "{{ .Secrets.aws.Data.aws_secret_access_key | toString }}"

Profiles
--------

Profiles are a Kanister CustomResource and capture information about a location
for data operation artifacts and corresponding credentials that will be made
available to a Blueprint.

Unlike Secrets and ConfigMaps, only a single profile can optionally be
referenced by an ActionSet. As a result, there it is not necessary to
name the Profiles in the Blueprint.

The following examples should be helpful.

.. code-block:: yaml

  # Access the Profile s3 location bucket
  "{{ .Profile.Location.Bucket }}"

  # Access the associated secret credential
  # Assuming "{{ .Profile.Credential.KeyPair.SecretField }}" is 'Secret'
  "{{ .Profile.Credential.KeyPair.Secret }}"

The currently supported Profile template is based on the following definitions

.. code-block:: go
  :linenos:

  type Profile struct {
    Location          Location
    Credential        Credential
    SkipSSLVerify     bool
  }

  type LocationType string

  const (
    LocationTypeGCS         LocationType = "gcs"
    LocationTypeS3Compliant LocationType = "s3Compliant"
    LocationTypeAzure       LocationType = "azure"
  )


  type Location struct {
    Type      LocationType
    Bucket    string
    Endpoint  string
    Prefix    string
    Region    string
  }

  type CredentialType string

  const (
    CredentialTypeKeyPair CredentialType = "keyPair"
  )

  // Only supporting KeyPair credentials currently
  type Credential struct {
    Type    CredentialType
    KeyPair *KeyPair
  }

  type KeyPair struct {
    IDField     string
    SecretField string
    Secret      ObjectReference
  }

Options
-------

Options map can be used to render any additional parameters in Blueprints.

For example, if you want to use a specific Pod to carry out actions in a Blueprint,
the Pod name can be specified using the Options as follows:

.. code-block:: yaml
  :linenos:

  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
  spec:
    actions:
    - name: backup
      blueprint: my-blueprint
      object:
        kind: deployment
        name: my-deployment
        namespace: default
      options:
        podName: some-pod

The Options can then be used in the Blueprint via templating:

.. code-block:: go

  "{{ .Options.podName }}"

Phases
------

Phases are used to capture information required or returned from Blueprint phases.
Currently, each phase contains a map of Secrets required to execute a phase,
or the Output map returned from the execution.

The definition is as follows:

.. code-block:: go
  :linenos:

  type Phase struct {
    Secrets map[string]v1.Secret
    Output  map[string]interface{}
  }

The phase parameters can be referenced by the phases following it,
or as output artifacts using templating.

For example, an output artifact can reference the output from a phase as follows:

.. code-block:: go

  "{{ .Phases.phase-name.Output.key-name }}"

Similarly, a phase can use Secrets as arguments:

.. code-block:: go

  "{{ .Phases.phase-name.Secrets.secret-name.Namespace }}"

DeferPhase
----------

``DeferPhase`` is used to capture information returned from the Blueprint's ``DeferPhase``
execution. The information is stored in the ``Phase`` struct that has the below
definition:

.. code-block:: go
  :linenos:

  type Phase struct {
    Secrets map[string]v1.Secret
    Output  map[string]interface{}
  }

Output artifact can be set as follows:

.. code-block:: go

  "{{ .DeferPhase.Output.key-name }}"


Output artifacts that are set using ``DeferPhase`` can be consumed by other actions'
phases using the same way other output artifacts are consumed.
