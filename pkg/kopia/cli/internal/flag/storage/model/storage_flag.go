package model

import (
	log "github.com/kanisterio/kanister/pkg/log"

	"github.com/kanisterio/kanister/pkg/safecli"
	"github.com/pkg/errors"

	cmdlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
)

var (
	ErrInvalidFactor = errors.New("factory cannot be nil")
)

// StorageFlag is a set of flags that are used to create a StorageFlag sub command.
type StorageFlag struct {
	Location       Location
	RepoPathPrefix string

	Factory StorageBuilderFactory
	Logger  log.Logger
}

// GetLogger returns the logger.
// If the logger is nil, it returns a NopLogger.
func (s StorageFlag) GetLogger() log.Logger {
	if s.Logger == nil {
		s.Logger = &cmdlog.NopLogger{}
	}
	return s.Logger
}

// Apply applies the storage flags to the command.
func (s StorageFlag) Apply(cli safecli.CommandAppender) error {
	if s.Factory == nil {
		return ErrInvalidFactor
	}
	storageBuilder := s.Factory.Create(s.Location.Type())
	storageCLI, err := storageBuilder(s)
	if err != nil {
		return errors.Wrap(err, "failed to apply storage args")
	}
	cli.Append(storageCLI)
	return nil
}
