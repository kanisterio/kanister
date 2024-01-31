package azure

import "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"

//
// Azure flags.
//

// Prefix creates a new Azure prefix flag with a given prefix.
func Prefix(prefix string) flag.Applier {
	return flag.NewStringFlag("--prefix", prefix)
}

// AzureContainer creates a new Azure container flag with a given container name.
func AzureCountainer(name string) flag.Applier {
	return flag.NewStringFlag("--container", name)
}
