.. _scaleworkloadexample:

Using ScaleWorkload function with output artifact
-------------------------------------------------

Traditionally, when ``ScaleWorkload`` functions were used to scale down the
workload, it was required to get the original replica count of the workload and
set it as output artifact. So that it can later be used to scale up the
workload to same number of replicas.

After the new changes, the ``ScaleWorkload`` function automatically sets the
original replica count of the workload as output artifact, which makes using
``ScaleWorkload`` function in blueprints a lot easier.

Below is an example of how this function can be used


.. code-block:: yaml

  apiVersion: cr.kanister.io/v1alpha1
  kind: Blueprint
  metadata:
    name: my-blueprint
  actions:
    backup:
      outputArtifacts:
        backupOutput:
          keyValue:
            origReplicas: "{{ .Phases.shutdownPod.Output.originalReplicaCount }}"
      phases:
      # before scaling replicas 0, ScaleWorkload will get the original replica count
      # to set that as output artifact (originalReplicaCount)
      - func: ScaleWorkload
        name: shutdownPod
        args:
          namespace: "{{ .StatefulSet.Namespace }}"
          name: "{{ .StatefulSet.Name }}"
          kind: StatefulSet
          replicas: 0 # this is the replica count, the STS will scaled to
    restore:
      inputArtifactNames:
        - backupOutput
      phases:
      - func: ScaleWorkload
        name: bringUpPod
        args:
          namespace: "{{ .StatefulSet.Namespace }}"
          name: "{{ .StatefulSet.Name }}"
          kind: StatefulSet
          replicas: "{{ .ArtifactsIn.backupOutput.KeyValue.origReplicas }}"
