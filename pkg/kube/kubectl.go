package kube

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func CreateResourceFromSpecs(specs string) error {
	// TODO: Create namespace if doesn't exists before creating an resource
	f := cmdutil.NewFactory(genericclioptions.NewConfigFlags(false))
	request := f.NewBuilder().
		Unstructured().
		NamespaceParam(metav1.NamespaceDefault).
		DefaultNamespace().
		Stream(strings.NewReader(specs), "resource").
		Flatten().
		Do()
	err := request.Err()
	if err != nil {
		return err
	}
	err = request.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		_, err = resource.
			NewHelper(info.Client, info.Mapping).
			WithFieldManager("kanister-create").
			Create(info.Namespace, true, info.Object)
		return err
	})
	return err
}
