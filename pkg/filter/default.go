package filter

// UnprotectedMatcher will match all the known resources that don't make sense
// to protect when backing up an application.
var UnprotectedMatcher ResourceMatcher = ResourceMatcher{
	ResourceRequirement{Group: "", Version: "v1", Resource: "bindings"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "componentstatuses"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "endpoints"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "events"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "limitranges"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "namespaces"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "nodes"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "pods"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "podtemplates"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "replicationcontrollers"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "resourcequotas"},
	ResourceRequirement{Group: "", Version: "v1", Resource: "serviceaccounts"},
	ResourceRequirement{Group: "extensions", Version: "v1beta1"},
	ResourceRequirement{Group: "apps", Version: "v1", Resource: "controllerrevisions"},
	ResourceRequirement{Group: "apps", Version: "v1", Resource: "replicasets"},
	ResourceRequirement{Group: "events.k8s.io", Version: "v1beta1", Resource: "events"},
	ResourceRequirement{Group: "authorization.k8s.io"},
	ResourceRequirement{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
	ResourceRequirement{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
	ResourceRequirement{Group: "policy", Version: "v1beta1"},
	ResourceRequirement{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
	ResourceRequirement{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
}
