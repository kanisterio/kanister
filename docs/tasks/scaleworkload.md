# Using ScaleWorkload function with output artifact {#scaleworkloadexample}

`ScaleWorkload` function can be used to scale a workload to specified
replicas. It automatically sets the original replica count of the
workload as output artifact, which makes using `ScaleWorkload` function
in blueprints a lot easier.

Below is an example of how this function can be used

``` yaml
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
```
