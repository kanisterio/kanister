package restore

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

// TargetPath creates a new target path argument with a given path.
// If the path is empty, it returns ErrInvalidTargetPath.
func TargetPath(path string) flag.Applier {
	if path == "" {
		return flag.ErrorFlag(cli.ErrInvalidTargetPath)
	}
	return flag.NewStringArgument(path)
}

// IgnorePermissionErrors enables to ignore permission errors.
func IgnorePermissionErrors(enable bool) flag.Applier {
	if enable {
		return flag.NewBoolFlag("--ignore-permission-errors", true)
	}
	return flag.NewBoolFlag("--no-ignore-permission-errors", true)
}

// WriteSparseFiles enables to write files sparsely-allocating
func WriteSparseFiles(enable bool) flag.Applier {
	return flag.NewBoolFlag("--write-sparse-files", enable)
}

// UnsafeIgnoreSource enables to ignore source errors.
func UnsafeIgnoreSource(enable bool) flag.Applier {
	return flag.NewBoolFlag("--unsafe-ignore-source", enable)
}
