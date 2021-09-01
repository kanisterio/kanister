package kube

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	factory cmdutil.Factory
	specs   io.Reader
}

// NewKubectlOperations returns new KubectlOperations object
func NewKubectlOperations(specsString string) *KubectlOperation {
	return &KubectlOperation{
		factory: cmdutil.NewFactory(genericclioptions.NewConfigFlags(false)),
		specs:   strings.NewReader(specsString),
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
	request := k.factory.NewBuilder().
		Unstructured().
		NamespaceParam(metav1.NamespaceDefault).
		DefaultNamespace().
		Stream(k.specs, "resource").
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
