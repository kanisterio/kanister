.. _operator:

The Operator Pattern
====================

.. contents:: The Kanister Operator
  :local:

Kanister follows the operator pattern. This means Kanister defines its own
resources and interacts with those resources through a controller. `This
blog post <https://coreos.com/blog/introducing-operators.html>`_ from CoreOS
describes the pattern in detail.


Custom Resources
----------------
Users interact with Kanister through Kubernetes resources known as
CustomResources (CRs). When the controller starts, it creates the CR
definitions called CustomResourceDefinitions (CRDs).  `CRDs
<https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/>`_
were introduced in Kubernetes 1.7 and replaced TPRs. The lifecycle of these
objects can be managed entirely through kubectl. Kanister uses Kubernetes' code
generation tools to create go client libraries for its CRs.

The schemas of the Kanisters CRDs can be found in `types.go
<https://github.com/kanisterio/kanister/tree/master/pkg/apis/cr/v1alpha1/types.go>`_

Blueprints
++++++++++

Blueprint CRs are a set of instructions that tell the controller how to perform 
actions on a specific application.

A Blueprint contains a field called `Actions` which is a mapping of Action Name
to `BlueprintAction`.

The definition of a `BlueprintAction` is:

.. code-block:: go
  :linenos:

  // BlueprintAction describes the set of phases that constitute an action.
  type BlueprintAction struct {
      Name               string              `json:"name"`
      Kind               string              `json:"kind"`
      ConfigMapNames     []string            `json:"configMapNames"`
      SecretNames        []string            `json:"secretNames"`
      InputArtifactNames []string            `json:"inputArtifactNames"`
      OutputArtifacts    map[string]Artifact `json:"outputArtifacts"`
      Phases             []BlueprintPhase    `json:"phases"`
  }

.. todo::

  Name is redundant in BluerintAction since we already specify it in the Blueprint Actions map

- `Kind` is the type of object we'll act on. Currently we support `Deployment` or
  `Statefulset`
- `ConfigMapNames`, `SecretNames`, `InputArtifactNames` are lists of named
  parameters that must be included by the ActionSet.
- `Phases` are a list of `BlueprintPhases`. These phases are invoked in order
  when executing this Action. 

.. code-block:: go
  :linenos:

  // BlueprintPhase is a an individual unit of execution.
  type BlueprintPhase struct {
      Func string   `json:"func"`
      Name string   `json:"name"`
      Args []string `json:"args"`
  }

- `Func` is the name of the registered Kanister function. By default, the
  controller includes two Kanister functions `"KubeExec"` and `'KubeTask"`
- `Name` is mostly cosmetic. It is useful in quickly identifying which
  phases the controller has finished executing.
- `Args` are a list of argument templates that the controller will render using the
  template parameters. Each argument is rendered individually.


ActionSets
++++++++++

Creating an ActionSet instructs the controller to run an action now.
The user specifies the runtime parameters inside the spec of the ActionSet.
Based on the parameters, the Controller populates the Status of the object,
executes the actions, and updates the ActionSet's status.

An ActionSetSpec contains a list of ActionSpecs. An ActionSpec is defined
as follows:

.. code-block:: go
 :linenos:

  // ActionSpec is the specification for a single Action.
  type ActionSpec struct {
      Name string                           `json:"name"`
      Object ObjectReference                `json:"object"`
      Blueprint string                      `json:"blueprint,omitempty"`
      Artifacts map[string]Artifact         `json:"artifacts,omitempty"`
      ConfigMaps map[string]ObjectReference `json:"configMaps"`
      Secrets map[string]ObjectReference    `json:"secrets"`
  }

- `Name` chooses the action in the Blueprint.
- `Object` is the Kubernetes reference to the object we're performing the action
  on.
- `Blueprint` is the name of the Blueprint that contains the action we're going
  to run
- `Artifacts` are input Artifacts that we pass into the Blueprint. This must
  contain an Artifact for each name listed in the BlueprintAction's InputArtifacts.
- `ConfigMaps` and `Secrets` are a mappings of names specified in the Blueprint
  to Kubernetes references.

An ActionSetStatus mirrors the Spec, but contains the phases of execution, their
state, and the overall execution progress.

.. code-block:: go

  // ActionStatus is updated as we execute phases.
  type ActionStatus struct {
      Name string                   `json:"name"`
      Object ObjectReference        `json:"object"`
      Blueprint string              `json:"blueprint"`
      Phases []Phase                `json:"phases"`
      Artifacts map[string]Artifact `json:"artifacts"`
  }

Unlike in the ActionSpec, the Artifacts in the ActionStatus are the rendered
output artifacts from the Blueprint. These are populated as soon as they are
rendered, but should only be considered valid once the action is complete.


Each phase in the ActionStatus phases list contains the phase name of the
Blueprint phase and its state of execution.

.. code-block:: go

  // Phase is subcomponent of an action.
  type Phase struct {
      Name  string `json:"name"`
      State State  `json:"state"`
  }


Controller
----------

The Kanister controller is a Kubernetes Deployment and is installed easily using
`kubectl`. See :ref:`install` for more information on deploying the controller.

Exectution Walkthrough
++++++++++++++++++++++

The controller watches for new/updated ActionSets in the same namespace in which
it is deployed. When it sees an ActionSet without a nil status field, it 
immediately initializes the ActionSet's status to the Pending State. The status is
also prepopulated with the pending phases.

Execution begins by resolving all the :ref:`templates`. If any required
object references or artifacts are missing from the ActionSet, the ActionSet
status is marked as failed. Otherwise, the template params are used to render the 
output Artifacts, and then the args in the Blueprint.

For each action, all phases are executed in-order. The rendered args are
passed to :ref:`templates` which correspond to a single phase. When a phase
completes, the status of the phase is updated. If any single phase fails, the
entire ActionSet is marked as failed.  Upon failure, the controller ceases
execution of the ActionSet.

Within an ActionSet, individual Actions are run in parallel.

Currently the user is responsible for cleaning up ActionSets once they complete.

Kanctl
----------

Although all Kanister actions can be run using kubectl, there are situations
where this may be cumbersome. Many actions depend on the Artifacts created by
another action. The canonical example is backup/restore. Manually creating a
restore ACtionSet requires copying Artifacts from the status of the complete
backup ActionSet, which is an error prone process. 

`kanctl` helps make running dependant ActionSets more robust.  Kanctl is a
command-line tool that makes it easier to create ActionSets.

To demonstrate backup/restore ActionSet chaining, we'll perform "`kanctl perform
from`".

.. code-block:: bash

  $ kanctl  perform
  Create and ActionSet to perform an action

  Usage:
    kanctl perform [command]

  Available Commands:
    from        Perform an action on the artifacts from <parent>

  Flags:
    -h, --help   help for perform

  Global Flags:
    -n, --namespace string   Override namespace obtained from kubectl context

.. code-block:: bash

  # perform backup
  $ kubectl create -f examples/time-log/backup-actionset.yaml
  actionset "s3backup-j4z6f" created

  # restore from the backup we just created
  $ kanctl  perform from restore s3backup-j4z6f 

.. todo::

  Add resulting action set.

