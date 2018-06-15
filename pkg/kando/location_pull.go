package kando

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/param"
)

func newLocationPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <target>",
		Short: "Pull from s3-compliant object storage to a file or stdout",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationPull(c, args)
		},
	}
	return cmd

}

func runLocationPull(cmd *cobra.Command, args []string) error {
	source := args[0]
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	return locationPull(ctx, p, s, source)
}

// TODO: Implement this function
func locationPull(ctx context.Context, p *param.Profile, path string, source string) error {
	return nil
}
