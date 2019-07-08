package getter

import (
	"context"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/blockstorage/gcepd"
	"github.com/kanisterio/kanister/pkg/blockstorage/ibm"
)

// Getter is a resolver for a storage provider.
type Getter interface {
	Get(blockstorage.Type, map[string]string) (blockstorage.Provider, error)
}

var _ Getter = (*getter)(nil)

type getter struct{}

// New retuns a new Getter
func New() Getter {
	return &getter{}
}

// Get returns a provider for the requested storage type in the specified region
func (*getter) Get(storageType blockstorage.Type, config map[string]string) (blockstorage.Provider, error) {
	switch storageType {
	case blockstorage.TypeEBS:
		return awsebs.NewProvider(config)
	case blockstorage.TypeSoftlayerBlock:
		return ibm.NewProvider(context.TODO(), config)
	case blockstorage.TypeGPD:
		return gcepd.NewProvider(config)
	case blockstorage.TypeSoftlayerFile:
		config[ibm.SoftlayerFileAttName] = "true"
		return ibm.NewProvider(context.TODO(), config)
	default:
		return nil, errors.Errorf("Unsupported storage type %v", storageType)
	}
}

// Supported returns true if the storage type is supported
func Supported(st blockstorage.Type) bool {
	switch st {
	case blockstorage.TypeEBS:
		return true
	case blockstorage.TypeSoftlayerBlock:
		return true
	case blockstorage.TypeGPD:
		return true
	case blockstorage.TypeSoftlayerFile:
		return true
	default:
		return false
	}
}
