package flag

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// Applier is an interface for applying flags to a command.
type Applier = internal.Applier

// Flag is a flag interface.
type Flag interface {
	Flag() string
	Applier
}

// FlagApplierHandler type is an adapter to allow the use of
// ordinary functions as FlagApplier. If f is a function
// with the appropriate signature, FlagApplierHandler(f) is a
// FlagApplier that calls f.
type FlagApplierHandler func(safecli.CommandAppender) error

func (h FlagApplierHandler) Apply(cli safecli.CommandAppender) error {
	return h(cli)
}

// Apply attaches multiple flags to the CLI.
// If any of the flags encounter an error during the Apply process,
// the error is returned and no changes are made to the CLI.
// If no error is encountered, the flags are appended to the CLI.
func Apply(cli safecli.CommandAppender, flags ...Applier) error {
	// create a new builder which will be used to apply the flags
	// to avoid mutating the CLI if an error is encountered.
	b := safecli.NewBuilder()
	for _, f := range flags {
		if f == nil {
			continue // if the flag is nil, skip it
		}
		if err := f.Apply(b); err != nil {
			return err
		}
	}
	cli.Append(b) // if no error, append the flags to the CLI
	return nil
}

// SwitchFlag creates a new switch flag with a given flag name.
func SwitchFlag(flag string) Applier {
	return FlagApplierHandler(func(cli safecli.CommandAppender) error {
		cli.AppendLoggable(flag)
		return nil
	})
}

// BoolFlag creates a new bool flag with a given flag name and value.
type boolFlag struct {
	flag    string
	enabled bool
}

func (f boolFlag) Apply(cli safecli.CommandAppender) error {
	if f.enabled {
		cli.AppendLoggable(f.flag)
	}
	return nil
}

// NewBoolFlag creates a new bool flag with a given flag name and value.
func NewBoolFlag(flag string, enabled bool) Applier {
	if flag == "" {
		return ErrorFlag(cli.ErrInvalidFlag)
	}
	return boolFlag{flag, enabled}
}

// StringFlag is a flag with a string value.
// If the value is empty, the flag is not applied.
type stringFlag struct {
	flag     string // flag name
	val      string // flag value
	redacted bool   // output the value as redacted
}

func (f stringFlag) Apply(cli safecli.CommandAppender) error {
	if f.val == "" {
		return nil
	}
	if f.redacted {
		f.applyRedacted(cli)
	} else {
		f.applyLoggable(cli)
	}
	return nil
}

func (f stringFlag) applyLoggable(cli safecli.CommandAppender) error {
	if f.flag == "" {
		cli.AppendLoggable(f.val)
	} else {
		cli.AppendLoggableKV(f.flag, f.val)
	}
	return nil
}

func (f stringFlag) applyRedacted(cli safecli.CommandAppender) error {
	if f.flag == "" {
		cli.AppendRedacted(f.val)
	} else {
		cli.AppendRedactedKV(f.flag, f.val)
	}
	return nil
}

// newStringFlag creates a new string flag with a given flag name and value.
func newStringFlag(flag, val string, redacted bool) Applier {
	if flag == "" && val == "" {
		return ErrorFlag(cli.ErrInvalidFlag)
	}
	return stringFlag{flag: flag, val: val, redacted: redacted}
}

// NewStringFlag creates a new string flag with a given flag name and value.
func NewStringFlag(flag, val string) Applier {
	return newStringFlag(flag, val, false)
}

// NewRedactedStringFlag creates a new string flag with a given flag name and value.
func NewRedactedStringFlag(flag, val string) Applier {
	return newStringFlag(flag, val, true)
}

// NewStringValue creates a new string flag with a given value.
func NewStringValue(val string) Applier {
	return newStringFlag("", val, false)
}

// flagAppliers is a collection of string flags.
type flagAppliers []Applier

func (f flagAppliers) Apply(cli safecli.CommandAppender) error {
	for _, flag := range f {
		flag.Apply(cli)
	}
	return nil
}

// NewFlags creates a new collection of flags.
func NewFlags(flags ...Applier) Applier {
	return flagAppliers(flags)
}

// errorFlag is a flag that does nothing.
type errorFlag struct {
	err error
}

func (f errorFlag) Apply(safecli.CommandAppender) error {
	return f.err
}

// DoNothingFlag creates a new void flag.
func DoNothingFlag() Applier {
	return errorFlag{}
}

// ErrorFlag creates a new flag that returns an error.
func ErrorFlag(err error) Applier {
	return errorFlag{err}
}
