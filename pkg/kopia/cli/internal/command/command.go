package command

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
	"github.com/kanisterio/kanister/pkg/safecli"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
)

// CommandApplier is an interface alias for applying commands to a CLI.
type CommandApplier = internal.Applier

// Command is a CLI command/subcommand.
type Command string

// Apply applies the command to the CLI.
func (c Command) Apply(cli safecli.CommandAppender) error {
	cli.AppendLoggable(string(c))
	return nil
}

// KopiaBinaryName is the name of the Kopia binary.
const (
	KopiaBinaryName = Command("kopia")
)

// Repository commands.
const (
	Repository    = Command("repository")
	Create        = Command("create")
	Connect       = Command("connect")
	Server        = Command("server")
	Status        = Command("status")
	SetParameters = Command("set-parameters")
)

// Repository storage sub commands.
const (
	S3         = Command("s3")
	GCS        = Command("gcs")
	Azure      = Command("azure")
	FileSystem = Command("filesystem")
)

// Blob commands.
const (
	Blob  = Command("blob")
	List  = Command("list")
	Stats = Command("stats")
)

// Maintenance commands.
const (
	Maintenance = Command("maintenance")
	Info        = Command("info")
	Set         = Command("set")
	Run         = Command("run")
)

// Policy commands.
const (
	Policy = Command("policy")
	Show   = Command("show")
	_      = Set
)

// Restore commands.
const (
	Restore = Command("restore")
)

// Snapshot commands.
const (
	Snapshot = Command("snapshot")
	_        = Create
	_        = Restore
	_        = List
	Delete   = Command("delete")
	Expire   = Command("expire")
)

// Manifest commands.
const (
	Manifest = Command("manifest")
)

// Server commands.
const (
	_       = Server
	Start   = Command("start")
	Stop    = Command("stop")
	Refresh = Command("refresh")
	_       = Status
	User    = Command("user")
	_       = List
	Add     = Command("add")
)

// NewKopiaCommandBuilder returns a new Kopia command builder.
func NewKopiaCommandBuilder(args cli.CommonArgs, flags ...flag.Applier) (*safecli.Builder, error) {
	flags = append([]flag.Applier{flagcommon.Common(args)}, flags...)
	return NewCommandBuilder(KopiaBinaryName, flags...)
}

// NewCommandBuilder returns a new safecli.Builder for the storage sub command.
func NewCommandBuilder(cmd flag.Applier, flags ...flag.Applier) (*safecli.Builder, error) {
	b := safecli.NewBuilder()
	if err := cmd.Apply(b); err != nil {
		return nil, err
	}
	if err := flag.Apply(b, flags...); err != nil {
		return nil, err
	}
	return b, nil
}
