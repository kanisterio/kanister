.. _tutorial:

Tutorial
********

In this tutorial you'll deploy a simple application in Kubernetes. We'll start
by invoking a trivial Kanister action, then incrementally use more
of Kanister's features to manage the application's data.

.. contents:: Tutorial Overview
  :local:

Prerequisites
=============

* Kubernetes ``1.16`` or higher. For cluster version lower than ``1.16``,
  we recommend installing Kanister version ``0.62.0`` or lower.

* `kubectl <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed
  and setup

* `helm <https://helm.sh>`_ installed and initialized using the command `helm init`

* docker

* A running Kanister controller. See :ref:`install`

* Access to an S3 bucket and credentials.

Example Application
===================

This tutorial begins by deploying a sample application. The application is
contrived, but useful for demonstrating Kanister's features. The application
appends the current time to a log file every second. The application's container
includes the aws command-line client which we'll use later in the tutorial. The
application is installed in the ``default`` namespace.

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: time-logger
  spec:
    replicas: 1
    template:
      metadata:
        labels:
          app: time-logger
      spec:
        containers:
        - name: test-container
          image: amazon/aws-cli
          command: ["sh", "-c"]
          args: ["while true; do for x in $(seq 1200); do date >> /var/log/time.log; sleep 1; done; truncate /var/log/time.log --size 0; done"]
  EOF


Invoking Kanister Actions
=========================

Kanister CustomResources are created in the same namespace as
the Kanister controller.

The first Kanister CustomResource we're going to deploy is a Blueprint.
Blueprints are a set of instructions that tell the controller how to perform
actions on an application. An action consists of one or more phases. Each phase
invokes a :doc:`Kanister Function </functions>`. All Kanister functions accept a
list of strings. The ``args`` field in a Blueprint's phase is rendered and passed
into the specified Function.

For more on CustomResources in Kanister, see :ref:`architecture`.


The Blueprint we'll create has a single action called ``backup``.  The action
``backup`` has a single phase named ``backupToS3``. ``backupToS3`` invokes the
Kanister function ``KubeExec``, which is similar to invoking ``kubectl exec ...``.
At this stage, we'll use ``KubeExec`` to echo our time log's name and
:doc:`Kanister's parameter templating </functions>` to specify the container
with our log.


First Blueprint
---------------

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: Blueprint
  metadata:
    name: time-log-bp
    namespace: kanister
  actions:
    backup:
      phases:
      - func: KubeExec
        name: backupToS3
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod: "{{ index .Deployment.Pods 0 }}"
          container: test-container
          command:
            - sh
            - -c
            - echo /var/log/time.log
  EOF

Once we create a Blueprint, we can see its events by using the following command:

.. code-block:: yaml

  $ kubectl --namespace kanister describe Blueprint time-log-bp
  Events:
    Type     Reason    Age   From                 Message
    ----     ------    ----  ----                 -------
    Normal   Added      4m   Kanister Controller  Added blueprint time-log-bp

When a blueprint resource is created, it goes through a
`validating webhook <https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook>`_
controller that validates the resource. Refer to :ref:`this<kanctlvalidate>`
documentation for more details.

The next CustomResource we'll deploy is an ActionSet. An ActionSet is created
each time you want to execute any Kanister actions. The ActionSet contains all
the runtime information the controller needs during execution. It may contain
multiple actions, each acting on a different Kubernetes object. The ActionSet
we're about to create in this tutorial specifies the ``time-logger`` Deployment we
created earlier and selects the ``backup`` action inside our Blueprint.


First ActionSet
---------------

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
  spec:
    actions:
    - name: backup
      blueprint: time-log-bp
      object:
        kind: Deployment
        name: time-logger
        namespace: default
  EOF

Get the Action's Status
-----------------------

The controller watches its namespace for any ActionSets we create.  Once it
sees a new ActionSet, it will start executing each action. Since our example is
pretty simple, it's probably done by the time you finished reading this. Let's
look at the updated status of the ActionSet and tail the controller logs.

.. code-block:: bash

  # get the ActionSet status
  $ kubectl --namespace kanister get actionsets.cr.kanister.io -o yaml

  # check the controller log
  $ kubectl --namespace kanister get pod -l app=kanister-operator

