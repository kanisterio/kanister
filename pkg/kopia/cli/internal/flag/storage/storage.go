package storage

import (
	"sync"

	cmdlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
	"github.com/kanisterio/kanister/pkg/log"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/azure"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/fs"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/gcs"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/s3"
)

// StorageOption is a function that sets a storage option.
type StorageOption func(*model.StorageFlag)

// WithLogger sets the logger for the storage.
func WithLogger(logger log.Logger) StorageOption {
	return func(s *model.StorageFlag) {
		s.Logger = logger
	}
}

// WithFactory sets the storage args builder factory for the storage.
func WithFactory(factory model.StorageBuilderFactory) StorageOption {
	return func(s *model.StorageFlag) {
		s.Factory = factory
	}
}

var (
	// factoryOnce is used to initialize the factory once.
	factoryOnce sync.Once
	// factory creates a new StorageBuilder by LocType.
	factory = model.BuildersFactory{}
)

func Storage(location model.Location, repoPathPrefix string, opts ...StorageOption) model.StorageFlag {
	factoryOnce.Do(func() {
		// Register storage builders.
		factory[rs.LocTypeAzure] = azure.New
		factory[rs.LocTypeS3] = s3.New
		factory[rs.LocTypes3Compliant] = s3.New
		factory[rs.LocTypeGCS] = gcs.New
		factory[rs.LocTypeFilestore] = fs.New
	})
	// create a new storage with the given location, repo path prefix and defaults.
	s := model.StorageFlag{
		Location:       location,
		RepoPathPrefix: repoPathPrefix,
		Logger:         &cmdlog.NopLogger{},
		Factory:        &factory,
	}
	// apply storage options.
	for _, opt := range opts {
		opt(&s)
	}
	return s
}
