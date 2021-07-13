## Runtime reconcile loop

### Events

- Cannot figure out type of event (create/update/delete). Need to have some logic to differentiate among the events
- Need to add finalizers to the resources in order to capture delete events

### Clients

- Generally, uses dynamic client (https://godoc.org/sigs.k8s.io/controller-runtime/pkg/client) which deals with `unstructured.Unstructured` objects. So create/update object with the client, we need to convert `runtime.Object` to `unstructured.Unstructured` (https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured).
- With dynamic client, we don't need code generation. One client works for all types. But there is less type safety.
- It is easier to create sub-resources/child resources for the CR with the dynamic client

### Adding new API

- Adding new resource is relatively easy.
    - create .go file containing type definition in expected path and correct markers for code generation
    - run object generator in controller-tools to generate `runtime.Object` interface for the types
    - create controller implementing Reconciler logic for the API and add the controller to manager (https://godoc.org/sigs.k8s.io/controller-runtime#Manager)

### Conclusion

Runtime-controller focuses more on ease of addition new APIs, creating sub-resources/child resources for a custom resource than event handling


## Informers

### Events

- Calls event handlers on the occurance of events - create, update or delete. No additional logic required to figure out type of events
- No need to add finalizers as it receives delete events

### Clients

- Uses typed client.
- Using typed client needs code generation. While dealing with multiple types, we need to create client for each type so it's type safe
- With typed client, it is easy to configure informers and handlers on the specific types 

### Adding new resource

- Adding new API relatively lengthy process
    - create .go file containing type definition in expected path and correct markers for interface and client generation
    - Run code generator to generate `runtime.Object` interface, clients, listers and informers
    - Add informers to the controller logic. Use generated clients to access and manage resources.

### Conclusion

With typed clients and informers, the controller focuses more on the event handling.
