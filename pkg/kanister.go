package kanister

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

var (
	funcMu sync.RWMutex
	funcs  = make(map[string]Func)
)

// Func allows custom actions to be executed.
type Func interface {
	Name() string
	Exec(context.Context, ...string) error
}

// TemplateParams are the values that will change between separate runs of Phases.
type TemplateParams struct {
	StatefulSet  *StatefulSetParams
	Deployment   *DeploymentParams
	ArtifactsIn  map[string]crv1alpha1.Artifact
	ArtifactsOut map[string]crv1alpha1.Artifact
	ConfigMaps   map[string]v1.ConfigMap
	Secrets      map[string]v1.Secret
	Time         string
}

// StatefulSetParams are params for stateful sets.
type StatefulSetParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims [][]string
}

// DeploymentParams are params for deployments
type DeploymentParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims [][]string
}

// Register allows Funcs to be references by User Defined YAMLs
func Register(f Func) error {
	funcMu.Lock()
	defer funcMu.Unlock()
	if f == nil {
		return errors.Errorf("kanister: Cannot register nil function")
	}
	if _, dup := funcs[f.Name()]; dup {
		panic("kanister: Register called twice for function " + f.Name())
	}
	funcs[f.Name()] = f
	return nil
}
