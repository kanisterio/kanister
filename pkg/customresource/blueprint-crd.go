package customresource

const blueprintCRD = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: blueprints.cr.kanister.io
spec:
  group: cr.kanister.io
  names:
    kind: Blueprint
    listKind: BlueprintList
    plural: blueprints
    singular: blueprint
  scope: Namespaced
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        properties:
          actions:
            additionalProperties:
              properties:
                configMapNames:
                  items:
                    type: string
                  type: array
                inputArtifactNames:
                  items:
                    type: string
                  type: array
                kind:
                  type: string
                name:
                  type: string
                outputArtifacts:
                  additionalProperties:
                    properties:
                      keyValue:
                        additionalProperties:
                          type: string
                        type: object
                      kopiaSnapshot:
                        type: string
                        x-kubernetes-preserve-unknown-fields: true
                    type: object
                  type: object
                phases:
                  items:
                    properties:
                      args:
                        x-kubernetes-preserve-unknown-fields: true
                        type: object
                      func:
                        type: string
                      name:
                        type: string
                      objects:
                        additionalProperties:
                          properties:
                            apiVersion:
                              description: API version of the referent.
                              type: string
                            group:
                              description: API Group of the referent.
                              type: string
                            kind:
                              description: 'Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
                              type: string
                            name:
                              description: 'Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names'
                              type: string
                            namespace:
                              description: 'Namespace of the referent. More info: http://kubernetes.io/docs/user-guide/namespaces'
                              type: string
                            resource:
                              description: Resource name of the referent.
                              type: string
                          type: object
                        type: object
                    type: object
                  type: array
                secretNames:
                  items:
                    type: string
                  type: array
              type: object
            type: object
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
        type: object
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`
