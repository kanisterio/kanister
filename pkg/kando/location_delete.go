package kando

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func newLocationDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete artifacts from s3-compliant object storage",
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationDelete(c)
		},
	}
	return cmd

}

func runLocationDelete(cmd *cobra.Command) error {
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	s := pathFlag(cmd)
	ctx := context.Background()
	return locationDelete(ctx, p, s)
}

func locationDelete(ctx context.Context, p *param.Profile, path string) error {
	return location.Delete(ctx, *p, path)
}
