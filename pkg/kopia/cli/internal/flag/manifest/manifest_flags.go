package manifest

import "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"

// Filter creates a new filter flag with a given filter.
func Filter(filter string) flag.Applier {
	return flag.NewStringFlag("--filter", filter)
}
