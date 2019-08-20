package kando

import (
	"time"

	"github.com/kanisterio/kanister/pkg/chronicle"
	"github.com/spf13/cobra"
)

const (
	profilePathFlagName  = "profile-path"
	artifactPathFlagName = "artifact-path"
	frequencyFlagName    = "frequency"
	envDirFlagName       = "env-dir"
)

func newChroniclePushCommand() *cobra.Command {
	params := chronicle.PushParams{}
	cmd := &cobra.Command{
		Use:   "push <command>",
		Short: "Periodically push the output of a command to object storage",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := params.Validate(); err != nil {
				return err
			}
			params.Command = args
			return chronicle.Push(params)
		},
	}
	cmd.PersistentFlags().StringVarP(&params.ProfilePath, profilePathFlagName, "p", "", "Path to a Profile as a JSON string (required)")
	cmd.MarkPersistentFlagRequired(profilePathFlagName)
	cmd.PersistentFlags().StringVarP(&params.ArtifactFile, artifactPathFlagName, "s", "", "Specify a file that contains an object store suffix (optional)")
	cmd.MarkPersistentFlagRequired(artifactPathFlagName)
	cmd.PersistentFlags().StringVarP(&params.EnvDir, envDirFlagName, "e", "", "Get environment variables from a envdir style directory(optional)")
	cmd.PersistentFlags().DurationVarP(&params.Frequency, frequencyFlagName, "f", time.Minute, "The Frequency to push to object storage ")
	return cmd
}
