package kube

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Operation string

const (
	CreateOperation Operation = "create"
)

// KubectlOperation implements methods to perform kubectl operations
type KubectlOperation struct {
	factory   cmdutil.Factory
	specs     io.Reader
	namespace string
}

// NewKubectlOperations returns new KubectlOperations object
func NewKubectlOperations(specsString, namespace string) *KubectlOperation {
	return &KubectlOperation{
		factory:   cmdutil.NewFactory(genericclioptions.NewConfigFlags(false)),
		specs:     strings.NewReader(specsString),
		namespace: namespace,
	}
}

// Execute executes kubectl operation
func (k *KubectlOperation) Execute(op Operation) error {
	switch op {
	case CreateOperation:
		return k.create()
	default:
		return errors.New("not implemented")
	}
}

func (k *KubectlOperation) create() error {
	// TODO: Create namespace if doesn't exists before creating an resource
	result := k.factory.NewBuilder().
		Unstructured().
		NamespaceParam(k.namespace).
		Stream(k.specs, "resource").
		Flatten().
		Do()
	err := result.Err()
	if err != nil {
		return err
	}
	err = result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		namespace := k.namespace
		// Override namespace if the namespace is set in resource specs
		if info.Namespace != "" {
			namespace = info.Namespace
		}
		_, err = resource.
			NewHelper(info.Client, info.Mapping).
			WithFieldManager("kanister-create").
			Create(namespace, true, info.Object)
		return err
	})
	return err
}
