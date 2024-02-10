package policy

import (
	"math"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagpolicy "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/policy"
)

// RetentionPolicyArgs defines the retention policy arguments.
type RetentionPolicyArgs struct {
	KeepLatest  int
	KeepHourly  int
	KeepDaily   int
	KeepWeekly  int
	KeepMonthly int
	KeepAnnual  int
}

// RetentionPolicyOption defines the retention policy option.
type RetentionPolicyOption func(*RetentionPolicyArgs)

// WithKeepLatest is a retention policy option that sets the number of latest backups to keep.
func WithKeepLatest(keepLatest int) RetentionPolicyOption {
	return func(args *RetentionPolicyArgs) {
		args.KeepLatest = keepLatest
	}
}

// WithKeepHourly is a retention policy option that sets the number of hourly backups to keep.
func WithKeepHourly(keepHourly int) RetentionPolicyOption {
	return func(args *RetentionPolicyArgs) {
		args.KeepHourly = keepHourly
	}
}

// WithKeepDaily is a retention policy option that sets the number of daily backups to keep.
func WithKeepDaily(keepDaily int) RetentionPolicyOption {
	return func(args *RetentionPolicyArgs) {
		args.KeepDaily = keepDaily
	}
}

// WithKeepWeekly is a retention policy option that sets the number of weekly backups to keep.
func WithKeepWeekly(keepWeekly int) RetentionPolicyOption {
	return func(args *RetentionPolicyArgs) {
		args.KeepWeekly = keepWeekly
	}
}

// WithKeepMonthly is a retention policy option that sets the number of monthly backups to keep.
func WithKeepMonthly(keepMonthly int) RetentionPolicyOption {
	return func(args *RetentionPolicyArgs) {
		args.KeepMonthly = keepMonthly
	}
}

// WithKeepAnnual is a retention policy option that sets the number of annual backups to keep.
func WithKeepAnnual(keepAnnual int) RetentionPolicyOption {
	return func(args *RetentionPolicyArgs) {
		args.KeepAnnual = keepAnnual
	}
}

// defaultRetentionPolicy is the default retention policy.
var defaultRetentionPolicy = []RetentionPolicyOption{
	WithKeepLatest(math.MaxInt32),
	WithKeepHourly(0),
	WithKeepDaily(0),
	WithKeepWeekly(0),
	WithKeepMonthly(0),
	WithKeepAnnual(0),
}

// NewRetentionPolicyArgs creates a new retention policy arguments with defaults.
// opts are applied in order, with later options overriding earlier ones.
func NewRetentionPolicyArgs(opts ...RetentionPolicyOption) *RetentionPolicyArgs {
	args := &RetentionPolicyArgs{}
	for _, opt := range append(defaultRetentionPolicy, opts...) {
		opt(args)
	}
	return args
}

// CompressionPolicyArgs defines the compression policy arguments.
type CompressionPolicyArgs struct {
	CompressionAlgorithm string
}

// CompressionPolicyOption defines the compression policy option.
type CompressionPolicyOption func(*CompressionPolicyArgs)

// WithCompressionAlgorithm is a compression policy option that sets the compression algorithm.
func WithCompressionAlgorithm(algo string) CompressionPolicyOption {
	return func(args *CompressionPolicyArgs) {
		args.CompressionAlgorithm = algo
	}
}

// defaultCompressionPolicy is the default compression policy.
var defaultCompressionPolicy = []CompressionPolicyOption{
	WithCompressionAlgorithm(flagpolicy.DefaultCompressionAlgorithm),
}

// NewCompressionPolicyArgs creates a new compression policy arguments with defaults.
// opts are applied in order, with later options overriding earlier ones.
func NewCompressionPolicyArgs(opts ...CompressionPolicyOption) *CompressionPolicyArgs {
	args := &CompressionPolicyArgs{}
	for _, opt := range append(defaultCompressionPolicy, opts...) {
		opt(args)
	}
	return args
}

// SetArgs defines the arguments for the `kopia policy set ...` command.
type SetArgs struct {
	cli.CommonArgs

	// If nil, the default retention policy (defaultRetentionPolicy) will be used.
	RetentionPolicyArgs *RetentionPolicyArgs

	// If nil, the default compression policy (defaultCompressionPolicy) will be used.
	CompressionPolicyArgs *CompressionPolicyArgs
}

// applyDefaults applies the default values to the arguments.
func (a *SetArgs) applyDefaults() {
	if a.RetentionPolicyArgs == nil {
		a.RetentionPolicyArgs = NewRetentionPolicyArgs()
	}
	if a.CompressionPolicyArgs == nil {
		a.CompressionPolicyArgs = NewCompressionPolicyArgs()
	}
}

// Set creates a new `kopia policy set ...` command.
func Set(args SetArgs) (safecli.CommandBuilder, error) {
	args.applyDefaults()
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Policy, command.Set,
		flagpolicy.Global(true),
		toRetentionPolicyFlag(args.RetentionPolicyArgs),
		toCompressionPolicyFlag(args.CompressionPolicyArgs),
	)
}

// toRetentionPolicyFlag converts the retention policy arguments to the retention policy flag.
func toRetentionPolicyFlag(args *RetentionPolicyArgs) flagpolicy.BackupRetentionPolicy {
	return flagpolicy.BackupRetentionPolicy{
		KeepLatest:  flagpolicy.KeepLatest(args.KeepLatest),
		KeepHourly:  flagpolicy.KeepHourly(args.KeepHourly),
		KeepDaily:   flagpolicy.KeepDaily(args.KeepDaily),
		KeepWeekly:  flagpolicy.KeepWeekly(args.KeepWeekly),
		KeepMonthly: flagpolicy.KeepMonthly(args.KeepMonthly),
		KeepAnnual:  flagpolicy.KeepAnnual(args.KeepAnnual),
	}
}

// toCompressionPolicyFlag converts the compression policy arguments to the compression policy flag.
func toCompressionPolicyFlag(args *CompressionPolicyArgs) flagpolicy.CompressionAlgorithm {
	return flagpolicy.CompressionAlgorithm{
		CompressionAlgorithm: args.CompressionAlgorithm,
	}
}
