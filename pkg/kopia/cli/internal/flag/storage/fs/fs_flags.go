package fs

import "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"

//
// Filestore flags.
//

// Path creates a new path flag with a given path.
func Path(path string) flag.Applier {
	return flag.NewStringFlag("--path", path)
}
