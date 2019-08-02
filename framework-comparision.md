# Framework Comparision

This document is a comparative analysis of framework between operator-sdk and kubebuilder to determine which is to be used for writing a kanister operator.


## Operator-SDK

- Provider & Maintainer : core-os & community
- Generated deepcopy functions and controllers separately.
- Generated APIs without OpenAPI validation by default, however there is an option to generate those using *operator-sdk generate openapi*
- Empty interfaces are allowed and deep copy functions are generated.
- Support: The features are being added and issues are being addressed currently so the project is actively maintained. Since openshift has significant market consumption in the provided kubernetes space, RedHat might continue to support it of a long time but as of now there's no clarity on whether the support will remain in the future. 
- Uses native *k8s*
- Provides *client.Client*, *manager.Manager* and *reconcile.Reconciler* in the controller to work with CRDs
- Need to use codegen to generate clients and listers for external use (out of controller)

## Kubebuilder
- Provider & Maintainer : community (supported by Google). Now a part of *kubernetes-sigs*
- Generated deepcopy functions and controllers simultaneously. Also provides a support for generating the api and resource specifications separately, resources include rbac for the CRDs.
- Generated APIs with OpenAPI validation by default. Validation can be turned off by *kubebuilder:validation:Optional*
- Empty interfaces are supposed to be allowed however this functionality seems to be broken in the latest released version as well as off the top of the master.
- Support: kubebuilder seems to have picked a lot of pace, including references in the *Programming Kubernetes*, it is already becoming the de-facto tool of implementation for controllers. Being a part of *kubernetes-sigs* and coming from Google, there is a serious support from the kubernetes community. 
- Uses *+kubebuilder* which is custom. There is a support of using codegen tags.
- Provides *client.Client*, *controller-runtime.Manager* and type of reconciler for example *type ActionSetReconciler* in the controller to work with CRDs
- Need to use codegen to generate clients and listers for external use (out of controller)

