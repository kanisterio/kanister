package filter

import (
	"context"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kanisterio/kanister/pkg/discovery"
	"github.com/kanisterio/kanister/pkg/kube"
)

type DefaultSuite struct {
	gvrs []schema.GroupVersionResource
}

var _ = Suite(&DefaultSuite{})

func (s *DefaultSuite) SetUpSuite(c *C) {
	ctx := context.Background()
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.gvrs, err = discovery.AllGVRs(ctx, cli.Discovery())
	c.Assert(err, IsNil)

}

func gvrSet(gvrs []schema.GroupVersionResource) map[schema.GroupVersionResource]struct{} {
	s := make(map[schema.GroupVersionResource]struct{}, len(gvrs))
	for _, gvr := range gvrs {
		s[gvr] = struct{}{}
	}
	return s
}

// knownCoreGVRs returns specific GVRs that will match the CoreGroups filter.
// This list may change slightly between different clusters. This list is likely
// to be common between different clusters.
func knownCoreGVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "podtemplates"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "bindings"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "replicationcontrollers"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "componentstatuses"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "deployments"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "replicasets"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "networkpolicies"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "daemonsets"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "ingresses"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "replicationcontrollers"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "podsecuritypolicies"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "controllerrevisions"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "selfsubjectaccessreviews"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "subjectaccessreviews"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "selfsubjectrulesreviews"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "localsubjectaccessreviews"},
		schema.GroupVersionResource{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
		schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
		schema.GroupVersionResource{Group: "batch", Version: "v1beta1", Resource: "cronjobs"},
		schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
		schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: "poddisruptionbudgets"},
		schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: "podsecuritypolicies"},
	}
}

func knownProtectedGVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
		schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
		schema.GroupVersionResource{Group: "batch", Version: "v1beta1", Resource: "cronjobs"},
	}
}

func knownUnprotectedGVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "podtemplates"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "bindings"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "replicationcontrollers"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "componentstatuses"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "replicasets"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "networkpolicies"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "replicationcontrollers"},
		schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "podsecuritypolicies"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
		schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "controllerrevisions"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "selfsubjectaccessreviews"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "subjectaccessreviews"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "selfsubjectrulesreviews"},
		schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "localsubjectaccessreviews"},
		schema.GroupVersionResource{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
		schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
		schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: "poddisruptionbudgets"},
		schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: "podsecuritypolicies"},
	}
}

func (s *DefaultSuite) TestUnprotectedGVRs(c *C) {
	protected := gvrSet(UnprotectedMatcher.Exclude(s.gvrs))
	for _, gvr := range knownProtectedGVRs() {
		_, ok := protected[gvr]
		c.Assert(ok, Equals, true, Commentf("GVR should be in protected list: %v", gvr))
	}
	for _, gvr := range knownUnprotectedGVRs() {
		_, ok := protected[gvr]
		c.Assert(ok, Equals, false, Commentf("GVR should not be in protected list: %v", gvr))
	}
}
