package kando

import (
	"context"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/location"
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
	source, err := sourceReader(args[0])
	if err != nil {
		return err
	}
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	return locationPush(ctx, p, s, source)
}

const usePipeParam = `-`

func sourceReader(source string) (io.Reader, error) {
	if source != usePipeParam {
		return os.Open(source)
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		errors.Wrap(err, "Failed to Stat stdin")
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("Stdin must be piped when the source parameter is \"-\"")
	}
	return os.Stdin, nil
}

func locationPush(ctx context.Context, p *param.Profile, path string, source io.Reader) error {
	return location.Write(ctx, source, *p, path)
}
