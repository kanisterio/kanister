package kando

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/param"
)

func newLocationPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source>",
		Short: "Push a source file or stdin stream to s3-compliant object storage",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationPush(c, args)
		},
	}
	return cmd

}

func runLocationPush(cmd *cobra.Command, args []string) error {
	source := args[0]
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	return locationPush(ctx, p, s, source)
}

// TODO: Implement this function
func locationPush(ctx context.Context, p *param.Profile, path string, source string) error {
	return nil
}
