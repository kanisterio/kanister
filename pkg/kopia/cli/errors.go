package cli

import (
	"github.com/pkg/errors"
)

var (
	// flag errors
	ErrInvalidCommonArgs = errors.New("common flag expects at most one cli.CommonArgs argument")
	ErrInvalidCacheArgs  = errors.New("cache flag expects at most one cli.CacheArgs argument")
	ErrInvalidFlag       = errors.New("invalid flag")
	ErrInvalidID         = errors.New("invalid ID")
	ErrInvalidTargetPath = errors.New("invalid target path")
	ErrInvalidBackupPath = errors.New("invalid backup path")
	ErrInvalidTag        = errors.New("invalid tag")

	// storage errors
	ErrUnsupportedStorage = errors.New("unsupported storage")
)
