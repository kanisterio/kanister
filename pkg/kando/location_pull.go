package kando

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/location"
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
	target, err := targetWriter(args[0])
	if err != nil {
		return err
	}
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	return locationPull(ctx, p, s, target)
}

func targetWriter(target string) (io.Writer, error) {
	if target != usePipeParam {
		return os.Open(target)
	}
	return os.Stdout, nil
}

func locationPull(ctx context.Context, p *param.Profile, path string, target io.Writer) error {
	return location.Read(ctx, target, *p, path)
}
