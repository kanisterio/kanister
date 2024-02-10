package server

import (
	"github.com/kanisterio/safecli"
)

const (
	BashBinaryName = "bash"
)

// NewBashBuilder creates a new bash builder.
func NewBashBuilder(cmd *safecli.Builder) *safecli.Builder {
	b := safecli.NewBuilder(BashBinaryName) // create a new bash builder
	b.AppendLoggable("-o", "errexit")       // exit on error
	b.AppendLoggable("-c")                  // read commands from the first non-option argument
	b.Append(cmd)                           // append the command to the bash builder
	return b
}
