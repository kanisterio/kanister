package kube

import (
	"encoding/json"

	"github.com/pkg/errors"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// CreateSpec creates a spec object from the given object
// Expects a Kubernetes object which implements runtime.Object, in particular,
// either a Deployment or StatefulSet object
func CreateSpec(obj runtime.Object) ([]byte, error) {
	switch obj.(type) {
	case *v1beta1.Deployment, *v1beta1.StatefulSet:
		return json.Marshal(obj)
	default:
		return nil, errors.New("Unsupported type for spec serialization: Expected Deployment or StatefulSet")
	}
}

// GetStatefulSetFromSpec returns a StatefulSet object created from the given bytes
func GetStatefulSetFromSpec(spec []byte) (*v1beta1.StatefulSet, error) {
	ss := &v1beta1.StatefulSet{}
	if err := decodeSpecIntoObject(spec, ss); err != nil {
		return nil, err
	}
	clearObjectMetaFields(ss.GetObjectMeta())
	return ss, nil
}

// GetDeploymentFromSpec returns a Deployment object created from the given spec
func GetDeploymentFromSpec(spec []byte) (*v1beta1.Deployment, error) {
	dep := &v1beta1.Deployment{}
	if err := decodeSpecIntoObject(spec, dep); err != nil {
		return nil, err
	}
	clearObjectMetaFields(dep.GetObjectMeta())
	return dep, nil
}

func decodeSpecIntoObject(spec []byte, intoObj runtime.Object) error {
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, intoObj); err != nil {
		return errors.Wrap(err, "Failed to decode spec into object")
	}
	return nil
}

// TODO: Should any other fields should be removed as well?
func clearObjectMetaFields(obj metav1.Object) {
	obj.SetResourceVersion("")
}

// TODO: Add helper(s) for setting fields of interest, such as namespace
