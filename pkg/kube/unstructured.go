package kube

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// FetchUnstructuredObject returns the referenced API object as a map[string]interface{}
func FetchUnstructuredObject(resource schema.GroupVersionResource, namespace, name string) (runtime.Unstructured, error) {
	cli, err := client()
	if err != nil {
		return nil, err
	}
	return fetchCR(cli, resource, namespace, name)
}

func fetchCR(cli dynamic.Interface, resource schema.GroupVersionResource, namespace, name string) (runtime.Unstructured, error) {
	return cli.Resource(resource).Namespace(namespace).Get(name, metav1.GetOptions{})
}

// ListUnstructuredObject returns the referenced API objects as a map[string]interface{}
func ListUnstructuredObject(resource schema.GroupVersionResource, namespace string) (runtime.Unstructured, error) {
	cli, err := client()
	if err != nil {
		return nil, err
	}
	return listCR(cli, resource, namespace)
}

func listCR(cli dynamic.Interface, resource schema.GroupVersionResource, namespace string) (runtime.Unstructured, error) {
	//return cli.Resource(resource).Namespace(namespace).List(metav1.ListOptions{})
	return cli.Resource(resource).List(metav1.ListOptions{})
}

func client() (dynamic.Interface, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}
