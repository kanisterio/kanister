package maintenance

import "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"

// NewCustomerOwner creates a new customer owner flag with a given owner.
func CustomerOwner(customOwner string) flag.Applier {
	return flag.NewStringFlag("--owner", customOwner)
}
