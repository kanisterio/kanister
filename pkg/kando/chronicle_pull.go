package kando

import (
	"context"
	"encoding/json"

	"github.com/kanisterio/kanister/pkg/chronicle"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newChroniclePullCommand() *cobra.Command {
	params := locationParams{}
	cmd := &cobra.Command{
		Use:   "push <command>",
		Short: "Periodically push the output of a command to object storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runChroniclePull(c, params, args[0])
		},
	}
	cmd.PersistentFlags().StringVarP(&params.suffix, pathFlagName, "s", "", "Specify a path suffix (optional)")
	cmd.PersistentFlags().StringVarP(&params.profile, pathFlagName, "p", "", "Pass a Profile as a JSON string (required)")
	cmd.MarkPersistentFlagRequired(profileFlagName)
	return cmd
}

type locationParams struct {
	suffix  string
	profile string
}

func unmarshalProfile(prof string) (*param.Profile, error) {
	p := &param.Profile{}
	err := json.Unmarshal([]byte(prof), p)
	return p, errors.Wrap(err, "failed to unmarshal profile")
}

func runChroniclePull(cmd *cobra.Command, p locationParams, arg string) error {
	target, err := targetWriter(arg)
	if err != nil {
		return err
	}
	prof, err := unmarshalProfile(p.profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return chronicle.Pull(ctx, target, *prof, p.suffix)
}