ActionSet's ``Status.Progress.RunningPhase`` field can be used to figure out
the phase being run currently, for a particular action. Once the ActionSet has
completed, this value is set to ``""``.

During execution, Kanister controller emits events to the respective ActionSets.
The execution transitions of an ActionSet can be seen by using the following command:

.. code-block:: bash

  $ kubectl --namespace kanister describe actionset <ActionSet Name>
  Events:
    Type    Reason           Age   From                 Message
    ----    ------           ----  ----                 -------
    Normal  Started Action   23s   Kanister Controller  Executing action backup
    Normal  Started Phase    23s   Kanister Controller  Executing phase backupToS3
    Normal  Update Complete  19s   Kanister Controller  Updated ActionSet 'ActionSet Name' Status->complete
    Normal  Ended Phase      19s   Kanister Controller  Completed phase backupToS3

In case of an action failure, the Kanister controller will emit failure events to both
the ActionSet and its associated Blueprint.

Consuming ConfigMaps
====================

Congrats on running your first Kanister action! We were able to get data out of
time-logger, but if we want to really protect time-logger's precious log,
you'll need to back it up outside Kubernetes.  We'll choose where to store the
log based on values in a ConfigMap.  ConfigMaps are referenced in an ActionSet,
which are fetched by the controller and made available to Blueprints through
parameter templating.

For more on templating in Kanister, see :ref:`templates`.

In this section of the tutorial, we're going to use a ConfigMap to choose where
to backup our time log. We'll name our ConfigMap and consume it through
argument templating in the Blueprint. We'll map the name to a ConfigMap
reference in the ActionSet.

We create the ConfigMap with an S3 path where we'll eventually push our time
log. Please change the bucket path in the following ConfigMap to something you
have access to.


.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: s3-location
    namespace: kanister
  data:
    path: s3://time-log-test-bucket/tutorial
  EOF

We modify the Blueprint to consume the path from the ConfigMap. We give it a
name ``location`` in the ``configMapNames`` section. We can access the values in the
map through Argument templating. For now we'll just print the path name to
stdout, but eventually we'll backup the time log to that path.

.. code-block:: yaml

  cat <<EOF | kubectl apply -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: Blueprint
  metadata:
    name: time-log-bp
    namespace: kanister
  actions:
    backup:
      configMapNames:
      - location
      phases:
      - func: KubeExec
        name: backupToS3
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod:  "{{ index .Deployment.Pods 0 }}"
          container: test-container
          command:
            - sh
            - -c
            - |
              echo /var/log/time.log
              echo "{{ .ConfigMaps.location.Data.path }}"
  EOF

We create a new ActionSet that maps the name in the Blueprint, ``location``, to
a reference to the ConfigMap we just created.

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
  spec:
    actions:
    - name: backup
      blueprint: time-log-bp
      object:
        kind: Deployment
        name: time-logger
        namespace: default
      configMaps:
        location:
          name: s3-location
          namespace: kanister
  EOF

You can check the controller logs to see if your bucket path rendered
successfully.

Consuming Secrets
=================

In order for us to actually push the time log to S3, we'll need to use AWS
credentials. In Kubernetes, credentials are stored in secrets. Kanister supports
Secrets in the same way it supports ConfigMaps. The secret is named and rendered
in the Blueprint. The name to reference mapping is created in the ActionSet.

In our example, we'll need to use secrets to push the time log to S3.

.. warning::

  Secrets may contain sensitive information. It is up to the author of each
  Blueprint to guarantee that secrets are not logged.

This step requires a bit of homework. You'll need to create aws credentials that
have read/write access to the bucket you specified in the ConfigMap.
Base64 credentials and put them below.

.. code-block:: bash

  echo -n "YOUR_KEY" | base64


.. code-block:: yaml

  apiVersion: v1
  kind: Secret
  metadata:
    name: aws-creds
    namespace: kanister
  type: Opaque
  data:
    aws_access_key_id: XXXX
    aws_secret_access_key: XXXX


Give the secret the name ``aws`` in the Blueprint the secret in the ``secretNames``
section. We can then consume it through templates and assign it to bash
variables. Because we now have access to the bucket in the ConfigMap, we can
also push the log to S3. In this Secret, we store the credentials as binary
data. We can use the templating engine ``toString`` and ``quote`` functions, courtesy of sprig.

