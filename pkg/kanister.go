package kanister

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/param"
)

var (
	funcMu sync.RWMutex
	funcs  = make(map[string]Func)
)

// Func allows custom actions to be executed.
type Func interface {
	Name() string
	RequiredArgs() []string
	Exec(context.Context, param.TemplateParams, map[string]interface{}) error
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
