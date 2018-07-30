package kanctl

import (
	"github.com/spf13/cobra"
)

const (
	dryRunFlag         = "dry-run"
	skipValidationFlag = "skip-validation"
)

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom kanister resource",
	}
	cmd.AddCommand(newActionSetCmd())
	cmd.AddCommand(newProfileCommand())
	cmd.PersistentFlags().Bool(dryRunFlag, false, "if set, resource YAML will be printed but not created")
	cmd.PersistentFlags().Bool(skipValidationFlag, false, "if set, resource is not validated before creation")
	return cmd
}