For more on this templating, see :ref:`templates`

.. code-block:: yaml

  cat <<EOF | kubectl apply -f -
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
      phases:
      - func: KubeExec
        name: backupToS3
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod: "{{ index .Deployment.Pods 0 }}"
          container: test-container
          command:
            - sh
            - -c
            - |
              AWS_ACCESS_KEY_ID={{ .Secrets.aws.Data.aws_access_key_id | toString }}         \
              AWS_SECRET_ACCESS_KEY={{ .Secrets.aws.Data.aws_secret_access_key | toString }} \
              aws s3 cp /var/log/time.log {{ .ConfigMaps.location.Data.path | quote }}
  EOF

Create a new ActionSet that has the name-to-Secret reference in its action's
``secrets`` field.

.. code-block:: yaml

  cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
  spec:
    actions:
    - name: backup
      blueprint: time-log-bp
      object:
        kind: Deployment
        name: time-logger
        namespace: default
      configMaps:
        location:
          name: s3-location
          namespace: kanister
      secrets:
        aws:
          name: aws-creds
          namespace: kanister
  EOF

Configuring Labels and Annotations of Kanister function pods using ActionSet
============================================================================

We create an ActionSet each time we want to execute a
Kanister action. This action is going to be defined in Kanister blueprints
using Kanister functions.

If the specified Kanister function creates a pod, labels and annotations of
that pod can be configured via ``podLabels`` and ``podAnnotations`` fields
of the ActionSet resource.

Once these fields are configured in the ActionSet resource, all the pods that
are created by Kanister functions that is run by this ActionSet would have these
labels and annotations.

.. code-block:: yaml

  $ cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3backup-
    namespace: kanister
  spec:
    actions:
    - name: backup
      blueprint: time-log-bp
      podLabels:
        labelKeyZero: labelValueZero
        labelKeyOne: labelValueone
      podAnnotations:
        annotationKey: annotationValue
      object:
        kind: Deployment
        name: time-logger
        namespace: default
  EOF


Artifacts
=========

At this point, we have successfully backed up our application's data to S3. In
order to retrieve the information we have pushed to S3, we must store a reference
to that data. In Kanister we call these references Artifacts. Kanister's
Artifact mechanism manages data we have externalized.  Once an artifact has been
created, it can be consumed in a Blueprint to retrieve data from external
sources.  Any time Kanister is used to protect data, it creates a corresponding
Artifact.

An Artifact is a set of key-value pairs. It is up to the Blueprint author to
ensure that the data referenced by Artifacts is valid. Artifacts passed into
Blueprints are Input Artifacts and Artifacts created by Blueprints are output
Artifacts.

Output Artifacts
----------------

In our example, we'll create an outputArtifact called ``timeLog`` that contains
the full path of our data in S3. This path's base will be configured using a
ConfigMap.

.. code-block:: yaml

  cat <<EOF | kubectl apply -f -
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
            path: '{{ .ConfigMaps.location.Data.path }}/time-log/'
      phases:
        - func: KubeExec
          name: backupToS3
          args:
            namespace: "{{ .Deployment.Namespace }}"
            pod: "{{ index .Deployment.Pods 0 }}"
            container: test-container
            command:
              - sh
              - -c
              - |
                AWS_ACCESS_KEY_ID={{ .Secrets.aws.Data.aws_access_key_id | toString }}         \
                AWS_SECRET_ACCESS_KEY={{ .Secrets.aws.Data.aws_secret_access_key | toString }} \
                aws s3 cp /var/log/time.log {{ .ConfigMaps.location.Data.path }}/time-log/
  EOF

If you re-execute this Kanister Action, you'll be able to see the Artifact in the
ActionSet status.

If you use a ``DeferPhase``, below is how you can set the output artifact
from the output that is being generated from ``DeferPhase`` as shown below.

