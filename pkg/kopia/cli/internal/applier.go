package internal

import "github.com/kanisterio/kanister/pkg/safecli"

// Applier is an interface for applying commands/flags to a CLI.
type Applier interface {
	Apply(safecli.CommandAppender) error
}
