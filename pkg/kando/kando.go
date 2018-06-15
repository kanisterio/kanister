package kando

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kanister "github.com/kanisterio/kanister/pkg"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		log.Errorf("%+v", err)
	}
}

func newRootCommand() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     "kando <command>",
		Short:   "A set of tools used from Kanister Blueprints",
		Version: kanister.VERSION,
	}
	rootCmd.AddCommand(newLocationCommand())
	return rootCmd
}