.. code-block:: yaml

  cat <<EOF | kubectl apply -f -
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
            path: '{{ .ConfigMaps.location.Data.path }}/time-log/'
        deferPhaseArt:
          keyValue:
            time: "{{ .DeferPhase.Output.bkpCompletedTime }}"
      phases:
        - func: KubeExec
          name: backupToS3
          args:
            namespace: "{{ .Deployment.Namespace }}"
            pod: "{{ index .Deployment.Pods 0 }}"
            container: test-container
            command:
              - sh
              - -c
              - |
                echo "Main Phase"
      deferPhase:
        func: KubeExec
        name: saveBackupTime
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod: "{{ index .Deployment.Pods 0 }}"
          container: test-container
          command:
            - sh
            - -c
            - |
              echo "DeferPhase"
              kando output bkpCompletedTime "10Minutes"
  EOF


Output from the previous phases can also be used in the ``DeferPhase`` like it
is used in normal scenarios.

Input Artifacts
---------------

Kanister can consume artifacts it creates using ``inputArtifacts``.
``inputArtifacts`` are named in Blueprints and are explicitly listed in the
ActionSet.

In our example we'll restore an older time log. We have already pushed one to S3
and created an Artifact using the backup action. We'll now restore that time log
by using a new restore action.

We create a new ActionSet on our ``time-logger`` deployment with the action name
``restore``. This time we also include the full path in S3 as an Artifact.

.. code-block:: yaml

  cat <<EOF | kubectl create -f -
  apiVersion: cr.kanister.io/v1alpha1
  kind: ActionSet
  metadata:
    generateName: s3restore
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
  EOF

We add a restore action to the Blueprint. This action does not need the
ConfigMap because the ``inputArtifact`` contains the fully specified path.

.. code-block:: yaml

  cat <<EOF | kubectl apply -f -
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
            path: '{{ .ConfigMaps.location.Data.path }}/time-log/'
      phases:
        - func: KubeExec
          name: backupToS3
          args:
            namespace: "{{ .Deployment.Namespace }}"
            pod: "{{ index .Deployment.Pods 0 }}"
            container: test-container
            command:
              - sh
              - -c
              - |
                AWS_ACCESS_KEY_ID={{ .Secrets.aws.Data.aws_access_key_id | toString }}         \
                AWS_SECRET_ACCESS_KEY={{ .Secrets.aws.Data.aws_secret_access_key | toString }} \
                aws s3 cp /var/log/time.log {{ .ConfigMaps.location.Data.path }}/time-log/
    restore:
      secretNames:
      - aws
      inputArtifactNames:
      - timeLog
      phases:
      - func: KubeExec
        name: restoreFromS3
        args:
          namespace: "{{ .Deployment.Namespace }}"
          pod: "{{ index .Deployment.Pods 0 }}"
          container: test-container
          command:
            - sh
            - -c
            - |
              AWS_ACCESS_KEY_ID={{ .Secrets.aws.Data.aws_access_key_id | toString }}         \
              AWS_SECRET_ACCESS_KEY={{ .Secrets.aws.Data.aws_secret_access_key | toString }} \
              aws s3 cp {{ .ArtifactsIn.timeLog.KeyValue.path | quote }} /var/log/time.log
  EOF

We can check the controller logs to see that the time log was restored
successfully.


Time
====

It is often useful to include the current time as parameters to an action.
Kanister provides the job's start time in UTC. We can modify the Blueprint's
output artifact to include the day the backup was taken:

.. code-block:: yaml

  outputArtifacts:
    timeLog:
      path: '{{ .ConfigMaps.location.Data.path }}/time-log/{{ toDate "2006-01-02T15:04:05.999999999Z07:00" .Time  | date "2006-01-02" }}'

For more on using the time template parameter, see :ref:`templates` .


Using kanctl to Chain ActionSets
================================

So far in this tutorial, we have shown you how to manually create action
sets via YAML files. In some cases, an action depends on a previous action,
and manually updating the action set to use artifacts created by the
previous action set can be cumbersome. In situations like this, it is
useful to instead use ``kanctl``. To learn how to leverage ``kanctl`` to
create action sets, see :ref:`architecture` .

Next Step
=========

Congratulations! You have reached the end of this long tutorial! 🎉🎉🥳🥳

Don't stop here. There are many more example blueprints on the Kanister GitHub
`repository <https://github.com/kanisterio/kanister/tree/master/examples>`_
to explore. Use them to help you define your next blueprint.

We would love to hear from you. If you have any feedback or questions, find us
on Slack at `kanisterio.slack.com <https://kanisterio.slack.com>`_.
