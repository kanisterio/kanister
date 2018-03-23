.. _functions:

Kanister Functions
******************

Kanister Functions are written in go and are compiled when building the
controller. They are referenced by Blueprints phases. A Kanister Function
implements the following go interface:

.. code-block:: go

  // Func allows custom actions to be executed.
  type Func interface {
      Name() string
      Exec(ctx context.Context, args ...string) error
  }

Kanister Functions are registered by the return value of `Name()`, which must be
static.

Each phase in a Blueprint executes a Kanister Function.  The `Func` field in
a `BlueprintPhase` is used to lookup a Kanister Function.  After
`BlueprintPhase.Args` are rendered, they are passed into the Kanister Function's
`Exec()` method.

Existing Functions
==================

The Kanister controller ships with the following Kanister Functions out-of-the-box
that provide integration with Kubernetes:

KubeExec
--------

KubeExec is similar to running

.. code-block:: bash

  `kubectl exec -it --namespace <NAMESPACE> <POD> -c <CONTAINER> [CMD LIST...]

It requires at least four arguments. The first three arguments are used to
determine the container to exec into. The remaining arguments are grouped and
executed as a command.

The arguments are:

#. `namespace`
#. `pod`
#. `container`
#. [`command`]

KubeTask
++++++++

KubeTask spins up a new container and executes a command via a Kubernetes job.
This allows you to run a new Pod from a Blueprint.

KubeTask takes the following three arguments:

#. `namespace`
#. `image`
#. [`command`]


Registering Functions
---------------------

Kanister can be extended by registering new Kanister Functions.

Kanister Functions are registered using a similar mechanism to `database/sql
<https://golang.org/pkg/database/sql/>`_ drivers. To register new Kanister
Functions, import a package with those new functions into the controller and
recompile it.
