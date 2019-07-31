package kando

import (
	"github.com/spf13/cobra"
)

func newChronicleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chronicle <command>",
		Short: "Manage periodic output streams in object storage",
	}
	cmd.AddCommand(newChroniclePushCommand())
	return cmd
}
